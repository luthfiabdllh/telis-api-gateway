package domain

import "context"

// AuthUsecase defines the contract for authentication business logic
type AuthUsecase interface {
	Register(ctx context.Context, username, email, password string, roleID int) (*User, error)
	// Login returns (accessToken, refreshToken, error)
	Login(ctx context.Context, email, password string) (string, string, error) 
}
