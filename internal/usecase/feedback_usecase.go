package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"telis-api-gateway/internal/domain"
)

type feedbackUsecase struct {
	repo domain.FeedbackRepository
}

func NewFeedbackUsecase(repo domain.FeedbackRepository) domain.FeedbackUsecase {
	return &feedbackUsecase{
		repo: repo,
	}
}

func (u *feedbackUsecase) SubmitFeedback(ctx context.Context, messageID string, userID uuid.UUID, rating int, comment string) (*domain.UserFeedback, error) {
	// Validate rating
	if rating != 1 && rating != -1 {
		return nil, errors.New("rating must be 1 (Thumbs Up) or -1 (Thumbs Down)")
	}

	if messageID == "" {
		return nil, errors.New("message_id is required")
	}

	feedback := &domain.UserFeedback{
		MessageID: messageID,
		UserID:    userID,
		Rating:    rating,
		Comment:   comment,
	}

	err := u.repo.Create(ctx, feedback)
	if err != nil {
		return nil, err
	}

	return feedback, nil
}
