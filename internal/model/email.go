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

type EmailWithMetadata struct {
	ID        int
	Subject   string
	Body      string
	Status    string
	CreatedAt time.Time
	Metadata  *EmailMetadata
}
