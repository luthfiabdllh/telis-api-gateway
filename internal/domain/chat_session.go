package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ChatSession struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ChatMessage struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	SessionID uuid.UUID `json:"session_id"`
	Sender    string    `json:"sender"` // "user" or "ai"
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func (ChatSession) TableName() string {
	return "gateway.chat_sessions"
}

func (ChatMessage) TableName() string {
	return "gateway.chat_messages"
}

type ChatRepository interface {
	CreateSession(ctx context.Context, session *ChatSession) error
	GetSessionByID(ctx context.Context, id uuid.UUID) (*ChatSession, error)
	GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]*ChatSession, error)
	UpdateSessionTitle(ctx context.Context, id uuid.UUID, title string) error
	DeleteSession(ctx context.Context, id uuid.UUID) error

	CreateMessage(ctx context.Context, message *ChatMessage) error
	GetMessagesBySessionID(ctx context.Context, sessionID uuid.UUID) ([]*ChatMessage, error)
}

type ChatUsecase interface {
	GetSessions(ctx context.Context, userID uuid.UUID) ([]*ChatSession, error)
	GetMessages(ctx context.Context, sessionID uuid.UUID, userID uuid.UUID) ([]*ChatMessage, error)
	RenameSession(ctx context.Context, sessionID uuid.UUID, userID uuid.UUID, newTitle string) error
	DeleteSession(ctx context.Context, sessionID uuid.UUID, userID uuid.UUID) error

	// Called internally during stream
	EnsureSessionExists(ctx context.Context, sessionID string, userID uuid.UUID, initialMessage string) error
	SaveMessage(ctx context.Context, sessionID string, sender string, content string) error
}
