package entity

import (
	"time"
)

type User struct {
	ID          string    `json:"id" firestore:"id"`
	Email       string    `json:"email" firestore:"email"`
	Username    string    `json:"username" firestore:"username"`
	Phone       string    `json:"phone" firestore:"phone"`
	Bio         string    `json:"bio" firestore:"bio"`
	Role        string    `json:"role" firestore:"role"`
	Status      string    `json:"status" firestore:"status"`
	CreatedAt   time.Time `json:"created_at" firestore:"createdAt"`
	UpdatedAt   time.Time `json:"updated_at" firestore:"updatedAt"`
} 