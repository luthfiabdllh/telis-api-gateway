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

	// Phase 1 — Rich Metadata (v2.0)
	DocumentType  string     `json:"document_type,omitempty"`  // Kategori klasifikasi otomatis
	RiskLevel     string     `json:"risk_level,omitempty"`     // LOW / MEDIUM / HIGH / UNKNOWN
	VendorName    string     `json:"vendor_name,omitempty"`    // Nama pihak/vendor
	BusinessUnit  string     `json:"business_unit,omitempty"` // Divisi terkait
	EffectiveDate *time.Time `json:"effective_date,omitempty"` // Tanggal mulai berlaku
	ExpiryDate    *time.Time `json:"expiry_date,omitempty"`    // Tanggal berakhir
	Summary       string     `json:"summary,omitempty"`        // Ringkasan dalam Bahasa Indonesia
}

type DocumentFilter struct {
	Limit        int
	Offset       int
	Search       string
	Status       string
	IsDeprecated *bool
	FolderID     *string
	// Phase 1 filters
	DocumentType string // e.g. "NDA", "REGULATORY_DOCUMENT"
	RiskLevel    string // e.g. "HIGH", "MEDIUM"
	VendorName   string
	BusinessUnit string

	// Sorting
	SortBy    string // e.g. "filename", "created_at"
	SortOrder string // e.g. "asc", "desc"
}

type MetadataOptions struct {
	Vendors       []string `json:"vendors"`
	BusinessUnits []string `json:"business_units"`
}

type DocumentRepository interface {
	GetAll(ctx context.Context, filter DocumentFilter) ([]Document, int, error)
	GetByID(ctx context.Context, id string) (*Document, error)
	GetMetadataOptions(ctx context.Context) (*MetadataOptions, error)
	UpdateMetadata(ctx context.Context, id string, filename *string, folderID *string) error
	UpdateRichMetadata(ctx context.Context, id string, meta DocumentRichMetadata) error // Phase 1
	CreatePendingDocument(ctx context.Context, doc *Document) error
	RestoreDocument(ctx context.Context, id string) error
	SaveDocumentSummary(ctx context.Context, id string, summary string) error // Phase 1
}

// DocumentRichMetadata holds updatable metadata fields from Phase 1
type DocumentRichMetadata struct {
	DocumentType  *string    `json:"document_type"`
	RiskLevel     *string    `json:"risk_level"`
	VendorName    *string    `json:"vendor_name"`
	BusinessUnit  *string    `json:"business_unit"`
	EffectiveDate *time.Time `json:"effective_date"`
	ExpiryDate    *time.Time `json:"expiry_date"`
}

type DocumentUsecase interface {
	UploadDocument(ctx context.Context, userID string, fileHeader *multipart.FileHeader, folderID string, replacesDocumentID string) (string, error)
	DeleteDocument(ctx context.Context, documentID string, userID string) error
	DeprecateDocument(ctx context.Context, documentID string, userID string) error
	RestoreDocument(ctx context.Context, documentID string, userID string) error
	RenameDocument(ctx context.Context, documentID string, newName string) error
	MoveDocument(ctx context.Context, documentID string, newFolderID *string) error
	UpdateRichMetadata(ctx context.Context, documentID string, meta DocumentRichMetadata) error // Phase 1

	GetAllDocuments(ctx context.Context, filter DocumentFilter) ([]Document, int, error)
	GetDocumentByID(ctx context.Context, documentID string) (*Document, error)
	GetMetadataOptions(ctx context.Context) (*MetadataOptions, error)
	GetDocumentFilePath(ctx context.Context, documentID string) (string, string, error) // Returns filePath, filename, error
	SummarizeDocument(ctx context.Context, documentID string, force bool) (*DocumentSummaryResult, error) // Phase 1
}

// DocumentSummaryResult is the structured output of the summarization endpoint
type DocumentSummaryResult struct {
	DocumentID   string                 `json:"document_id"`
	Filename     string                 `json:"filename"`
	DocumentType string                 `json:"document_type"`
	Summary      map[string]interface{} `json:"summary"`
	Cached       bool                   `json:"cached"` // true if returned from DB cache
}
