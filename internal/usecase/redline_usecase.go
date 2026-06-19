package usecase

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"telis-api-gateway/internal/domain"
	"telis-api-gateway/internal/infrastructure/rabbitmq"
)

type redlineUsecase struct {
	repo         domain.RedlineRepository
	rmqPublisher rabbitmq.Publisher
	baseDir      string
}

func NewRedlineUsecase(repo domain.RedlineRepository, rmqPublisher rabbitmq.Publisher, baseDir string) domain.RedlineUsecase {
	// Ensure redlines directory exists
	os.MkdirAll(filepath.Join(baseDir, "redlines"), os.ModePerm)
	return &redlineUsecase{
		repo:         repo,
		rmqPublisher: rmqPublisher,
		baseDir:      baseDir,
	}
}

func (u *redlineUsecase) CreateRedlineJob(ctx context.Context, userID uuid.UUID, sourceFile, targetFile []byte, sourceFilename, targetFilename string) (*domain.RedlineJob, error) {
	// Generate unique filenames to avoid collision
	timestamp := time.Now().UnixNano()
	sourceFilePath := filepath.Join(u.baseDir, "redlines", fmt.Sprintf("%d_src_%s", timestamp, sourceFilename))
	targetFilePath := filepath.Join(u.baseDir, "redlines", fmt.Sprintf("%d_tgt_%s", timestamp, targetFilename))

	// Save files
	if err := ioutil.WriteFile(sourceFilePath, sourceFile, 0644); err != nil {
		return nil, err
	}
	if err := ioutil.WriteFile(targetFilePath, targetFile, 0644); err != nil {
		return nil, err
	}

	// Create job record
	job := &domain.RedlineJob{
		UserID:         userID,
		SourceFilePath: sourceFilePath,
		TargetFilePath: targetFilePath,
		Status:         domain.RedlineStatusPending,
	}

	if err := u.repo.Create(ctx, job); err != nil {
		return nil, err
	}

	// Publish to RabbitMQ
	if err := u.rmqPublisher.PublishRedlineTask(ctx, job.ID.String(), sourceFilePath, targetFilePath); err != nil {
		// Even if publish fails, we return the job (it will be stuck in PENDING, could be retried later)
		return job, fmt.Errorf("failed to publish to RabbitMQ: %v", err)
	}

	// Update status to PROCESSING (or let worker do it, but here we can just leave it as PENDING and worker updates it)
	return job, nil
}

func (u *redlineUsecase) GetRedlineJob(ctx context.Context, id uuid.UUID) (*domain.RedlineJob, error) {
	return u.repo.GetByID(ctx, id)
}
