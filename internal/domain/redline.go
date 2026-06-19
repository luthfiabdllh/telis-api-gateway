package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type RedlineStatus string

const (
	RedlineStatusPending    RedlineStatus = "PENDING"
	RedlineStatusProcessing RedlineStatus = "PROCESSING"
	RedlineStatusCompleted  RedlineStatus = "COMPLETED"
	RedlineStatusFailed     RedlineStatus = "FAILED"
)

type RedlineJob struct {
	ID             uuid.UUID     `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID         uuid.UUID     `gorm:"type:uuid;not null" json:"user_id"`
	SourceFilePath string        `gorm:"not null" json:"source_file_path"`
	TargetFilePath string        `gorm:"not null" json:"target_file_path"`
	Status         RedlineStatus `gorm:"type:varchar(20);default:'PENDING'" json:"status"`
	AnalysisResult string        `gorm:"type:text" json:"analysis_result"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

func (RedlineJob) TableName() string {
	return "gateway.redline_jobs"
}

type RedlineRepository interface {
	Create(ctx context.Context, job *RedlineJob) error
	GetByID(ctx context.Context, id uuid.UUID) (*RedlineJob, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status RedlineStatus, result string) error
}

type RedlineUsecase interface {
	CreateRedlineJob(ctx context.Context, userID uuid.UUID, sourceFile, targetFile []byte, sourceFilename, targetFilename string) (*RedlineJob, error)
	GetRedlineJob(ctx context.Context, id uuid.UUID) (*RedlineJob, error)
}
