package entity

import (
	"time"
)

type WishlistItem struct {
	ID        string    `json:"id" firestore:"id"`
	UserID    string    `json:"user_id" firestore:"userId"`
	ProductID string    `json:"product_id" firestore:"productId"`
	CreatedAt time.Time `json:"created_at" firestore:"createdAt"`
}

type WishlistItemWithProduct struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ProductID string    `json:"product_id"`
	Product   *Product  `json:"product"`
	CreatedAt time.Time `json:"created_at"`
}