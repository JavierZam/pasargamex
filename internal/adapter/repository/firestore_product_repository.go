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

	if product.ID == "" {
		product.ID = uuid.New().String()
	}

	now := time.Now()
	product.CreatedAt = now
	product.UpdatedAt = now

	product.BumpedAt = now

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

	var minPrice, maxPrice float64
	if minPriceVal, ok := filter["min_price"]; ok {
		minPrice = minPriceVal.(float64)
		delete(filter, "min_price")
	}
	if maxPriceVal, ok := filter["max_price"]; ok {
		maxPrice = maxPriceVal.(float64)
		delete(filter, "max_price")
	}

	collection := r.client.Collection("products")
	var query firestore.Query = collection.Query

	for key, value := range filter {
		query = query.Where(key, "==", value)
	}

	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		log.Printf("Error getting products: %v", err)
		return nil, 0, errors.Internal("Failed to get products", err)
	}

	log.Printf("Found %d documents in Firestore", len(docs))

	var allProducts []*entity.Product
	for _, doc := range docs {
		var product entity.Product
		if err := doc.DataTo(&product); err != nil {
			log.Printf("Error parsing product: %v", err)
			continue
		}

		if (minPrice > 0 && product.Price < minPrice) ||
			(maxPrice > 0 && product.Price > maxPrice) {

			log.Printf("Skipping product %s with price %.2f (outside range %.2f-%.2f)",
				product.ID, product.Price, minPrice, maxPrice)
			continue
		}

		product.ID = doc.Ref.ID

		if product.BumpedAt.IsZero() {
			product.BumpedAt = product.CreatedAt
		}

		allProducts = append(allProducts, &product)
	}

	log.Printf("After price filtering: %d products remain", len(allProducts))

	if sortType == "price_asc" {

		slices.SortFunc(allProducts, func(a, b *entity.Product) int {
			if a.Price < b.Price {
				return -1
			} else if a.Price > b.Price {
				return 1
			}
			return 0
		})
	} else if sortType == "price_desc" {

		slices.SortFunc(allProducts, func(a, b *entity.Product) int {
			if a.Price > b.Price {
				return -1
			} else if a.Price < b.Price {
				return 1
			}
			return 0
		})
	} else {

		slices.SortFunc(allProducts, func(a, b *entity.Product) int {
			if a.BumpedAt.After(b.BumpedAt) {
				return -1
			} else if a.BumpedAt.Before(b.BumpedAt) {
				return 1
			}
			return 0
		})
	}

	total := int64(len(allProducts))

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

func (r *firestoreProductRepository) Search(ctx context.Context, query string, filter map[string]interface{}, limit, offset int) ([]*entity.Product, int64, error) {
	log.Printf("=== Search repository called with query: '%s' ===", query)
	log.Printf("Filter: %v", filter)

	var minPrice, maxPrice float64
	if minPriceVal, ok := filter["min_price"]; ok {
		minPrice = minPriceVal.(float64)
		log.Printf("Filtering with min_price: %.2f", minPrice)
		delete(filter, "min_price")
	}
	if maxPriceVal, ok := filter["max_price"]; ok {
		maxPrice = maxPriceVal.(float64)
		log.Printf("Filtering with max_price: %.2f", maxPrice)
		delete(filter, "max_price")
	}

	collection := r.client.Collection("products")
	docs, err := collection.Documents(ctx).GetAll()
	if err != nil {
		log.Printf("Error getting products: %v", err)
		return nil, 0, errors.Internal("Failed to search products", err)
	}

	log.Printf("Found %d raw documents in collection", len(docs))

	var allProducts []*entity.Product
	for _, doc := range docs {
		var product entity.Product
		if err := doc.DataTo(&product); err != nil {
			log.Printf("Error parsing product: %v", err)
			continue
		}
		product.ID = doc.Ref.ID
		allProducts = append(allProducts, &product)
	}

	log.Printf("Successfully parsed %d products", len(allProducts))

	var filteredProducts []*entity.Product
	searchTerms := strings.ToLower(query)

	for _, product := range allProducts {
		log.Printf("Checking product %s: Title='%s', Price=%.2f", product.ID, product.Title, product.Price)

		if minPrice > 0 {
			if product.Price < minPrice {
				log.Printf("❌ Skipping product %s: price %.2f < min_price %.2f",
					product.ID, product.Price, minPrice)
				continue
			}
		}

		if maxPrice > 0 {
			if product.Price > maxPrice {
				log.Printf("❌ Skipping product %s: price %.2f > max_price %.2f",
					product.ID, product.Price, maxPrice)
				continue
			}
		}

		skipProduct := false
		for key, value := range filter {
			if key == "gameTitleId" && value != product.GameTitleID {
				skipProduct = true
				break
			}
			if key == "type" && value != product.Type {
				skipProduct = true
				break
			}
			if key == "status" && value != product.Status {
				skipProduct = true
				break
			}
		}

		if skipProduct {
			log.Printf("Skipping product %s due to filter mismatch", product.ID)
			continue
		}

		titleLower := strings.ToLower(product.Title)
		descLower := strings.ToLower(product.Description)

		if strings.Contains(titleLower, searchTerms) || strings.Contains(descLower, searchTerms) {
			log.Printf("✅ MATCH Product %s matches search terms", product.ID)
			filteredProducts = append(filteredProducts, product)
		} else {
			log.Printf("❌ NO MATCH Product %s does not match search terms", product.ID)
		}
	}

	log.Printf("After filtering: %d products match criteria", len(filteredProducts))

	if len(filteredProducts) > 0 {
		slices.SortFunc(filteredProducts, func(a, b *entity.Product) int {
			if a.BumpedAt.After(b.BumpedAt) {
				return -1
			} else if a.BumpedAt.Before(b.BumpedAt) {
				return 1
			}
			return 0
		})
	}

	var paginatedProducts []*entity.Product
	start := offset
	end := offset + limit

	if len(filteredProducts) > 0 && start < len(filteredProducts) {
		if end > len(filteredProducts) {
			end = len(filteredProducts)
		}
		paginatedProducts = filteredProducts[start:end]
	} else {
		paginatedProducts = []*entity.Product{}
	}

	return paginatedProducts, int64(len(filteredProducts)), nil
}

func (r *firestoreProductRepository) ListBySellerID(ctx context.Context, sellerID string, status string, limit, offset int) ([]*entity.Product, int64, error) {
	query := r.client.Collection("products").Query.Where("sellerId", "==", sellerID).Where("deletedAt", "==", nil)

	if status != "" {
		query = query.Where("status", "==", status)
	}

	allDocs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, 0, errors.Internal("Failed to count seller products", err)
	}
	total := int64(len(allDocs))

	query = query.OrderBy("createdAt", firestore.Desc)
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

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
