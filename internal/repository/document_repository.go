package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"telis-api-gateway/internal/domain"
)

type documentRepository struct {
	db               *sql.DB
	mu               sync.Mutex
	cachedOptions    *domain.MetadataOptions
	optionsCacheTime time.Time
}

func NewDocumentRepository(db *sql.DB) domain.DocumentRepository {
	return &documentRepository{db: db}
}

func (r *documentRepository) GetAll(ctx context.Context, filter domain.DocumentFilter) ([]domain.Document, int, error) {
	query := `
		WITH RECURSIVE folder_tree AS (
			SELECT id, name, parent_id, name::text as folder_path
			FROM ingestion.folders
			WHERE parent_id IS NULL
			UNION ALL
			SELECT f.id, f.name, f.parent_id, (ft.folder_path || ' > ' || f.name)
			FROM ingestion.folders f
			INNER JOIN folder_tree ft ON f.parent_id = ft.id
		)
		SELECT d.id, d.folder_id, d.filename, d.file_path, d.status, d.uploaded_by,
		       d.file_size_bytes, d.is_deprecated, d.previous_version_id, d.version,
		       d.created_at, d.updated_at, COALESCE(ft.folder_path, '') as folder_path,
		       COALESCE(d.document_type, 'OTHER') as document_type,
		       COALESCE(d.risk_level, 'UNKNOWN') as risk_level,
		       COALESCE(d.risk_reasoning, '') as risk_reasoning,
		       COALESCE(d.vendor_name, '') as vendor_name,
		       COALESCE(d.business_unit, '') as business_unit,
		       d.effective_date, d.expiry_date,
		       COALESCE(d.summary, '') as summary
		FROM ingestion.documents d
		LEFT JOIN folder_tree ft ON d.folder_id = ft.id
		WHERE d.status != 'DELETED'
	`
	countQuery := `SELECT count(*) FROM ingestion.documents d WHERE d.status != 'DELETED'`

	var conditions []string
	var args []interface{}
	argId := 1

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("d.filename ILIKE $%d", argId))
		args = append(args, "%"+filter.Search+"%")
		argId++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("d.status = $%d", argId))
		args = append(args, filter.Status)
		argId++
	}

	if filter.IsDeprecated != nil {
		conditions = append(conditions, fmt.Sprintf("d.is_deprecated = $%d", argId))
		args = append(args, *filter.IsDeprecated)
		argId++
	}

	if filter.FolderID != nil {
		if *filter.FolderID == "null" || *filter.FolderID == "" {
			conditions = append(conditions, "d.folder_id IS NULL")
		} else {
			conditions = append(conditions, fmt.Sprintf("d.folder_id = $%d", argId))
			args = append(args, *filter.FolderID)
			argId++
		}
	}

	// Phase 1 filters
	if filter.DocumentType != "" {
		conditions = append(conditions, fmt.Sprintf("d.document_type = $%d", argId))
		args = append(args, filter.DocumentType)
		argId++
	}

	if filter.RiskLevel != "" {
		conditions = append(conditions, fmt.Sprintf("d.risk_level = $%d", argId))
		args = append(args, filter.RiskLevel)
		argId++
	}

	if filter.VendorName != "" {
		conditions = append(conditions, fmt.Sprintf("d.vendor_name = $%d", argId))
		args = append(args, filter.VendorName)
		argId++
	}

	if filter.BusinessUnit != "" {
		conditions = append(conditions, fmt.Sprintf("d.business_unit = $%d", argId))
		args = append(args, filter.BusinessUnit)
		argId++
	}

	if len(conditions) > 0 {
		condStr := " AND " + strings.Join(conditions, " AND ")
		query += condStr
		countQuery += condStr
	}

	// Get total count
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Add Order and Pagination
	validSortColumns := map[string]string{
		"filename":        "d.filename",
		"created_at":      "d.created_at",
		"file_size_bytes": "d.file_size_bytes",
		"risk_level":      "d.risk_level",
	}

	sortColumn := "d.filename" // Default
	if col, ok := validSortColumns[filter.SortBy]; ok {
		sortColumn = col
	}

	sortOrder := "ASC" // Default
	if strings.ToUpper(filter.SortOrder) == "DESC" {
		sortOrder = "DESC"
	}

	query += fmt.Sprintf(" ORDER BY %s %s", sortColumn, sortOrder)

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argId)
		args = append(args, filter.Limit)
		argId++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argId)
		args = append(args, filter.Offset)
		argId++
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var docs []domain.Document
	for rows.Next() {
		var doc domain.Document
		err := rows.Scan(
			&doc.ID, &doc.FolderID, &doc.Filename, &doc.FilePath, &doc.Status, &doc.UploadedBy,
			&doc.FileSizeBytes, &doc.IsDeprecated, &doc.PreviousVersionID,
			&doc.Version, &doc.CreatedAt, &doc.UpdatedAt, &doc.FolderPath,
			&doc.DocumentType, &doc.RiskLevel, &doc.RiskReasoning, &doc.VendorName, &doc.BusinessUnit,
			&doc.EffectiveDate, &doc.ExpiryDate, &doc.Summary,
		)
		if err != nil {
			return nil, 0, err
		}
		docs = append(docs, doc)
	}

	return docs, total, nil
}

func (r *documentRepository) GetByID(ctx context.Context, id string) (*domain.Document, error) {
	query := `
		SELECT id, folder_id, filename, file_path, status, uploaded_by, file_size_bytes,
		       is_deprecated, previous_version_id, version, created_at, updated_at,
		       COALESCE(document_type, 'OTHER') as document_type,
		       COALESCE(risk_level, 'UNKNOWN') as risk_level,
		       COALESCE(risk_reasoning, '') as risk_reasoning,
		       COALESCE(vendor_name, '') as vendor_name,
		       COALESCE(business_unit, '') as business_unit,
		       effective_date, expiry_date,
		       COALESCE(summary, '') as summary
		FROM ingestion.documents WHERE id = $1`

	var doc domain.Document
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&doc.ID, &doc.FolderID, &doc.Filename, &doc.FilePath, &doc.Status, &doc.UploadedBy,
		&doc.FileSizeBytes, &doc.IsDeprecated, &doc.PreviousVersionID,
		&doc.Version, &doc.CreatedAt, &doc.UpdatedAt,
		&doc.DocumentType, &doc.RiskLevel, &doc.RiskReasoning, &doc.VendorName, &doc.BusinessUnit,
		&doc.EffectiveDate, &doc.ExpiryDate, &doc.Summary,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, err
	}

	return &doc, nil
}

func (r *documentRepository) UpdateMetadata(ctx context.Context, id string, filename *string, folderID *string) error {
	query := "UPDATE ingestion.documents SET updated_at = CURRENT_TIMESTAMP"
	var args []interface{}
	argCount := 1

	if filename != nil {
		query += fmt.Sprintf(", filename = $%d", argCount)
		args = append(args, *filename)
		argCount++
	}

	if folderID != nil {
		query += fmt.Sprintf(", folder_id = $%d", argCount)
		if *folderID == "null" || *folderID == "" {
			args = append(args, nil)
		} else {
			args = append(args, *folderID)
		}
		argCount++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, id)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *documentRepository) CreatePendingDocument(ctx context.Context, doc *domain.Document) error {
	version := 1
	if doc.PreviousVersionID != nil {
		var prevVersion int
		err := r.db.QueryRowContext(ctx, "SELECT version FROM ingestion.documents WHERE id = $1", *doc.PreviousVersionID).Scan(&prevVersion)
		if err == nil {
			version = prevVersion + 1
		}
	}
	doc.Version = version

	query := `
		INSERT INTO ingestion.documents (
			id, status, file_path, filename, previous_version_id, version, folder_id, file_size_bytes, uploaded_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`
	_, err := r.db.ExecContext(ctx, query,
		doc.ID, doc.Status, doc.FilePath, doc.Filename, doc.PreviousVersionID, doc.Version, doc.FolderID, doc.FileSizeBytes, doc.UploadedBy,
	)
	return err
}

func (r *documentRepository) RestoreDocument(ctx context.Context, id string) error {
	query := "UPDATE ingestion.documents SET status = 'PENDING', is_deprecated = FALSE WHERE id = $1"
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}


func (r *documentRepository) GetMetadataOptions(ctx context.Context) (*domain.MetadataOptions, error) {
	r.mu.Lock()
	if r.cachedOptions != nil && time.Since(r.optionsCacheTime) < 5*time.Minute {
		defer r.mu.Unlock()
		return r.cachedOptions, nil
	}
	r.mu.Unlock()

	// Need to query DB
	var vendors []string
	var busUnits []string

	vRows, err := r.db.QueryContext(ctx, "SELECT DISTINCT vendor_name FROM ingestion.documents WHERE vendor_name IS NOT NULL AND vendor_name != ''")
	if err != nil {
		return nil, err
	}
	defer vRows.Close()
	for vRows.Next() {
		var v string
		if err := vRows.Scan(&v); err == nil {
			vendors = append(vendors, v)
		}
	}

	bRows, err := r.db.QueryContext(ctx, "SELECT DISTINCT business_unit FROM ingestion.documents WHERE business_unit IS NOT NULL AND business_unit != ''")
	if err != nil {
		return nil, err
	}
	defer bRows.Close()
	for bRows.Next() {
		var b string
		if err := bRows.Scan(&b); err == nil {
			busUnits = append(busUnits, b)
		}
	}

	opts := &domain.MetadataOptions{
		Vendors:       vendors,
		BusinessUnits: busUnits,
	}

	r.mu.Lock()
	r.cachedOptions = opts
	r.optionsCacheTime = time.Now()
	r.mu.Unlock()

	return opts, nil
}

// UpdateRichMetadata updates Phase 1 metadata fields — only non-nil fields are updated.
func (r *documentRepository) UpdateRichMetadata(ctx context.Context, id string, meta domain.DocumentRichMetadata) error {
	query := "UPDATE ingestion.documents SET updated_at = CURRENT_TIMESTAMP"
	var args []interface{}
	argCount := 1

	if meta.DocumentType != nil {
		query += fmt.Sprintf(", document_type = $%d", argCount)
		args = append(args, *meta.DocumentType)
		argCount++
	}
	if meta.RiskLevel != nil {
		query += fmt.Sprintf(", risk_level = $%d", argCount)
		args = append(args, *meta.RiskLevel)
		argCount++
	}
	if meta.VendorName != nil {
		query += fmt.Sprintf(", vendor_name = $%d", argCount)
		args = append(args, *meta.VendorName)
		argCount++
	}
	if meta.BusinessUnit != nil {
		query += fmt.Sprintf(", business_unit = $%d", argCount)
		args = append(args, *meta.BusinessUnit)
		argCount++
	}
	if meta.EffectiveDate != nil {
		query += fmt.Sprintf(", effective_date = $%d", argCount)
		args = append(args, *meta.EffectiveDate)
		argCount++
	}
	if meta.ExpiryDate != nil {
		query += fmt.Sprintf(", expiry_date = $%d", argCount)
		args = append(args, *meta.ExpiryDate)
		argCount++
	}

	if argCount == 1 {
		return nil // Nothing to update
	}

	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, id)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// SaveDocumentSummary persists the generated summary to the DB for caching.
func (r *documentRepository) SaveDocumentSummary(ctx context.Context, id string, summary string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE ingestion.documents SET summary = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2",
		summary, id,
	)
	return err
}


func (r *documentRepository) UpdateDocumentStatus(ctx context.Context, documentID string, status string) error {
	query := `
		UPDATE ingestion.documents 
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	_, err := r.db.ExecContext(ctx, query, status, documentID)
	return err
}
