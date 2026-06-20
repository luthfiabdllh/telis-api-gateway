package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"telis-api-gateway/internal/domain"
)

type chatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) domain.ChatRepository {
	return &chatRepository{db: db}
}

func (r *chatRepository) CreateSession(ctx context.Context, session *domain.ChatSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *chatRepository) GetSessionByID(ctx context.Context, id uuid.UUID) (*domain.ChatSession, error) {
	var session domain.ChatSession
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *chatRepository) GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.ChatSession, error) {
	var sessions []*domain.ChatSession
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("updated_at desc").Find(&sessions).Error
	return sessions, err
}

func (r *chatRepository) UpdateSessionTitle(ctx context.Context, id uuid.UUID, title string) error {
	return r.db.WithContext(ctx).Model(&domain.ChatSession{}).Where("id = ?", id).Update("title", title).Error
}

func (r *chatRepository) DeleteSession(ctx context.Context, id uuid.UUID) error {
	// Karena di skema SQL ada ON DELETE CASCADE, menghapus session akan menghapus messages juga.
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.ChatSession{}).Error
}

func (r *chatRepository) CreateMessage(ctx context.Context, message *domain.ChatMessage) error {
	return r.db.WithContext(ctx).Create(message).Error
}

func (r *chatRepository) GetMessagesBySessionID(ctx context.Context, sessionID uuid.UUID) ([]*domain.ChatMessage, error) {
	var messages []*domain.ChatMessage
	err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at asc").Find(&messages).Error
	return messages, err
}
