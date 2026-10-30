package services

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"mondje-backend/config"
	"mondje-backend/models"

	"github.com/google/uuid"
)

func ProcessCSVStream(ctx context.Context, r io.Reader, fileName string, agencyName string) (*models.Import, error) {
	hasher := sha256.New()

	var readStream io.Reader = r
	if seeker, ok := r.(io.ReadSeeker); ok {
		if _, err := io.Copy(hasher, seeker); err == nil {
			seeker.Seek(0, io.SeekStart)
		}
	} else {
		readStream = io.TeeReader(r, hasher)
	}

	hashStr := hex.EncodeToString(hasher.Sum(nil))

	// Vérification de l'idempotence (Défi 8)
	if _, ok := r.(io.ReadSeeker); ok {
		existing, err := checkExistingHash(ctx, hashStr)
		if err == nil && existing != nil {
			return existing, nil
		}
	}

	imp := &models.Import{
		ID:         uuid.New(),
		FileName:   fileName,
		AgencyName: agencyName,
		FileHash:   hashStr,
		Status:     models.ImportStatus("IN_PROGRESS"),
		TotalLines: 0,
		CreatedAt:  time.Now(),
	}

	if err := saveImportRecord(ctx, imp); err != nil {
		return nil, fmt.Errorf("erreur enregistrement import: %w", err)
	}

	scanner := bufio.NewScanner(readStream)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var validTransactions []models.Transaction
	var importErrors []models.ImportError

	lineNumber := 0
	validCount := 0
	batchSize := 1000

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// Ignorer la ligne d'en-tête (Header)
		if lineNumber == 1 {
			lowerLine := strings.ToLower(line)
			if strings.Contains(lowerLine, "type") && (strings.Contains(lowerLine, "montant") || strings.Contains(lowerLine, "amount")) {
				continue
			}
		}

		tx, errReason := parseLine(line, lineNumber, imp.ID, agencyName)
		if errReason != "" {
			importErrors = append(importErrors, models.ImportError{
				ID:         uuid.New(),
				ImportID:   imp.ID,
				LineNumber: lineNumber,
				RawContent: line,
				Reason:     errReason,
				CreatedAt:  time.Now(),
			})
		} else if tx != nil {
			validTransactions = append(validTransactions, *tx)
			validCount++
		}

		// Traitement par batch pour garder une empreinte mémoire faible (< 256 Mo RAM)
		if len(validTransactions) >= batchSize {
			if err := insertBatch(ctx, validTransactions); err != nil {
				updateImportStatus(ctx, imp.ID, models.ImportStatus("FAILED"), validCount)
				return nil, fmt.Errorf("erreur insertion batch: %w", err)
			}
			validTransactions = validTransactions[:0]
		}

		if len(importErrors) >= batchSize {
			insertErrorBatch(ctx, importErrors)
			importErrors = importErrors[:0]
		}
	}

	if err := scanner.Err(); err != nil {
		updateImportStatus(ctx, imp.ID, models.ImportStatus("FAILED"), validCount)
		return nil, fmt.Errorf("erreur lecture du flux CSV: %w", err)
	}

	if len(validTransactions) > 0 {
		if err := insertBatch(ctx, validTransactions); err != nil {
			updateImportStatus(ctx, imp.ID, models.ImportStatus("FAILED"), validCount)
			return nil, fmt.Errorf("erreur insertion dernier batch: %w", err)
		}
	}

	if len(importErrors) > 0 {
		insertErrorBatch(ctx, importErrors)
	}

	if _, ok := r.(io.ReadSeeker); !ok {
		finalHash := hex.EncodeToString(hasher.Sum(nil))
		imp.FileHash = finalHash
		config.DB.Exec(ctx, `UPDATE imports SET file_hash = $1 WHERE id = $2`, finalHash, imp.ID)
	}

	updateImportStatus(ctx, imp.ID, models.ImportStatus("COMPLETED"), validCount)
	imp.Status = models.ImportStatus("COMPLETED")
	imp.TotalLines = validCount

	return imp, nil
}

func parseLine(line string, lineNum int, importID uuid.UUID, agencyName string) (*models.Transaction, string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, "Ligne vide"
	}

	var parts []string
	if strings.Contains(line, ";") {
		parts = strings.Split(line, ";")
	} else {
		parts = strings.Split(line, ",")
	}

	if len(parts) < 3 {
		return nil, fmt.Sprintf("Format de ligne invalide (3 colonnes attendues, %d trouvées)", len(parts))
	}

	txType := strings.ToUpper(strings.TrimSpace(parts[0]))
	if txType != "CREDIT" && txType != "DEBIT" {
		return nil, fmt.Sprintf("Type de transaction invalide: '%s' (doit être CREDIT ou DEBIT)", parts[0])
	}

	amountStr := strings.TrimSpace(parts[1])
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return nil, fmt.Sprintf("Montant invalide: '%s'", parts[1])
	}

	if amount <= 0 {
		return nil, fmt.Sprintf("Montant négatif ou nul non autorisé: %f", amount)
	}

	descStr := strings.TrimSpace(parts[2])

	return &models.Transaction{
		ID:          uuid.New(),
		ImportID:    importID,
		LineNumber:  lineNum,
		AgencyName:  agencyName,
		Type:        txType,
		Amount:      amount,
		Description: descStr,
		CreatedAt:   time.Now(),
	}, ""
}

func checkExistingHash(ctx context.Context, hash string) (*models.Import, error) {
	query := `SELECT id, file_name, agency_name, status, total_lines, created_at FROM imports WHERE file_hash = $1 AND status IN ('COMPLETED', 'IN_PROGRESS') LIMIT 1`
	row := config.DB.QueryRow(ctx, query, hash)

	var imp models.Import
	err := row.Scan(&imp.ID, &imp.FileName, &imp.AgencyName, &imp.Status, &imp.TotalLines, &imp.CreatedAt)
	if err != nil {
		return nil, nil
	}
	return &imp, nil
}

func saveImportRecord(ctx context.Context, imp *models.Import) error {
	query := `INSERT INTO imports (id, file_name, agency_name, file_hash, status, total_lines, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := config.DB.Exec(ctx, query, imp.ID, imp.FileName, imp.AgencyName, imp.FileHash, imp.Status, imp.TotalLines, imp.CreatedAt)
	return err
}

func updateImportStatus(ctx context.Context, id uuid.UUID, status models.ImportStatus, totalLines int) {
	query := `UPDATE imports SET status = $1, total_lines = $2 WHERE id = $3`
	config.DB.Exec(ctx, query, status, totalLines, id)
}

func insertBatch(ctx context.Context, batch []models.Transaction) error {
	tx, err := config.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `INSERT INTO transactions (
		id, import_id, line_number, file_line_number, agency_name, type, amount, montant, description, created_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	for _, t := range batch {
		_, err := tx.Exec(ctx, query,
			t.ID,          // $1
			t.ImportID,    // $2
			t.LineNumber,  // $3
			t.LineNumber,  // $4 (file_line_number)
			t.AgencyName,  // $5
			t.Type,        // $6
			t.Amount,      // $7
			t.Amount,      // $8 (montant)
			t.Description, // $9
			t.CreatedAt,   // $10
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func insertErrorBatch(ctx context.Context, errors []models.ImportError) {
	for _, e := range errors {
		query := `INSERT INTO import_errors (id, import_id, line_number, raw_content, reason, created_at) VALUES ($1, $2, $3, $4, $5, $6)`
		config.DB.Exec(ctx, query, e.ID, e.ImportID, e.LineNumber, e.RawContent, e.Reason, e.CreatedAt)
	}
}

func DeleteImportRecord(ctx context.Context, importID uuid.UUID) error {
	tx, err := config.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("erreur début transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Supprimer les transactions liées
	_, err = tx.Exec(ctx, `DELETE FROM transactions WHERE import_id = $1`, importID)
	if err != nil {
		return fmt.Errorf("erreur suppression transactions: %w", err)
	}

	// 2. Supprimer les erreurs d'import liées
	_, err = tx.Exec(ctx, `DELETE FROM import_errors WHERE import_id = $1`, importID)
	if err != nil {
		return fmt.Errorf("erreur suppression erreurs d'import: %w", err)
	}

	// 3. Supprimer le registre d'importation
	res, err := tx.Exec(ctx, `DELETE FROM imports WHERE id = $1`, importID)
	if err != nil {
		return fmt.Errorf("erreur suppression enregistrement import: %w", err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("IMPORT_NOT_FOUND: aucun import trouvé avec cet ID")
	}

	return tx.Commit(ctx)
}
