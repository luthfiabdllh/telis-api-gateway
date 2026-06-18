package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Role Entity
type Role struct {
	ID   int    `gorm:"primaryKey" json:"id"`
	Name string `json:"name"`
}

// User Entity
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex" json:"username"`
	Email        string    `gorm:"uniqueIndex" json:"email"`
	PasswordHash string    `json:"-"`
	RoleID       int       `json:"role_id"`
	Role         Role      `gorm:"foreignKey:RoleID" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName overrides the table name used by GORM (gateway schema)
func (User) TableName() string {
	return "gateway.users"
}

func (Role) TableName() string {
	return "gateway.roles"
}

// UserRepository defines the contract for User DB operations
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
}
