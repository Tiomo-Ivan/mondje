package services

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"

	"mondje-backend/config"
)

func ExportTransactionsCSV(ctx context.Context, w io.Writer, agencyName string) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	header := []string{"id", "import_id", "line_number", "agency_name", "type", "amount", "description", "created_at"}
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("erreur écriture en-tête CSV: %w", err)
	}
	csvWriter.Flush()

	query := `
		SELECT id, import_id, line_number, agency_name, type, amount, description, created_at
		FROM transactions
		WHERE 1=1`

	args := []interface{}{}
	if agencyName != "" {
		query += " AND agency_name = $1"
		args = append(args, agencyName)
	}

	query += " ORDER BY created_at ASC, id ASC"

	rows, err := config.DB.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("erreur requête export: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, importID, agency, txType, desc string
		var lineNum int
		var amount float64
		var createdAt time.Time

		err := rows.Scan(&id, &importID, &lineNum, &agency, &txType, &amount, &desc, &createdAt)
		if err != nil {
			return fmt.Errorf("erreur lecture ligne export: %w", err)
		}

		record := []string{
			id,
			importID,
			strconv.Itoa(lineNum),
			agency,
			txType,
			strconv.FormatFloat(amount, 'f', 2, 64),
			desc,
			createdAt.Format(time.RFC3339),
		}

		if err := csvWriter.Write(record); err != nil {
			return fmt.Errorf("erreur écriture ligne CSV: %w", err)
		}
	}

	return rows.Err()
}

type SystemMetrics struct {
	TotalImports      int64   `json:"total_imports"`
	CompletedImports  int64   `json:"completed_imports"`
	FailedImports     int64   `json:"failed_imports"`
	TotalTransactions int64   `json:"total_transactions"`
	TotalErrors       int64   `json:"total_errors"`
	TotalCredit       float64 `json:"total_credit"`
	TotalDebit        float64 `json:"total_debit"`
	NetBalance        float64 `json:"net_balance"`
}

func GetSystemMetrics(ctx context.Context) (*SystemMetrics, error) {
	var m SystemMetrics

	err := config.DB.QueryRow(ctx, `
		SELECT 
			COUNT(*),
			COALESCE(COUNT(*) FILTER (WHERE status = 'COMPLETED'), 0),
			COALESCE(COUNT(*) FILTER (WHERE status = 'FAILED'), 0)
		FROM imports`).Scan(&m.TotalImports, &m.CompletedImports, &m.FailedImports)
	if err != nil {
		return nil, fmt.Errorf("erreur métriques imports: %w", err)
	}

	err = config.DB.QueryRow(ctx, `
		SELECT 
			COUNT(*),
			COALESCE(SUM(CASE WHEN type = 'CREDIT' THEN amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type = 'DEBIT' THEN amount ELSE 0 END), 0)
		FROM transactions`).Scan(&m.TotalTransactions, &m.TotalCredit, &m.TotalDebit)
	if err != nil {
		return nil, fmt.Errorf("erreur métriques transactions: %w", err)
	}

	m.NetBalance = m.TotalCredit - m.TotalDebit

	err = config.DB.QueryRow(ctx, `SELECT COUNT(*) FROM import_errors`).Scan(&m.TotalErrors)
	if err != nil {
		return nil, fmt.Errorf("erreur métriques erreurs: %w", err)
	}

	return &m, nil
}
