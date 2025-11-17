package model

import "time"

type EmailRaw struct {
	ID        int
	UserID    int
	Subject   string
	Body      string
	RawJSON   string
	Status    string
	CreatedAt time.Time
}

type EmailMetadata struct {
	Category   string
	Confidence float64
}

