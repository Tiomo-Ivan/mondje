package models

import (
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	ID          uuid.UUID `json:"id"`
	ImportID    uuid.UUID `json:"import_id"`
	LineNumber  int       `json:"line_number"`
	AgencyName  string    `json:"agency_name"`
	Type        string    `json:"type"`
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}
