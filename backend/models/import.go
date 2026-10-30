package models

import (
	"time"

	"github.com/google/uuid"
)

type ImportStatus string

const (
	StatusInProgress ImportStatus = "IN_PROGRESS"
	StatusCompleted  ImportStatus = "COMPLETED"
	StatusFailed     ImportStatus = "FAILED"
)

type Import struct {
	ID         uuid.UUID    `json:"id"`
	FileName   string       `json:"file_name"`
	AgencyName string       `json:"agency_name"`
	FileHash   string       `json:"file_hash,omitempty"`
	Status     ImportStatus `json:"status"`
	TotalLines int          `json:"total_lines"`
	CreatedAt  time.Time    `json:"created_at"`
}

