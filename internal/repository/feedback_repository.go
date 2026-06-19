package repository

import (
	"context"

	"gorm.io/gorm"
	"telis-api-gateway/internal/domain"
)

type feedbackRepository struct {
	db *gorm.DB
}

func NewFeedbackRepository(db *gorm.DB) domain.FeedbackRepository {
	return &feedbackRepository{db: db}
}

func (r *feedbackRepository) Create(ctx context.Context, feedback *domain.UserFeedback) error {
	return r.db.WithContext(ctx).Create(feedback).Error
}
