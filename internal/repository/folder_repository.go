package repository

import (
	"context"
	"database/sql"
	"fmt"

	"telis-api-gateway/internal/domain"
)

type folderRepository struct {
	db *sql.DB
}

func NewFolderRepository(db *sql.DB) domain.FolderRepository {
	return &folderRepository{db: db}
}

func (r *folderRepository) Create(ctx context.Context, folder *domain.Folder) error {
	query := `INSERT INTO ingestion.folders (name, parent_id, created_by) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`
	err := r.db.QueryRowContext(ctx, query, folder.Name, folder.ParentID, folder.CreatedBy).Scan(&folder.ID, &folder.CreatedAt, &folder.UpdatedAt)
	return err
}

func (r *folderRepository) GetByID(ctx context.Context, id string) (*domain.Folder, error) {
	query := `SELECT id, name, parent_id, created_by, created_at, updated_at FROM ingestion.folders WHERE id = $1`
	var f domain.Folder
	err := r.db.QueryRowContext(ctx, query, id).Scan(&f.ID, &f.Name, &f.ParentID, &f.CreatedBy, &f.CreatedAt, &f.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *folderRepository) GetAll(ctx context.Context, parentID *string, search string) ([]domain.Folder, error) {
	var query string
	var args []interface{}
	argCount := 1

	if search != "" {
		query = `
			WITH RECURSIVE folder_tree AS (
				SELECT id, name, parent_id, created_by, created_at, updated_at, name::text as folder_path
				FROM ingestion.folders
				WHERE parent_id IS NULL
				UNION ALL
				SELECT f.id, f.name, f.parent_id, f.created_by, f.created_at, f.updated_at, (ft.folder_path || ' > ' || f.name)
				FROM ingestion.folders f
				INNER JOIN folder_tree ft ON f.parent_id = ft.id
			)
			SELECT id, name, parent_id, created_by, created_at, updated_at, folder_path
			FROM folder_tree
			WHERE name ILIKE $1
		`
		args = append(args, "%"+search+"%")
		argCount++
	} else {
		query = `SELECT id, name, parent_id, created_by, created_at, updated_at, '' as folder_path FROM ingestion.folders WHERE 1=1`
	}

	if parentID != nil {
		if *parentID == "null" || *parentID == "" {
			query += " AND parent_id IS NULL"
		} else {
			query += fmt.Sprintf(" AND parent_id = $%d", argCount)
			args = append(args, *parentID)
			argCount++
		}
	}

	query += " ORDER BY name ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []domain.Folder
	for rows.Next() {
		var f domain.Folder
		if err := rows.Scan(&f.ID, &f.Name, &f.ParentID, &f.CreatedBy, &f.CreatedAt, &f.UpdatedAt, &f.FolderPath); err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	return folders, nil
}

func (r *folderRepository) Update(ctx context.Context, id string, name string) error {
	query := `UPDATE ingestion.folders SET name = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, name, id)
	return err
}

func (r *folderRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM ingestion.folders WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *folderRepository) GetAllDocumentsInFolderAndSubfolders(ctx context.Context, folderID string) ([]string, error) {
	query := `
		WITH RECURSIVE folder_tree AS (
			SELECT id FROM ingestion.folders WHERE id = $1
			UNION ALL
			SELECT f.id FROM ingestion.folders f
			INNER JOIN folder_tree ft ON f.parent_id = ft.id
		)
		SELECT d.id::text FROM ingestion.documents d
		WHERE d.folder_id IN (SELECT id FROM folder_tree)
	`
	rows, err := r.db.QueryContext(ctx, query, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docIDs []string
	for rows.Next() {
		var docID string
		if err := rows.Scan(&docID); err != nil {
			return nil, err
		}
		docIDs = append(docIDs, docID)
	}
	return docIDs, nil
}

func (r *folderRepository) Move(ctx context.Context, id string, parentID *string) error {
	query := `UPDATE ingestion.folders SET parent_id = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	var pID interface{}
	if parentID == nil || *parentID == "null" || *parentID == "" {
		pID = nil
	} else {
		pID = *parentID
	}
	_, err := r.db.ExecContext(ctx, query, pID, id)
	return err
}

func (r *folderRepository) GetPath(ctx context.Context, id string) ([]domain.Folder, error) {
	query := `
		WITH RECURSIVE path AS (
			SELECT id, name, parent_id, created_by, created_at, updated_at, 1 as level
			FROM ingestion.folders
			WHERE id = $1
			UNION ALL
			SELECT f.id, f.name, f.parent_id, f.created_by, f.created_at, f.updated_at, p.level + 1
			FROM ingestion.folders f
			INNER JOIN path p ON f.id = p.parent_id
		)
		SELECT id, name, parent_id, created_by, created_at, updated_at
		FROM path
		ORDER BY level DESC
	`
	rows, err := r.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var path []domain.Folder
	for rows.Next() {
		var f domain.Folder
		if err := rows.Scan(&f.ID, &f.Name, &f.ParentID, &f.CreatedBy, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		path = append(path, f)
	}
	return path, nil
}
