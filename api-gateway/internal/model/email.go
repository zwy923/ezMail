package model

import "time"

type EmailWithMetadata struct {
	ID        int
	Subject   string
	Body      string
	Status    string
	CreatedAt time.Time
	Metadata  *EmailMetadata
}

type EmailMetadata struct {
	Category   string
	Confidence float64
}

