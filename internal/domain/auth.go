package domain

import "context"

// AuthUsecase defines the contract for authentication business logic
type AuthUsecase interface {
	Register(ctx context.Context, username, email, password string, roleID int) (*User, error)
	// Login returns (accessToken, refreshToken, error)
	Login(ctx context.Context, email, password string) (string, string, error) 
	// LoginSSO returns (accessToken, refreshToken, error) based on email only
	LoginSSO(ctx context.Context, email string) (string, string, error)
	// RefreshToken returns (newAccessToken, newRefreshToken, error)
	RefreshToken(ctx context.Context, refreshToken string) (string, string, error)
}
