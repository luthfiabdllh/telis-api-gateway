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

func (r *userRepository) GetAll(ctx context.Context, page, limit int, search string, roleID *int, isBanned *bool, sortBy, sortDir string) ([]*domain.User, int64, error) {
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
	
	// Ensure safe order by fallback
	validSortCols := map[string]bool{"username": true, "email": true, "created_at": true, "role_id": true, "is_banned": true}
	if !validSortCols[sortBy] {
		sortBy = "created_at"
	}
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
	}

	// Pagination
	offset := (page - 1) * limit
	err := query.Preload("Role").Order(sortBy + " " + sortDir).Offset(offset).Limit(limit).Find(&users).Error

	return users, total, err
}

func (r *userRepository) UpdateRole(ctx context.Context, id uuid.UUID, roleID int) error {
	return r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", id).Update("role_id", roleID).Error
}

func (r *userRepository) UpdateStatus(ctx context.Context, id uuid.UUID, isBanned bool) error {
	return r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", id).Update("is_banned", isBanned).Error
}

func (r *userRepository) GetUserMetrics(ctx context.Context) (*domain.UserMetrics, error) {
	var total, active, banned, admins int64

	if err := r.db.WithContext(ctx).Model(&domain.User{}).Count(&total).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&domain.User{}).Where("is_banned = ?", false).Count(&active).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&domain.User{}).Where("is_banned = ?", true).Count(&banned).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&domain.User{}).Where("role_id = ?", 1).Count(&admins).Error; err != nil {
		return nil, err
	}

	return &domain.UserMetrics{
		TotalUsers:  total,
		ActiveUsers: active,
		BannedUsers: banned,
		TotalAdmins: admins,
	}, nil
}
