package models

import (
	"time"

	"github.com/google/uuid"
)

type ImportError struct {
	ID         uuid.UUID `json:"id"`
	ImportID   uuid.UUID `json:"import_id"`
	LineNumber int       `json:"line_number"`
	RawContent string    `json:"raw_content"`
	Reason     string    `json:"reason"`
	CreatedAt  time.Time `json:"created_at"`
}