package entity

import (
	"time"
)

type GameTitleAttribute struct {
	Name        string   `json:"name" firestore:"name"`
	Type        string   `json:"type" firestore:"type"`
	Required    bool     `json:"required" firestore:"required"`
	Options     []string `json:"options,omitempty" firestore:"options,omitempty"`
	Description string   `json:"description,omitempty" firestore:"description,omitempty"`
}

type GameTitle struct {
	ID          string               `json:"id" firestore:"id"`
	Name        string               `json:"name" firestore:"name"`
	Slug        string               `json:"slug" firestore:"slug"`
	Description string               `json:"description" firestore:"description"`
	Icon        string               `json:"icon,omitempty" firestore:"icon,omitempty"`
	Banner      string               `json:"banner,omitempty" firestore:"banner,omitempty"`
	Attributes  []GameTitleAttribute `json:"attributes" firestore:"attributes"`
	Status      string               `json:"status" firestore:"status"`
	CreatedAt   time.Time            `json:"created_at" firestore:"createdAt"`
	UpdatedAt   time.Time            `json:"updated_at" firestore:"updatedAt"`
}
