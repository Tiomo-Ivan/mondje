package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mondje-backend/config"
	"mondje-backend/models"

	"github.com/google/uuid"
)

type TransactionFilter struct {
	AgencyName string
	Type       string
	Search     string
	Cursor     string // Format: "timestamp,uuid"
	Limit      int
}

type PaginatedResponse struct {
	Data       []models.Transaction `json:"data"`
	NextCursor string               `json:"next_cursor,omitempty"`
	HasMore    bool                 `json:"has_more"`
}

func GetTransactions(ctx context.Context, filter TransactionFilter) (*PaginatedResponse, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}

	query := `
		SELECT id, import_id, line_number, agency_name, type, amount, description, created_at
		FROM transactions
		WHERE 1=1`

	args := []interface{}{}
	argID := 1

	if filter.AgencyName != "" {
		query += fmt.Sprintf(" AND agency_name = $%d", argID)
		args = append(args, filter.AgencyName)
		argID++
	}

	if filter.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", argID)
		args = append(args, strings.ToUpper(filter.Type))
		argID++
	}

	if filter.Search != "" {
		query += fmt.Sprintf(" AND (description ILIKE $%d OR agency_name ILIKE $%d)", argID, argID)
		args = append(args, "%"+filter.Search+"%")
		argID++
	}

	// Pagination par curseur (Timestamp + UUID pour désamboguer)
	if filter.Cursor != "" {
		parts := strings.Split(filter.Cursor, ",")
		if len(parts) == 2 {
			cursorTime, errTime := time.Parse(time.RFC3339Nano, parts[0])
			cursorID, errUUID := uuid.Parse(parts[1])
			if errTime == nil && errUUID == nil {
				query += fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", argID, argID+1)
				args = append(args, cursorTime, cursorID)
				argID += 2
			}
		}
	}

	// Récupérer N + 1 éléments pour déterminer s'il y a une page suivante
	query += fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", argID)
	args = append(args, filter.Limit+1)

	rows, err := config.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("erreur requête transactions: %w", err)
	}
	defer rows.Close()

	var transactions []models.Transaction
	for rows.Next() {
		var t models.Transaction
		err := rows.Scan(&t.ID, &t.ImportID, &t.LineNumber, &t.AgencyName, &t.Type, &t.Amount, &t.Description, &t.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("erreur lecture transaction: %w", err)
		}
		transactions = append(transactions, t)
	}

	hasMore := false
	nextCursor := ""

	if len(transactions) > filter.Limit {
		hasMore = true
		transactions = transactions[:filter.Limit]
		lastItem := transactions[len(transactions)-1]
		nextCursor = fmt.Sprintf("%s,%s", lastItem.CreatedAt.Format(time.RFC3339Nano), lastItem.ID.String())
	}

	if transactions == nil {
		transactions = []models.Transaction{}
	}

	return &PaginatedResponse{
		Data:       transactions,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

type BalanceResponse struct {
	AgencyName   string    `json:"agency_name"`
	TotalCredit  float64   `json:"total_credit"`
	TotalDebit   float64   `json:"total_debit"`
	NetBalance   float64   `json:"net_balance"`
	CalculatedAt time.Time `json:"calculated_at"`
}

func GetAgencyBalance(ctx context.Context, agencyName string) (*BalanceResponse, error) {
	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN type = 'CREDIT' THEN amount ELSE 0 END), 0) AS total_credit,
			COALESCE(SUM(CASE WHEN type = 'DEBIT' THEN amount ELSE 0 END), 0) AS total_debit
		FROM transactions
		WHERE agency_name = $1`

	var totalCredit, totalDebit float64
	err := config.DB.QueryRow(ctx, query, agencyName).Scan(&totalCredit, &totalDebit)
	if err != nil {
		return nil, fmt.Errorf("erreur calcul solde: %w", err)
	}

	return &BalanceResponse{
		AgencyName:   agencyName,
		TotalCredit:  totalCredit,
		TotalDebit:   totalDebit,
		NetBalance:   totalCredit - totalDebit,
		CalculatedAt: time.Now(),
	}, nil
}
