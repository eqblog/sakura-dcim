package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
)

type OSProfileRepo struct {
	db *pgxpool.Pool
}

func NewOSProfileRepo(db *pgxpool.Pool) *OSProfileRepo {
	return &OSProfileRepo{db: db}
}

func (r *OSProfileRepo) Create(ctx context.Context, profile *domain.OSProfile) error {
	profile.ID = uuid.New()
	profile.CreatedAt = time.Now()

	query := `INSERT INTO os_profiles (id, name, os_family, version, arch, kernel_url, initrd_url, boot_args, template_type, template, is_active, tags, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	_, err := r.db.Exec(ctx, query,
		profile.ID, profile.Name, profile.OSFamily, profile.Version, profile.Arch,
		profile.KernelURL, profile.InitrdURL, profile.BootArgs,
		profile.TemplateType, profile.Template, profile.IsActive, profile.Tags, profile.CreatedAt)
	return err
}

func (r *OSProfileRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.OSProfile, error) {
	query := `SELECT id, name, os_family, version, arch, kernel_url, initrd_url, boot_args, template_type, template, is_active, tags, created_at
		FROM os_profiles WHERE id = $1`

	var p domain.OSProfile
	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.OSFamily, &p.Version, &p.Arch,
		&p.KernelURL, &p.InitrdURL, &p.BootArgs,
		&p.TemplateType, &p.Template, &p.IsActive, &p.Tags, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get os_profile: %w", err)
	}
	return &p, nil
}

func (r *OSProfileRepo) List(ctx context.Context, activeOnly bool) ([]domain.OSProfile, error) {
	query := `SELECT id, name, os_family, version, arch, kernel_url, initrd_url, boot_args, template_type, template, is_active, tags, created_at
		FROM os_profiles`
	if activeOnly {
		query += ` WHERE is_active = true`
	}
	query += ` ORDER BY os_family, name`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.OSProfile, error) {
		var p domain.OSProfile
		err := row.Scan(&p.ID, &p.Name, &p.OSFamily, &p.Version, &p.Arch,
			&p.KernelURL, &p.InitrdURL, &p.BootArgs,
			&p.TemplateType, &p.Template, &p.IsActive, &p.Tags, &p.CreatedAt)
		return p, err
	})
}

func (r *OSProfileRepo) Update(ctx context.Context, profile *domain.OSProfile) error {
	query := `UPDATE os_profiles SET name=$2, os_family=$3, version=$4, arch=$5,
		kernel_url=$6, initrd_url=$7, boot_args=$8, template_type=$9, template=$10,
		is_active=$11, tags=$12 WHERE id=$1`
	_, err := r.db.Exec(ctx, query,
		profile.ID, profile.Name, profile.OSFamily, profile.Version, profile.Arch,
		profile.KernelURL, profile.InitrdURL, profile.BootArgs,
		profile.TemplateType, profile.Template, profile.IsActive, profile.Tags)
	return err
}

func (r *OSProfileRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM os_profiles WHERE id = $1`, id)
	return err
}
