package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"telis-api-gateway/internal/domain"
)

type chatUsecase struct {
	chatRepo domain.ChatRepository
}

func NewChatUsecase(chatRepo domain.ChatRepository) domain.ChatUsecase {
	return &chatUsecase{
		chatRepo: chatRepo,
	}
}

func (u *chatUsecase) GetSessions(ctx context.Context, userID uuid.UUID) ([]*domain.ChatSession, error) {
	return u.chatRepo.GetSessionsByUserID(ctx, userID)
}

func (u *chatUsecase) GetMessages(ctx context.Context, sessionID uuid.UUID, userID uuid.UUID) ([]*domain.ChatMessage, error) {
	// Verifikasi kepemilikan sesi
	session, err := u.chatRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("chat session not found")
		}
		return nil, err
	}

	if session.UserID != userID {
		return nil, errors.New("unauthorized to view this chat session")
	}

	return u.chatRepo.GetMessagesBySessionID(ctx, sessionID)
}

func (u *chatUsecase) DeleteSession(ctx context.Context, sessionID uuid.UUID, userID uuid.UUID) error {
	session, err := u.chatRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.UserID != userID {
		return errors.New("unauthorized to delete this chat session")
	}

	return u.chatRepo.DeleteSession(ctx, sessionID)
}

func (u *chatUsecase) EnsureSessionExists(ctx context.Context, sessionIDStr string, userID uuid.UUID, initialMessage string) error {
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return err
	}

	_, err = u.chatRepo.GetSessionByID(ctx, sessionID)
	if err == nil {
		return nil // Session already exists
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err // Some other DB error
	}

	// Buat judul (Potong 30 karakter pertama)
	title := initialMessage
	if len(title) > 30 {
		title = title[:30] + "..."
	}
	
	// Pastikan judul tidak kosong dan bebas newline
	title = strings.TrimSpace(strings.ReplaceAll(title, "\n", " "))
	if title == "" {
		title = "New Chat"
	}

	newSession := &domain.ChatSession{
		ID:        sessionID,
		UserID:    userID,
		Title:     title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return u.chatRepo.CreateSession(ctx, newSession)
}

func (u *chatUsecase) SaveMessage(ctx context.Context, sessionIDStr string, sender string, content string) error {
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return err
	}

	message := &domain.ChatMessage{
		SessionID: sessionID,
		Sender:    sender,
		Content:   content,
		CreatedAt: time.Now(),
	}

	return u.chatRepo.CreateMessage(ctx, message)
}
