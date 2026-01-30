package user

import (
	"time"
)

type User struct {
	ID           string    `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Name         string    `json:"name" db:"name"`
	Role         string    `json:"role" db:"role"`
	APIKey       string    `json:"apiKey,omitempty" db:"api_key"` // ChatGPT API Key
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type ProfileUpdateRequest struct {
	Name   string `json:"name"`
	APIKey string `json:"apiKey"`
}
