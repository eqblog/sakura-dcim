package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	TenantID     uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Email        string     `json:"email" db:"email"`
	PasswordHash string     `json:"-" db:"password_hash"`
	Name         string     `json:"name" db:"name"`
	RoleID       *uuid.UUID `json:"role_id,omitempty" db:"role_id"`
	IsActive     bool       `json:"is_active" db:"is_active"`
	LastLogin    *time.Time `json:"last_login,omitempty" db:"last_login"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`

	// Joined fields
	Role   *Role   `json:"role,omitempty" db:"-"`
	Tenant *Tenant `json:"tenant,omitempty" db:"-"`
}

type UserLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type UserCreateRequest struct {
	Email    string     `json:"email" binding:"required,email"`
	Password string     `json:"password" binding:"required,min=8"`
	Name     string     `json:"name" binding:"required"`
	RoleID   *uuid.UUID `json:"role_id"`
}

type UserUpdateRequest struct {
	Email    *string    `json:"email" binding:"omitempty,email"`
	Password *string    `json:"password" binding:"omitempty,min=8"`
	Name     *string    `json:"name"`
	RoleID   *uuid.UUID `json:"role_id"`
	IsActive *bool      `json:"is_active"`
}
