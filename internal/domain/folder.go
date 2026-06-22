package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Folder struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	CreatedBy uuid.UUID  `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	FolderPath string     `json:"folder_path,omitempty"`
}

type FolderRepository interface {
	Create(ctx context.Context, folder *Folder) error
	GetByID(ctx context.Context, id string) (*Folder, error)
	GetAll(ctx context.Context, parentID *string, search string) ([]Folder, error)
	Update(ctx context.Context, id string, name string) error
	Delete(ctx context.Context, id string) error
	GetAllDocumentsInFolderAndSubfolders(ctx context.Context, folderID string) ([]string, error) // Returns document IDs
	Move(ctx context.Context, id string, parentID *string) error
	GetPath(ctx context.Context, id string) ([]Folder, error)
}

type FolderUsecase interface {
	CreateFolder(ctx context.Context, userID string, name string, parentID *string) (*Folder, error)
	GetFolders(ctx context.Context, parentID *string, search string) ([]Folder, error)
	GetFolderByID(ctx context.Context, id string) (*Folder, error)
	RenameFolder(ctx context.Context, id string, name string) error
	MoveFolder(ctx context.Context, id string, parentID *string) error
	GetFolderPath(ctx context.Context, id string) ([]Folder, error)
	DeleteFolder(ctx context.Context, id string, userID string) error
}
