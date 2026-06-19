package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"telis-api-gateway/internal/domain"
)

type redlineRepository struct {
	db *gorm.DB
}

func NewRedlineRepository(db *gorm.DB) domain.RedlineRepository {
	return &redlineRepository{db: db}
}

func (r *redlineRepository) Create(ctx context.Context, job *domain.RedlineJob) error {
	return r.db.WithContext(ctx).Create(job).Error
}

func (r *redlineRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.RedlineJob, error) {
	var job domain.RedlineJob
	if err := r.db.WithContext(ctx).First(&job, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *redlineRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.RedlineStatus, result string) error {
	return r.db.WithContext(ctx).Model(&domain.RedlineJob{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":          status,
		"analysis_result": result,
	}).Error
}
