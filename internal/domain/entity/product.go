package entity

import (
	"time"
)

type ProductImage struct {
	ID           string `json:"id" firestore:"id"`
	URL          string `json:"url" firestore:"url"`
	DisplayOrder int    `json:"display_order" firestore:"displayOrder"`
}

type Product struct {
	ID          string                 `json:"id" firestore:"id"`
	GameTitleID string                 `json:"game_title_id" firestore:"gameTitleId"`
	SellerID    string                 `json:"seller_id" firestore:"sellerId"`
	Title       string                 `json:"title" firestore:"title"`
	Description string                 `json:"description" firestore:"description"`
	Price       float64                `json:"price" firestore:"price"`
	Type        string                 `json:"type" firestore:"type"`
	Attributes  map[string]interface{} `json:"attributes" firestore:"attributes"`
	Images      []ProductImage         `json:"images" firestore:"images"`
	Status      string                 `json:"status" firestore:"status"`
	Stock       int                    `json:"stock" firestore:"stock"`
	SoldCount   int                    `json:"sold_count" firestore:"soldCount"`

	DeliveryMethod       string                 `json:"delivery_method" firestore:"deliveryMethod"`
	Credentials          map[string]interface{} `json:"credentials,omitempty" firestore:"credentials,omitempty"`
	CredentialsValidated bool                   `json:"credentials_validated" firestore:"credentialsValidated"`

	Views     int        `json:"views" firestore:"views"`
	Featured  bool       `json:"featured" firestore:"featured"`
	CreatedAt time.Time  `json:"created_at" firestore:"createdAt"`
	UpdatedAt time.Time  `json:"updated_at" firestore:"updatedAt"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" firestore:"deletedAt,omitempty"`
	BumpedAt  time.Time  `json:"bumped_at" firestore:"bumpedAt"`
}
