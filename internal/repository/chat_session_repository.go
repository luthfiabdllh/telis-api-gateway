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

func (r *chatRepository) GetSessionsByUserID(ctx context.Context, userID uuid.UUID, search string, page int, limit int) ([]*domain.ChatSession, int64, error) {
	var sessions []*domain.ChatSession
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.ChatSession{}).Where("user_id = ?", userID)

	if search != "" {
		query = query.Where("title ILIKE ?", "%"+search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.Order("updated_at desc").Offset(offset).Limit(limit).Find(&sessions).Error
	return sessions, total, err
}

func (r *chatRepository) UpdateSessionTitle(ctx context.Context, id uuid.UUID, title string) error {
	return r.db.WithContext(ctx).Model(&domain.ChatSession{}).Where("id = ?", id).Update("title", title).Error
}

func (r *chatRepository) DeleteSession(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Hapus semua pesan yang berelasi dengan session_id ini terlebih dahulu (Manual Cascade)
		if err := tx.Where("session_id = ?", id).Delete(&domain.ChatMessage{}).Error; err != nil {
			return err
		}

		// Setelah pesan terhapus, hapus session-nya
		if err := tx.Where("id = ?", id).Delete(&domain.ChatSession{}).Error; err != nil {
			return err
		}
		
		return nil
	})
}

func (r *chatRepository) CreateMessage(ctx context.Context, message *domain.ChatMessage) error {
	return r.db.WithContext(ctx).Create(message).Error
}

func (r *chatRepository) GetMessagesBySessionID(ctx context.Context, sessionID uuid.UUID) ([]*domain.ChatMessage, error) {
	var messages []*domain.ChatMessage
	err := r.db.WithContext(ctx).Preload("Feedback").Where("session_id = ?", sessionID).Order("created_at asc").Find(&messages).Error
	return messages, err
}
