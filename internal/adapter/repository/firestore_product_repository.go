package repository

import (
	"context"
	"log"
	"slices"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
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
    // Generate ID jika belum ada
    if product.ID == "" {
        product.ID = uuid.New().String()
    }
    
    // Set timestamps
    now := time.Now()
    product.CreatedAt = now
    product.UpdatedAt = now
    
    // Inisialisasi bumpedAt ke waktu pembuatan
    product.BumpedAt = now
    
    // Simpan ke Firestore
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

func (r *firestoreProductRepository) List(ctx context.Context, filter map[string]interface{}, sortType string, limit, offset int) ([]*entity.Product, int64, error) {
    log.Printf("Listing products with filter: %v, sort: %s", filter, sortType)
    
    // Base query
    collection := r.client.Collection("products")
    var query firestore.Query = collection.Query
    
    // Apply filters
    for key, value := range filter {
        query = query.Where(key, "==", value)
    }
    
    // Don't exclude deleted for debug
    // query = query.Where("deletedAt", "==", nil)
    
    // Get all documents first
    docs, err := query.Documents(ctx).GetAll()
    if err != nil {
        log.Printf("Error getting products: %v", err)
        return nil, 0, errors.Internal("Failed to get products", err)
    }
    
    log.Printf("Found %d documents in Firestore", len(docs))
    
    // Parse products dari documents
    var allProducts []*entity.Product
    for _, doc := range docs {
        var product entity.Product
        if err := doc.DataTo(&product); err != nil {
            log.Printf("Error parsing product: %v", err)
            continue // Skip products that fail to parse
        }
        
        // Ensure ID is set
        product.ID = doc.Ref.ID
        
        // Set bumpedAt if not exists
        if product.BumpedAt.IsZero() {
            product.BumpedAt = product.CreatedAt
        }
        
        allProducts = append(allProducts, &product)
    }

    if sortType == "price_asc" {
        // Sort by price ascending
        slices.SortFunc(allProducts, func(a, b *entity.Product) int {
            if a.Price < b.Price {
                return -1
            } else if a.Price > b.Price {
                return 1
            }
            return 0
        })
    } else if sortType == "price_desc" {
        // Sort by price descending
        slices.SortFunc(allProducts, func(a, b *entity.Product) int {
            if a.Price > b.Price {
                return -1
            } else if a.Price < b.Price {
                return 1
            }
            return 0
        })
    } else {
        // Default to bumpedAt desc
        slices.SortFunc(allProducts, func(a, b *entity.Product) int {
            if a.BumpedAt.After(b.BumpedAt) {
                return -1
            } else if a.BumpedAt.Before(b.BumpedAt) {
                return 1
            }
            return 0
        })
    }
    
    // Calculate total
    total := int64(len(allProducts))
    
    // Manual pagination
    var paginatedProducts []*entity.Product
    start := offset
    end := offset + limit
    
    if start < len(allProducts) {
        if end > len(allProducts) {
            end = len(allProducts)
        }
        paginatedProducts = allProducts[start:end]
    }
    
    return paginatedProducts, total, nil
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