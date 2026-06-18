package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
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

func (u *documentUsecase) UploadDocument(ctx context.Context, userID string, fileHeader *multipart.FileHeader) (string, error) {
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
	err = u.publisher.PublishDocumentTask(ctx, documentID, filePath)
	if err != nil {
		return "", fmt.Errorf("failed to publish to RabbitMQ: %v", err)
	}

	return documentID, nil
}
