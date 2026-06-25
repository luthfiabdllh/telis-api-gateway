package repository

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"telis-api-gateway/internal/domain"
)

type feedbackRepository struct {
	db *gorm.DB
}

func NewFeedbackRepository(db *gorm.DB) domain.FeedbackRepository {
	return &feedbackRepository{db: db}
}

func (r *feedbackRepository) Create(ctx context.Context, feedback *domain.UserFeedback) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "message_id"}}, // assuming message_id is unique
		DoUpdates: clause.AssignmentColumns([]string{"rating", "comment"}),
	}).Create(feedback).Error
}
