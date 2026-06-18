package domain

import (
	"context"
	"mime/multipart"
)

type DocumentUsecase interface {
	UploadDocument(ctx context.Context, userID string, fileHeader *multipart.FileHeader) (string, error)
}
