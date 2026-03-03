package domain

import (
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	ParentID     *uuid.UUID `json:"parent_id,omitempty" db:"parent_id"`
	Name         string     `json:"name" db:"name"`
	Slug         string     `json:"slug" db:"slug"`
	CustomDomain *string    `json:"custom_domain,omitempty" db:"custom_domain"`
	LogoURL      *string    `json:"logo_url,omitempty" db:"logo_url"`
	PrimaryColor *string    `json:"primary_color,omitempty" db:"primary_color"`
	FaviconURL   *string    `json:"favicon_url,omitempty" db:"favicon_url"`
	KvmMode      string     `json:"kvm_mode" db:"kvm_mode"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}
