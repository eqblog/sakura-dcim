package domain

import (
	"time"

	"github.com/google/uuid"
)

type OSProfile struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	OSFamily     string    `json:"os_family" db:"os_family"`
	Version      string    `json:"version" db:"version"`
	Arch         string    `json:"arch" db:"arch"`
	KernelURL    string    `json:"kernel_url" db:"kernel_url"`
	InitrdURL    string    `json:"initrd_url" db:"initrd_url"`
	BootArgs     string    `json:"boot_args" db:"boot_args"`
	TemplateType string    `json:"template_type" db:"template_type"`
	Template     string    `json:"template" db:"template"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	Tags         []string  `json:"tags" db:"tags"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type DiskLayout struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Layout      any       `json:"layout" db:"layout"` // JSONB
	Tags        []string  `json:"tags" db:"tags"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type Script struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	Description  string    `json:"description" db:"description"`
	Content      string    `json:"content" db:"content"`
	RunOrder     int       `json:"run_order" db:"run_order"`
	OSProfileIDs []string  `json:"os_profile_ids" db:"os_profile_ids"`
	Tags         []string  `json:"tags" db:"tags"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type InstallTaskStatus string

const (
	InstallStatusPending     InstallTaskStatus = "pending"
	InstallStatusPXEBooting  InstallTaskStatus = "pxe_booting"
	InstallStatusInstalling  InstallTaskStatus = "installing"
	InstallStatusPostScripts InstallTaskStatus = "post_scripts"
	InstallStatusCompleted   InstallTaskStatus = "completed"
	InstallStatusFailed      InstallTaskStatus = "failed"
)

type InstallTask struct {
	ID               uuid.UUID         `json:"id" db:"id"`
	ServerID         uuid.UUID         `json:"server_id" db:"server_id"`
	OSProfileID      uuid.UUID         `json:"os_profile_id" db:"os_profile_id"`
	DiskLayoutID     *uuid.UUID        `json:"disk_layout_id,omitempty" db:"disk_layout_id"`
	RAIDLevel        string            `json:"raid_level" db:"raid_level"`
	Status           InstallTaskStatus `json:"status" db:"status"`
	RootPasswordHash string            `json:"-" db:"root_password_hash"`
	SSHKeys          []string          `json:"ssh_keys" db:"ssh_keys"`
	Progress         int               `json:"progress" db:"progress"`
	Log              string            `json:"log" db:"log"`
	StartedAt        *time.Time        `json:"started_at,omitempty" db:"started_at"`
	CompletedAt      *time.Time        `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt        time.Time         `json:"created_at" db:"created_at"`
}

type ReinstallRequest struct {
	OSProfileID  uuid.UUID  `json:"os_profile_id" binding:"required"`
	DiskLayoutID *uuid.UUID `json:"disk_layout_id"`
	RAIDLevel    string     `json:"raid_level"` // auto, raid1, raid5, raid10, none
	RootPassword string     `json:"root_password" binding:"required,min=8"`
	SSHKeys      []string   `json:"ssh_keys"`
}
