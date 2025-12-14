package entity

import (
	"time"
)

type User struct {
	ID       string `json:"id" firestore:"id"`
	Email    string `json:"email" firestore:"email"`
	Username string `json:"username" firestore:"username"`
	Phone    string `json:"phone" firestore:"phone"`
	Bio      string `json:"bio" firestore:"bio"`
	Role     string `json:"role" firestore:"role"`
	Status   string `json:"status" firestore:"status"`

	FullName           string    `json:"full_name,omitempty" firestore:"fullName,omitempty"`
	Address            string    `json:"address,omitempty" firestore:"address,omitempty"`
	DateOfBirth        time.Time `json:"date_of_birth,omitempty" firestore:"dateOfBirth,omitempty"`
	IdNumber           string    `json:"id_number,omitempty" firestore:"idNumber,omitempty"`
	IdCardImage        string    `json:"id_card_image,omitempty" firestore:"idCardImage,omitempty"`
	VerificationStatus string    `json:"verification_status" firestore:"verificationStatus"`

	SellerRating      float64 `json:"seller_rating,omitempty" firestore:"sellerRating,omitempty"`
	SellerReviewCount int     `json:"seller_review_count,omitempty" firestore:"sellerReviewCount,omitempty"`
	BuyerRating       float64 `json:"buyer_rating,omitempty" firestore:"buyerRating,omitempty"`
	BuyerReviewCount  int     `json:"buyer_review_count,omitempty" firestore:"buyerReviewCount,omitempty"`

	// Online presence and profile fields
	AvatarURL    string    `json:"avatar_url,omitempty" firestore:"avatarURL,omitempty"`
	PhotoURL     string    `json:"photo_url,omitempty" firestore:"photoURL,omitempty"`
	LastSeen     time.Time `json:"last_seen" firestore:"lastSeen"`
	OnlineStatus string    `json:"online_status" firestore:"onlineStatus"`
	Provider     string    `json:"provider,omitempty" firestore:"provider,omitempty"`

	CreatedAt time.Time `json:"created_at" firestore:"createdAt"`
	UpdatedAt time.Time `json:"updated_at" firestore:"updatedAt"`
}
