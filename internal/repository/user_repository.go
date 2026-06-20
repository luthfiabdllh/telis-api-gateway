package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"telis-api-gateway/internal/domain"
)

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Preload("Role").Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil if not found
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Preload("Role").Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetAll(ctx context.Context, page, limit int, search string, roleID *int, isBanned *bool) ([]*domain.User, int64, error) {
	var users []*domain.User
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.User{})

	if search != "" {
		query = query.Where("username ILIKE ? OR email ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if roleID != nil {
		query = query.Where("role_id = ?", *roleID)
	}
	if isBanned != nil {
		query = query.Where("is_banned = ?", *isBanned)
	}

	// Count total documents
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Pagination
	offset := (page - 1) * limit
	err := query.Preload("Role").Order("created_at desc").Offset(offset).Limit(limit).Find(&users).Error

	return users, total, err
}

func (r *userRepository) UpdateRole(ctx context.Context, id uuid.UUID, roleID int) error {
	return r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", id).Update("role_id", roleID).Error
}

func (r *userRepository) UpdateStatus(ctx context.Context, id uuid.UUID, isBanned bool) error {
	return r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", id).Update("is_banned", isBanned).Error
}
