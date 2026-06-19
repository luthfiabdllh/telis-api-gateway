package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UserFeedback struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	MessageID string    `gorm:"type:varchar(255);not null"`
	UserID    uuid.UUID `gorm:"type:uuid;not null"`
	Rating    int       `gorm:"type:smallint;not null"` // 1 for Thumbs Up, -1 for Thumbs Down
	Comment   string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// TableName overrides the table name used by UserFeedback to `user_feedbacks` in schema `gateway`
func (UserFeedback) TableName() string {
	return "gateway.user_feedbacks"
}

// FeedbackRepository defines the interface for database operations related to UserFeedback
type FeedbackRepository interface {
	Create(ctx context.Context, feedback *UserFeedback) error
}

// FeedbackUsecase defines the interface for the business logic related to UserFeedback
type FeedbackUsecase interface {
	SubmitFeedback(ctx context.Context, messageID string, userID uuid.UUID, rating int, comment string) (*UserFeedback, error)
}
