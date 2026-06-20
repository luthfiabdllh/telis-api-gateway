package domain

import (
	"context"
	"mime/multipart"
	"time"

	"github.com/google/uuid"
)

type Document struct {
	ID                uuid.UUID  `json:"id"`
	Filename          string     `json:"filename"`
	FilePath          string     `json:"-"` // Hide file path from API
	Status            string     `json:"status"`
	UploadedBy        uuid.UUID  `json:"uploaded_by"`
	FileSizeBytes     int64      `json:"file_size_bytes"`
	IsDeprecated      bool       `json:"is_deprecated"`
	PreviousVersionID *uuid.UUID `json:"previous_version_id,omitempty"`
	Version           int        `json:"version"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type DocumentFilter struct {
	Limit        int
	Offset       int
	Search       string
	Status       string
	IsDeprecated *bool
}

type DocumentRepository interface {
	GetAll(ctx context.Context, filter DocumentFilter) ([]Document, int, error)
	GetByID(ctx context.Context, id string) (*Document, error)
}

type DocumentUsecase interface {
	UploadDocument(ctx context.Context, userID string, fileHeader *multipart.FileHeader, replacesDocumentID string) (string, error)
	DeleteDocument(ctx context.Context, documentID string, userID string) error
	DeprecateDocument(ctx context.Context, documentID string, userID string) error
	
	GetAllDocuments(ctx context.Context, filter DocumentFilter) ([]Document, int, error)
	GetDocumentByID(ctx context.Context, documentID string) (*Document, error)
	GetDocumentFilePath(ctx context.Context, documentID string) (string, string, error) // Returns filePath, filename, error
}
