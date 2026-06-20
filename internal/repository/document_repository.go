package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"telis-api-gateway/internal/domain"
)

type documentRepository struct {
	db *sql.DB
}

func NewDocumentRepository(db *sql.DB) domain.DocumentRepository {
	return &documentRepository{db: db}
}

func (r *documentRepository) GetAll(ctx context.Context, filter domain.DocumentFilter) ([]domain.Document, int, error) {
	query := `SELECT id, filename, file_path, status, uploaded_by, file_size_bytes, is_deprecated, previous_version_id, version, created_at, updated_at FROM ingestion.documents WHERE 1=1`
	countQuery := `SELECT count(*) FROM ingestion.documents WHERE 1=1`

	var conditions []string
	var args []interface{}
	argId := 1

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("filename ILIKE $%d", argId))
		args = append(args, "%"+filter.Search+"%")
		argId++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argId))
		args = append(args, filter.Status)
		argId++
	}

	if filter.IsDeprecated != nil {
		conditions = append(conditions, fmt.Sprintf("is_deprecated = $%d", argId))
		args = append(args, *filter.IsDeprecated)
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
	query += " ORDER BY created_at DESC"

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
			&doc.ID, &doc.Filename, &doc.FilePath, &doc.Status, &doc.UploadedBy,
			&doc.FileSizeBytes, &doc.IsDeprecated, &doc.PreviousVersionID,
			&doc.Version, &doc.CreatedAt, &doc.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		docs = append(docs, doc)
	}

	return docs, total, nil
}

func (r *documentRepository) GetByID(ctx context.Context, id string) (*domain.Document, error) {
	query := `SELECT id, filename, file_path, status, uploaded_by, file_size_bytes, is_deprecated, previous_version_id, version, created_at, updated_at FROM ingestion.documents WHERE id = $1`

	var doc domain.Document
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&doc.ID, &doc.Filename, &doc.FilePath, &doc.Status, &doc.UploadedBy,
		&doc.FileSizeBytes, &doc.IsDeprecated, &doc.PreviousVersionID,
		&doc.Version, &doc.CreatedAt, &doc.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, err
	}

	return &doc, nil
}
