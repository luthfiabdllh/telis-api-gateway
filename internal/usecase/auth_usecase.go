package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"telis-api-gateway/config"
	"telis-api-gateway/internal/domain"
	"telis-api-gateway/pkg/utils"
)

type authUsecase struct {
	userRepo domain.UserRepository
	cfg      *config.Config
}

func NewAuthUsecase(userRepo domain.UserRepository, cfg *config.Config) domain.AuthUsecase {
	return &authUsecase{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

func (u *authUsecase) Register(ctx context.Context, username, email, password string, roleID int) (*domain.User, error) {
	// 1. Check if user already exists
	existingUser, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, errors.New("email already registered")
	}

	// 2. Hash Password
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}

	// 3. Create User entity
	user := &domain.User{
		Username:     username,
		Email:        email,
		PasswordHash: hashedPassword,
		RoleID:       roleID,
	}

	// 4. Save to DB
	err = u.userRepo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (u *authUsecase) Login(ctx context.Context, email, password string) (string, string, error) {
	// 1. Find user by email
	user, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", "", err
	}
	if user == nil {
		return "", "", errors.New("invalid email or password")
	}

	if user.IsBanned {
		return "", "", errors.New("account is banned. please contact administrator")
	}

	// 2. Verify password
	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return "", "", errors.New("invalid email or password")
	}

	// 3. Generate Tokens
	roleName := "User"
	if user.Role.Name != "" {
		roleName = user.Role.Name
	}
	
	accessToken, refreshToken, err := utils.GenerateTokens(
		user.ID, 
		roleName, 
		u.cfg.JWTSecret, 
		u.cfg.JWTAccessExpMinutes, 
		u.cfg.JWTRefreshExpDays,
	)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (u *authUsecase) LoginSSO(ctx context.Context, email string) (string, string, error) {
	// 1. Find user by email
	user, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", "", err
	}
	if user == nil {
		// Strict Whitelist: Jika tidak ada, tolak! Jangan auto-provision.
		return "", "", errors.New("email is not registered. please contact admin to register your account first")
	}

	if user.IsBanned {
		return "", "", errors.New("account is banned. please contact administrator")
	}

	// 2. Generate Tokens
	roleName := "User"
	if user.Role.Name != "" {
		roleName = user.Role.Name
	}
	
	accessToken, refreshToken, err := utils.GenerateTokens(
		user.ID, 
		roleName, 
		u.cfg.JWTSecret, 
		u.cfg.JWTAccessExpMinutes, 
		u.cfg.JWTRefreshExpDays,
	)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (u *authUsecase) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	// 1. Validate the refresh token
	claims, err := utils.ValidateToken(refreshToken, u.cfg.JWTSecret)
	if err != nil {
		return "", "", errors.New("invalid or expired refresh token")
	}

	// 2. Check token type
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "refresh" {
		return "", "", errors.New("invalid token type")
	}

	// 3. Extract user ID
	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return "", "", errors.New("invalid token claims")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return "", "", errors.New("invalid user id in token")
	}

	// 4. Verify user still exists
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", "", err
	}
	if user == nil {
		return "", "", errors.New("user not found")
	}

	if user.IsBanned {
		return "", "", errors.New("account is banned. please contact administrator")
	}

	// 5. Generate new tokens
	roleName := "User"
	if user.Role.Name != "" {
		roleName = user.Role.Name
	}

	newAccessToken, newRefreshToken, err := utils.GenerateTokens(
		user.ID,
		roleName,
		u.cfg.JWTSecret,
		u.cfg.JWTAccessExpMinutes,
		u.cfg.JWTRefreshExpDays,
	)
	if err != nil {
		return "", "", err
	}

	return newAccessToken, newRefreshToken, nil
}
