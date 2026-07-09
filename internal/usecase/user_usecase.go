package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"telis-api-gateway/internal/domain"
)

type userUsecase struct {
	userRepo domain.UserRepository
}

func NewUserUsecase(userRepo domain.UserRepository) domain.UserUsecase {
	return &userUsecase{
		userRepo: userRepo,
	}
}

func (u *userUsecase) GetAllUsers(ctx context.Context, page, limit int, search string, roleID *int, isBanned *bool) ([]*domain.User, int64, error) {
	return u.userRepo.GetAll(ctx, page, limit, search, roleID, isBanned)
}

func (u *userUsecase) UpdateUserRole(ctx context.Context, id uuid.UUID, roleID int, reqByAdminID uuid.UUID) error {
	if id == reqByAdminID {
		return errors.New("cannot change your own role")
	}
	
	// Validasi RoleID agar tidak sembarangan (Misal 1=Admin, 2=User, 3=Legal)
	if roleID < 1 || roleID > 3 {
		return errors.New("invalid role ID")
	}

	return u.userRepo.UpdateRole(ctx, id, roleID)
}

func (u *userUsecase) UpdateUserStatus(ctx context.Context, id uuid.UUID, isBanned bool, reqByAdminID uuid.UUID) error {
	if id == reqByAdminID {
		return errors.New("cannot ban yourself")
	}

	return u.userRepo.UpdateStatus(ctx, id, isBanned)
}

func (u *userUsecase) GetUserMetrics(ctx context.Context) (*domain.UserMetrics, error) {
	return u.userRepo.GetUserMetrics(ctx)
}
