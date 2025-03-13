package models

import "time"

// User domain model for user info
type User struct {
	UUID         string    `json:"uuid"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"passwordHash"`
	PasswordSalt string    `json:"passwordSalt"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	IsActive     bool      `json:"is_active"`
}
