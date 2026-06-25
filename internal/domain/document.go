package domain

import (
	"context"
	"mime/multipart"
	"time"

	"github.com/google/uuid"
)

type Document struct {
	ID                uuid.UUID  `json:"id"`
	FolderID          *uuid.UUID `json:"folder_id,omitempty"`
	Filename          string     `json:"filename"`
	FilePath          string     `json:"-"` // Hide file path from API
	Status            string     `json:"status"`
	UploadedBy        *uuid.UUID `json:"uploaded_by,omitempty"`
	FileSizeBytes     *int64     `json:"file_size_bytes,omitempty"`
	IsDeprecated      bool       `json:"is_deprecated"`
	PreviousVersionID *uuid.UUID `json:"previous_version_id,omitempty"`
	Version           int        `json:"version"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	FolderPath        string     `json:"folder_path,omitempty"`
}

type DocumentFilter struct {
	Limit        int
	Offset       int
	Search       string
	Status       string
	IsDeprecated *bool
	FolderID     *string
}

type DocumentRepository interface {
	GetAll(ctx context.Context, filter DocumentFilter) ([]Document, int, error)
	GetByID(ctx context.Context, id string) (*Document, error)
	UpdateMetadata(ctx context.Context, id string, filename *string, folderID *string) error
	CreatePendingDocument(ctx context.Context, doc *Document) error
	RestoreDocument(ctx context.Context, id string) error
}

type DocumentUsecase interface {
	UploadDocument(ctx context.Context, userID string, fileHeader *multipart.FileHeader, folderID string, replacesDocumentID string) (string, error)
	DeleteDocument(ctx context.Context, documentID string, userID string) error
	DeprecateDocument(ctx context.Context, documentID string, userID string) error
	RestoreDocument(ctx context.Context, documentID string, userID string) error
	RenameDocument(ctx context.Context, documentID string, newName string) error
	MoveDocument(ctx context.Context, documentID string, newFolderID *string) error
	
	GetAllDocuments(ctx context.Context, filter DocumentFilter) ([]Document, int, error)
	GetDocumentByID(ctx context.Context, documentID string) (*Document, error)
	GetDocumentFilePath(ctx context.Context, documentID string) (string, string, error) // Returns filePath, filename, error
}
