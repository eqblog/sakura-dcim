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

type TenantRepo struct {
	db *pgxpool.Pool
}

func NewTenantRepo(db *pgxpool.Pool) *TenantRepo {
	return &TenantRepo{db: db}
}

func (r *TenantRepo) Create(ctx context.Context, tenant *domain.Tenant) error {
	now := time.Now()
	query := `INSERT INTO tenants (id, parent_id, name, slug, custom_domain, logo_url, primary_color, favicon_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)`
	_, err := r.db.Exec(ctx, query,
		tenant.ID, tenant.ParentID, tenant.Name, tenant.Slug,
		tenant.CustomDomain, tenant.LogoURL, tenant.PrimaryColor, tenant.FaviconURL, now)
	if err == nil {
		tenant.CreatedAt = now
		tenant.UpdatedAt = now
	}
	return err
}

func (r *TenantRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	query := `SELECT id, parent_id, name, slug, custom_domain, logo_url, primary_color, favicon_url, created_at, updated_at
		FROM tenants WHERE id = $1`

	var t domain.Tenant
	err := r.db.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.ParentID, &t.Name, &t.Slug,
		&t.CustomDomain, &t.LogoURL, &t.PrimaryColor, &t.FaviconURL,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get tenant: %w", err)
	}
	return &t, nil
}

func (r *TenantRepo) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	query := `SELECT id, parent_id, name, slug, custom_domain, logo_url, primary_color, favicon_url, created_at, updated_at
		FROM tenants WHERE slug = $1`

	var t domain.Tenant
	err := r.db.QueryRow(ctx, query, slug).Scan(
		&t.ID, &t.ParentID, &t.Name, &t.Slug,
		&t.CustomDomain, &t.LogoURL, &t.PrimaryColor, &t.FaviconURL,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get tenant by slug: %w", err)
	}
	return &t, nil
}

func (r *TenantRepo) GetByDomain(ctx context.Context, customDomain string) (*domain.Tenant, error) {
	query := `SELECT id, parent_id, name, slug, custom_domain, logo_url, primary_color, favicon_url, created_at, updated_at
		FROM tenants WHERE custom_domain = $1`

	var t domain.Tenant
	err := r.db.QueryRow(ctx, query, customDomain).Scan(
		&t.ID, &t.ParentID, &t.Name, &t.Slug,
		&t.CustomDomain, &t.LogoURL, &t.PrimaryColor, &t.FaviconURL,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get tenant by domain: %w", err)
	}
	return &t, nil
}

func (r *TenantRepo) List(ctx context.Context, parentID *uuid.UUID, page, pageSize int) (*domain.PaginatedResult[domain.Tenant], error) {
	var total int64
	var countQuery string
	var args []any

	if parentID != nil {
		countQuery = `SELECT COUNT(*) FROM tenants WHERE parent_id = $1`
		args = []any{*parentID}
	} else {
		countQuery = `SELECT COUNT(*) FROM tenants`
	}
	_ = r.db.QueryRow(ctx, countQuery, args...).Scan(&total)

	offset := (page - 1) * pageSize
	var query string
	if parentID != nil {
		query = `SELECT id, parent_id, name, slug, custom_domain, logo_url, primary_color, favicon_url, created_at, updated_at
			FROM tenants WHERE parent_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		args = []any{*parentID, pageSize, offset}
	} else {
		query = `SELECT id, parent_id, name, slug, custom_domain, logo_url, primary_color, favicon_url, created_at, updated_at
			FROM tenants ORDER BY created_at DESC LIMIT $1 OFFSET $2`
		args = []any{pageSize, offset}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tenants, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Tenant, error) {
		var t domain.Tenant
		err := row.Scan(
			&t.ID, &t.ParentID, &t.Name, &t.Slug,
			&t.CustomDomain, &t.LogoURL, &t.PrimaryColor, &t.FaviconURL,
			&t.CreatedAt, &t.UpdatedAt,
		)
		return t, err
	})
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &domain.PaginatedResult[domain.Tenant]{
		Items:      tenants,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (r *TenantRepo) Update(ctx context.Context, tenant *domain.Tenant) error {
	now := time.Now()
	query := `UPDATE tenants SET name = $2, slug = $3, custom_domain = $4, logo_url = $5, primary_color = $6, favicon_url = $7, updated_at = $8
		WHERE id = $1`
	_, err := r.db.Exec(ctx, query,
		tenant.ID, tenant.Name, tenant.Slug,
		tenant.CustomDomain, tenant.LogoURL, tenant.PrimaryColor, tenant.FaviconURL, now)
	if err == nil {
		tenant.UpdatedAt = now
	}
	return err
}

func (r *TenantRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, id)
	return err
}
