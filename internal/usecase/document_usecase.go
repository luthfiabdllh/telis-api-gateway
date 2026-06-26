package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"telis-api-gateway/internal/domain"
	"telis-api-gateway/internal/infrastructure/rabbitmq"
	grpcClient "telis-api-gateway/internal/infrastructure/grpc"
)

type documentUsecase struct {
	publisher   rabbitmq.Publisher
	repo        domain.DocumentRepository
	baseDir     string // e.g. ../shared_docs
	agentClient grpcClient.AgentClient
}

func NewDocumentUsecase(publisher rabbitmq.Publisher, repo domain.DocumentRepository, baseDir string, agentClient grpcClient.AgentClient) domain.DocumentUsecase {
	return &documentUsecase{
		publisher:   publisher,
		repo:        repo,
		baseDir:     baseDir,
		agentClient: agentClient,
	}
}

func (u *documentUsecase) GetAllDocuments(ctx context.Context, filter domain.DocumentFilter) ([]domain.Document, int, error) {
	return u.repo.GetAll(ctx, filter)
}

func (u *documentUsecase) GetDocumentByID(ctx context.Context, documentID string) (*domain.Document, error) {
	doc, err := u.repo.GetByID(ctx, documentID)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, errors.New("document not found")
	}
	return doc, nil
}

func (u *documentUsecase) GetDocumentFilePath(ctx context.Context, documentID string) (string, string, error) {
	doc, err := u.repo.GetByID(ctx, documentID)
	if err != nil {
		return "", "", err
	}
	if doc == nil {
		return "", "", errors.New("document not found")
	}

	// We only return the file path and filename. The path is relative to ingestion.
	// But in Gateway, the physical file is at u.baseDir / {documentID}_{filename}
	fileName := fmt.Sprintf("%s_%s", documentID, doc.Filename)
	fullPath := filepath.Join(u.baseDir, fileName)

	return fullPath, doc.Filename, nil
}

func (u *documentUsecase) GetMetadataOptions(ctx context.Context) (*domain.MetadataOptions, error) {
	return u.repo.GetMetadataOptions(ctx)
}

func (u *documentUsecase) UploadDocument(ctx context.Context, userID string, fileHeader *multipart.FileHeader, folderID string, replacesDocumentID string) (string, error) {
	// 1. Generate unique Document ID
	documentID := uuid.New().String()

	// 2. Save file temporarily in shared_docs
	if err := os.MkdirAll(u.baseDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create directory: %v", err)
	}
	
	fileName := fmt.Sprintf("%s_%s", documentID, fileHeader.Filename)
	fullPath := filepath.Join(u.baseDir, fileName)

	// Save the physical file
	src, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return "", err
	}

	// Create pending document record
	var folderUUIDPtr *uuid.UUID
	if folderID != "" && folderID != "null" {
		id, err := uuid.Parse(folderID)
		if err == nil {
			folderUUIDPtr = &id
		}
	}

	var replacesUUIDPtr *uuid.UUID
	if replacesDocumentID != "" && replacesDocumentID != "null" {
		id, err := uuid.Parse(replacesDocumentID)
		if err == nil {
			replacesUUIDPtr = &id
		}
	}

	var userUUIDPtr *uuid.UUID
	if userID != "" {
		id, err := uuid.Parse(userID)
		if err == nil {
			userUUIDPtr = &id
		}
	}

	docUUID, _ := uuid.Parse(documentID)

	doc := &domain.Document{
		ID:                docUUID,
		Status:            "PENDING",
		Filename:          fileHeader.Filename,
		FilePath:          fileName,
		FolderID:          folderUUIDPtr,
		PreviousVersionID: replacesUUIDPtr,
		UploadedBy:        userUUIDPtr,
		FileSizeBytes:     &fileHeader.Size,
	}

	if err := u.repo.CreatePendingDocument(ctx, doc); err != nil {
		os.Remove(fullPath)
		return "", fmt.Errorf("failed to save document metadata: %v", err)
	}

	// 3. Publish message to RabbitMQ for ingestion worker
	payload := map[string]interface{}{
		"action":               "ingest",
		"document_id":          documentID,
		"file_path":            fullPath,
		"filename":             fileHeader.Filename,
		"user_id":              userID,
		"file_size_bytes":      fileHeader.Size,
		"replaces_document_id": replacesDocumentID,
		"folder_id":            folderID,
	}

	err = u.publisher.Publish(ctx, "ingestion_queue", payload)
	if err != nil {
		log.Printf("Failed to publish to RabbitMQ: %v", err)
		os.Remove(fullPath)
		return "", err
	}

	return documentID, nil
}

func (u *documentUsecase) DeprecateDocument(ctx context.Context, documentID string, userID string) error {
	// Publish deprecate task to RabbitMQ
	err := u.publisher.PublishDeprecateTask(ctx, documentID)
	if err != nil {
		return fmt.Errorf("failed to publish deprecate task to RabbitMQ: %v", err)
	}

	// NOTE: Unlike DeleteDocument, we DO NOT delete the physical PDF file
	// because it is preserved for archival purposes.

	return nil
}

func (u *documentUsecase) RestoreDocument(ctx context.Context, documentID string, userID string) error {
	doc, err := u.repo.GetByID(ctx, documentID)
	if err != nil {
		return err
	}
	if doc == nil {
		return errors.New("document not found")
	}

	fileName := fmt.Sprintf("%s_%s", documentID, doc.Filename)
	fullPath := filepath.Join(u.baseDir, fileName)

	// Verify physical file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return errors.New("File fisik tidak ditemukan, Restore gagal")
	}

	// Update DB Status to PENDING and is_deprecated to FALSE
	if err := u.repo.RestoreDocument(ctx, documentID); err != nil {
		return fmt.Errorf("failed to restore document status: %v", err)
	}

	var folderID string
	if doc.FolderID != nil {
		folderID = doc.FolderID.String()
	}

	var previousVersionID string
	if doc.PreviousVersionID != nil {
		previousVersionID = doc.PreviousVersionID.String()
	}

	// Publish message to RabbitMQ for ingestion worker
	payload := map[string]interface{}{
		"action":               "ingest",
		"document_id":          documentID,
		"file_path":            fullPath,
		"filename":             doc.Filename,
		"user_id":              userID,
		"file_size_bytes":      doc.FileSizeBytes,
		"replaces_document_id": previousVersionID,
		"folder_id":            folderID,
	}

	err = u.publisher.Publish(ctx, "ingestion_queue", payload)
	if err != nil {
		log.Printf("Failed to publish restore to RabbitMQ: %v", err)
		return err
	}

	return nil
}

func (u *documentUsecase) DeleteDocument(ctx context.Context, documentID string, userID string) error {
	// Publish delete task to RabbitMQ
	err := u.publisher.PublishDeleteTask(ctx, documentID)
	if err != nil {
		return fmt.Errorf("failed to publish delete task to RabbitMQ: %v", err)
	}

	// Attempt to delete any physical files matching the documentID prefix
	// Find the file by pattern since we don't store the exact filename in memory here
	pattern := filepath.Join(u.baseDir, fmt.Sprintf("%s_*.pdf", documentID))
	matches, _ := filepath.Glob(pattern)
	for _, match := range matches {
		os.Remove(match)
	}

	return nil
}

func (u *documentUsecase) RenameDocument(ctx context.Context, documentID string, newName string) error {
	if len(newName) < 4 || newName[len(newName)-4:] != ".pdf" {
		newName = newName + ".pdf"
	}
	
	doc, err := u.repo.GetByID(ctx, documentID)
	if err != nil {
		return err
	}
	if doc == nil {
		return errors.New("document not found")
	}

	oldFileName := fmt.Sprintf("%s_%s", documentID, doc.Filename)
	oldPath := filepath.Join(u.baseDir, oldFileName)
	
	newFileName := fmt.Sprintf("%s_%s", documentID, newName)
	newPath := filepath.Join(u.baseDir, newFileName)

	if err := os.Rename(oldPath, newPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to rename physical file: %v", err)
	}

	return u.repo.UpdateMetadata(ctx, documentID, &newName, nil)
}

func (u *documentUsecase) MoveDocument(ctx context.Context, documentID string, newFolderID *string) error {
	return u.repo.UpdateMetadata(ctx, documentID, nil, newFolderID)
}

// UpdateRichMetadata updates Phase 1 metadata fields for a document.
// Accessible by Admin and Legal roles via PATCH /documents/:id/metadata.
func (u *documentUsecase) UpdateRichMetadata(ctx context.Context, documentID string, meta domain.DocumentRichMetadata) error {
	_, err := u.repo.GetByID(ctx, documentID)
	if err != nil {
		return err
	}
	return u.repo.UpdateRichMetadata(ctx, documentID, meta)
}

// SummarizeDocument returns a structured summary of a document.
// Uses hybrid cache-first pattern:
//   - If summary already exists in DB, return it immediately.
//   - If not, call Agent Service gRPC to generate summary, persist to DB, then return.
func (u *documentUsecase) SummarizeDocument(ctx context.Context, documentID string, force bool) (*domain.DocumentSummaryResult, error) {
	doc, err := u.repo.GetByID(ctx, documentID)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, errors.New("document not found")
	}

	// Cache-first: return from DB if summary already generated and force is false
	if !force && doc.Summary != "" {
		var summaryMap map[string]interface{}
		if err := json.Unmarshal([]byte(doc.Summary), &summaryMap); err != nil {
			// Summary stored as plain text, wrap it
			summaryMap = map[string]interface{}{"ringkasan_singkat": doc.Summary}
		}
		return &domain.DocumentSummaryResult{
			DocumentID:   documentID,
			Filename:     doc.Filename,
			DocumentType: doc.DocumentType,
			Summary:      summaryMap,
			Cached:       true,
		}, nil
	}

	// Not cached — call Agent Service via gRPC
	summaryJSON, err := u.agentClient.SummarizeDocument(ctx, documentID, doc.DocumentType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %v", err)
	}

	// Persist to DB for future cache hits
	if saveErr := u.repo.SaveDocumentSummary(ctx, documentID, summaryJSON); saveErr != nil {
		log.Printf("Warning: failed to cache summary for document %s: %v", documentID, saveErr)
	}

	var summaryMap map[string]interface{}
	if err := json.Unmarshal([]byte(summaryJSON), &summaryMap); err != nil {
		summaryMap = map[string]interface{}{"ringkasan_singkat": summaryJSON}
	}

	return &domain.DocumentSummaryResult{
		DocumentID:   documentID,
		Filename:     doc.Filename,
		DocumentType: doc.DocumentType,
		Summary:      summaryMap,
		Cached:       false,
	}, nil
}
