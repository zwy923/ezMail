package model

import "time"

type User struct {
	ID           int
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

