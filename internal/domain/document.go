package domain

import (
	"context"
	"mime/multipart"
)

type DocumentUsecase interface {
	UploadDocument(ctx context.Context, userID string, fileHeader *multipart.FileHeader) (string, error)
	DeleteDocument(ctx context.Context, documentID string, userID string) error
}
