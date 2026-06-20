package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"telis-api-gateway/internal/domain"
)

type folderUsecase struct {
	folderRepo domain.FolderRepository
	docUsecase domain.DocumentUsecase
}

func NewFolderUsecase(folderRepo domain.FolderRepository, docUsecase domain.DocumentUsecase) domain.FolderUsecase {
	return &folderUsecase{
		folderRepo: folderRepo,
		docUsecase: docUsecase,
	}
}

func (u *folderUsecase) CreateFolder(ctx context.Context, userID string, name string, parentID *string) (*domain.Folder, error) {
	if name == "" {
		return nil, errors.New("folder name is required")
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user id")
	}

	var parentUUID *uuid.UUID
	if parentID != nil && *parentID != "" {
		pID, err := uuid.Parse(*parentID)
		if err != nil {
			return nil, errors.New("invalid parent id")
		}
		
		// Verify parent exists
		_, err = u.folderRepo.GetByID(ctx, pID.String())
		if err != nil {
			return nil, errors.New("parent folder not found")
		}
		
		parentUUID = &pID
	}

	folder := &domain.Folder{
		Name:      name,
		ParentID:  parentUUID,
		CreatedBy: userUUID,
	}

	err = u.folderRepo.Create(ctx, folder)
	if err != nil {
		return nil, err
	}

	return folder, nil
}

func (u *folderUsecase) GetFolders(ctx context.Context, parentID *string) ([]domain.Folder, error) {
	return u.folderRepo.GetAll(ctx, parentID)
}

func (u *folderUsecase) RenameFolder(ctx context.Context, id string, name string) error {
	if name == "" {
		return errors.New("folder name is required")
	}
	
	// Check if folder exists
	_, err := u.folderRepo.GetByID(ctx, id)
	if err != nil {
		return errors.New("folder not found")
	}

	return u.folderRepo.Update(ctx, id, name)
}

func (u *folderUsecase) DeleteFolder(ctx context.Context, id string, userID string) error {
	// 1. Get all documents in this folder and its subfolders (Recursive)
	docIDs, err := u.folderRepo.GetAllDocumentsInFolderAndSubfolders(ctx, id)
	if err != nil {
		return err
	}

	// 2. Hard Delete all documents (This triggers Celery to clear Neo4j, Qdrant, and physical PDF)
	for _, docID := range docIDs {
		// Ignore individual errors to ensure we try deleting as much as possible, or handle them.
		_ = u.docUsecase.DeleteDocument(ctx, docID, userID)
	}

	// 3. Delete the folder itself (ON DELETE CASCADE will drop subfolders and DB rows for documents)
	return u.folderRepo.Delete(ctx, id)
}
