package domain

import (
	"context"
	"mime/multipart"
)

type DocumentUsecase interface {
	UploadDocument(ctx context.Context, userID string, fileHeader *multipart.FileHeader, replacesDocumentID string) (string, error)
	DeleteDocument(ctx context.Context, documentID string, userID string) error
	DeprecateDocument(ctx context.Context, documentID string, userID string) error
}
