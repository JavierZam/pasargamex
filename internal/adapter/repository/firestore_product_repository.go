package repository

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/pkg/errors"
)

type firestoreProductRepository struct {
	client *firestore.Client
}

func NewFirestoreProductRepository(client *firestore.Client) repository.ProductRepository {
	return &firestoreProductRepository{
		client: client,
	}
}

func (r *firestoreProductRepository) Create(ctx context.Context, product *entity.Product) error {
	// Generate ID if not provided
	if product.ID == "" {
		doc := r.client.Collection("products").NewDoc()
		product.ID = doc.ID
	}

	// Set timestamps
	now := time.Now()
	if product.CreatedAt.IsZero() {
		product.CreatedAt = now
	}
	product.UpdatedAt = now
    product.BumpedAt = now

	// Save to Firestore
	_, err := r.client.Collection("products").Doc(product.ID).Set(ctx, product)
	if err != nil {
		return errors.Internal("Failed to create product", err)
	}

	return nil
}

func (r *firestoreProductRepository) GetByID(ctx context.Context, id string) (*entity.Product, error) {
	doc, err := r.client.Collection("products").Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, errors.NotFound("Product", err)
		}
		return nil, errors.Internal("Failed to get product", err)
	}

	var product entity.Product
	if err := doc.DataTo(&product); err != nil {
		return nil, errors.Internal("Failed to parse product data", err)
	}

	return &product, nil
}

func (r *firestoreProductRepository) List(ctx context.Context, filter map[string]interface{}, sort string, limit, offset int) ([]*entity.Product, int64, error) {
    // Inisialisasi query
    query := r.client.Collection("products").Query
    
    // Add default filter to exclude soft deleted
    if filter == nil {
        filter = make(map[string]interface{})
    }
   
    // Apply filters
    for key, value := range filter {
        query = query.Where(key, "==", value)
    }
    
    // Apply default filter to exclude deleted products
    query = query.Where("deletedAt", "==", nil)
    
    // Apply sorting
    if sort != "" {
        parts := strings.Split(sort, "_")
        field := parts[0]
        order := firestore.Asc
        if len(parts) > 1 && parts[1] == "desc" {
            order = firestore.Desc
        }
        query = query.OrderBy(field, order)
    } else {
        // Default sort - changed from createdAt to bumpedAt
        query = query.OrderBy("bumpedAt", firestore.Desc)
    }
    
    // Get total count - gunakan query yang sudah dibuat, bukan allQuery baru
    allDocs, err := query.Documents(ctx).GetAll()
    if err != nil {
        return nil, 0, errors.Internal("Failed to count products", err)
    }
    total := int64(len(allDocs))
    
    // Apply pagination
    if limit > 0 {
        query = query.Limit(limit)
    }
    if offset > 0 {
        query = query.Offset(offset)
    }
    
    // Execute query
    iter := query.Documents(ctx)
    var products []*entity.Product
    
    for {
        doc, err := iter.Next()
        if err == iterator.Done {
            break
        }
        if err != nil {
            return nil, 0, errors.Internal("Failed to iterate products", err)
        }
        var product entity.Product
        if err := doc.DataTo(&product); err != nil {
            return nil, 0, errors.Internal("Failed to parse product data", err)
        }
        products = append(products, &product)
    }
    
    return products, total, nil
}

func (r *firestoreProductRepository) Update(ctx context.Context, product *entity.Product) error {
	product.UpdatedAt = time.Now()

	_, err := r.client.Collection("products").Doc(product.ID).Set(ctx, product)
	if err != nil {
		return errors.Internal("Failed to update product", err)
	}

	return nil
}

func (r *firestoreProductRepository) Delete(ctx context.Context, id string) error {
	_, err := r.client.Collection("products").Doc(id).Delete(ctx)
	if err != nil {
		return errors.Internal("Failed to delete product", err)
	}

	return nil
}

func (r *firestoreProductRepository) SoftDelete(ctx context.Context, id string) error {
	now := time.Now()
	_, err := r.client.Collection("products").Doc(id).Update(ctx, []firestore.Update{
		{Path: "deletedAt", Value: now},
		{Path: "status", Value: "deleted"},
		{Path: "updatedAt", Value: now},
	})
	if err != nil {
		return errors.Internal("Failed to soft delete product", err)
	}

	return nil
}

func (r *firestoreProductRepository) IncrementViews(ctx context.Context, id string) error {
	_, err := r.client.Collection("products").Doc(id).Update(ctx, []firestore.Update{
		{Path: "views", Value: firestore.Increment(1)},
		{Path: "updatedAt", Value: time.Now()},
	})
	if err != nil {
		return errors.Internal("Failed to increment product views", err)
	}

	return nil
}

func (r *firestoreProductRepository) SearchByTitle(ctx context.Context, query string, filter map[string]interface{}, sort string, limit, offset int) ([]*entity.Product, int64, error) {
    // Basic search implementation with Firestore
    // In a real application, consider using a dedicated search service like Algolia or Elasticsearch
    query = strings.ToLower(query)
    
    baseQuery := r.client.Collection("products").Query.Where("deletedAt", "==", nil)
    
    // Apply filters
    for key, value := range filter {
        baseQuery = baseQuery.Where(key, "==", value)
    }
    
    // Get all documents (inefficient for large datasets, but Firestore doesn't support full-text search)
    docs, err := baseQuery.Documents(ctx).GetAll()
    if err != nil {
        return nil, 0, errors.Internal("Failed to search products", err)
    }
    
    // Filter manually by title
    var matchedProducts []*entity.Product
    for _, doc := range docs {
        var product entity.Product
        if err := doc.DataTo(&product); err != nil {
            continue
        }
        
        // Simple case-insensitive contains check
        if strings.Contains(strings.ToLower(product.Title), query) {
            matchedProducts = append(matchedProducts, &product)
        }
    }
    
    total := int64(len(matchedProducts))
    
    // Manual sort
    // Implement sorting logic here
    
    // Manual pagination
    start := offset
    end := offset + limit
    if start >= len(matchedProducts) {
        return []*entity.Product{}, total, nil
    }
    if end > len(matchedProducts) {
        end = len(matchedProducts)
    }
    
    return matchedProducts[start:end], total, nil
}

func (r *firestoreProductRepository) ListBySellerID(ctx context.Context, sellerID string, status string, limit, offset int) ([]*entity.Product, int64, error) {
    query := r.client.Collection("products").Query.Where("sellerId", "==", sellerID).Where("deletedAt", "==", nil)
    
    if status != "" {
        query = query.Where("status", "==", status)
    }
    
    // Get total count
    allDocs, err := query.Documents(ctx).GetAll()
    if err != nil {
        return nil, 0, errors.Internal("Failed to count seller products", err)
    }
    total := int64(len(allDocs))
    
    // Apply pagination
    query = query.OrderBy("createdAt", firestore.Desc)
    if limit > 0 {
        query = query.Limit(limit)
    }
    if offset > 0 {
        query = query.Offset(offset)
    }
    
    // Execute query
    iter := query.Documents(ctx)
    var products []*entity.Product
    
    for {
        doc, err := iter.Next()
        if err == iterator.Done {
            break
        }
        if err != nil {
            return nil, 0, errors.Internal("Failed to iterate seller products", err)
        }
        
        var product entity.Product
        if err := doc.DataTo(&product); err != nil {
            return nil, 0, errors.Internal("Failed to parse product data", err)
        }
        products = append(products, &product)
    }
    
    return products, total, nil
}