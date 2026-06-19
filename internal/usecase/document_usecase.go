package usecase

import (
	"context"
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
)

type documentUsecase struct {
	publisher rabbitmq.Publisher
	baseDir   string // e.g. ../shared_docs
}

func NewDocumentUsecase(publisher rabbitmq.Publisher, baseDir string) domain.DocumentUsecase {
	return &documentUsecase{
		publisher: publisher,
		baseDir:   baseDir,
	}
}

func (u *documentUsecase) UploadDocument(ctx context.Context, userID string, fileHeader *multipart.FileHeader, replacesDocumentID string) (string, error) {
	// 1. Generate unique Document ID
	documentID := uuid.New().String()

	// 2. Validate file type (basic check)
	if filepath.Ext(fileHeader.Filename) != ".pdf" {
		return "", errors.New("only PDF files are allowed")
	}

	// 3. Save file physically to shared_docs
	fileName := fmt.Sprintf("%s_%s", documentID, fileHeader.Filename)
	filePath := filepath.Join(u.baseDir, fileName)

	// Ensure directory exists
	if err := os.MkdirAll(u.baseDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %v", err)
	}

	// Open incoming file
	src, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	// Copy content
	if _, err = io.Copy(dst, src); err != nil {
		return "", err
	}

	// 4. Publish to RabbitMQ
	// Ingestion service will read from the same relative shared folder
	err = u.publisher.PublishDocumentTask(ctx, documentID, filePath, replacesDocumentID)
	if err != nil {
		return "", fmt.Errorf("failed to publish to RabbitMQ: %v", err)
	}

	// If it replaces an older version, also publish deprecate task for the old document
	if replacesDocumentID != "" {
		err = u.publisher.PublishDeprecateTask(ctx, replacesDocumentID)
		if err != nil {
			log.Printf("failed to deprecate old document %s: %v", replacesDocumentID, err)
		}
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
