package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
)

type RoleRepo struct {
	db *pgxpool.Pool
}

func NewRoleRepo(db *pgxpool.Pool) *RoleRepo {
	return &RoleRepo{db: db}
}

func (r *RoleRepo) Create(ctx context.Context, role *domain.Role) error {
	permsJSON, err := json.Marshal(role.Permissions)
	if err != nil {
		return fmt.Errorf("marshal permissions: %w", err)
	}

	query := `INSERT INTO roles (id, tenant_id, name, permissions, is_system, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = r.db.Exec(ctx, query,
		role.ID, role.TenantID, role.Name, permsJSON, role.IsSystem, time.Now())
	return err
}

func (r *RoleRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	query := `SELECT id, tenant_id, name, permissions, is_system, created_at FROM roles WHERE id = $1`

	var role domain.Role
	var permsJSON []byte
	err := r.db.QueryRow(ctx, query, id).Scan(
		&role.ID, &role.TenantID, &role.Name, &permsJSON, &role.IsSystem, &role.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get role: %w", err)
	}
	_ = json.Unmarshal(permsJSON, &role.Permissions)
	return &role, nil
}

func (r *RoleRepo) List(ctx context.Context, tenantID *uuid.UUID) ([]domain.Role, error) {
	query := `SELECT id, tenant_id, name, permissions, is_system, created_at FROM roles
		WHERE tenant_id IS NULL OR tenant_id = $1 ORDER BY is_system DESC, name`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Role, error) {
		var role domain.Role
		var permsJSON []byte
		err := row.Scan(&role.ID, &role.TenantID, &role.Name, &permsJSON, &role.IsSystem, &role.CreatedAt)
		if err == nil {
			_ = json.Unmarshal(permsJSON, &role.Permissions)
		}
		return role, err
	})
}

func (r *RoleRepo) Update(ctx context.Context, role *domain.Role) error {
	permsJSON, err := json.Marshal(role.Permissions)
	if err != nil {
		return fmt.Errorf("marshal permissions: %w", err)
	}
	_, err = r.db.Exec(ctx, `UPDATE roles SET name = $2, permissions = $3 WHERE id = $1`,
		role.ID, role.Name, permsJSON)
	return err
}

func (r *RoleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM roles WHERE id = $1 AND is_system = false`, id)
	return err
}
