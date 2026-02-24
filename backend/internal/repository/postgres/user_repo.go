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

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *domain.User) error {
	query := `INSERT INTO users (id, tenant_id, email, password_hash, name, role_id, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.Exec(ctx, query,
		user.ID, user.TenantID, user.Email, user.PasswordHash,
		user.Name, user.RoleID, user.IsActive, time.Now())
	return err
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `SELECT u.id, u.tenant_id, u.email, u.password_hash, u.name, u.role_id, u.is_active, u.last_login, u.created_at,
		r.id as role_id_j, r.name as role_name, r.permissions as role_permissions, r.is_system as role_is_system
		FROM users u LEFT JOIN roles r ON u.role_id = r.id WHERE u.id = $1`

	var user domain.User
	var roleID *uuid.UUID
	var roleName *string
	var rolePerms []byte
	var roleIsSystem *bool

	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.TenantID, &user.Email, &user.PasswordHash,
		&user.Name, &user.RoleID, &user.IsActive, &user.LastLogin, &user.CreatedAt,
		&roleID, &roleName, &rolePerms, &roleIsSystem,
	)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	if roleID != nil {
		var perms []string
		_ = json.Unmarshal(rolePerms, &perms)
		user.Role = &domain.Role{
			ID:          *roleID,
			Name:        *roleName,
			Permissions: perms,
			IsSystem:    *roleIsSystem,
		}
	}

	return &user, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.User, error) {
	query := `SELECT id, tenant_id, email, password_hash, name, role_id, is_active, last_login, created_at
		FROM users WHERE tenant_id = $1 AND email = $2`

	var user domain.User
	err := r.db.QueryRow(ctx, query, tenantID, email).Scan(
		&user.ID, &user.TenantID, &user.Email, &user.PasswordHash,
		&user.Name, &user.RoleID, &user.IsActive, &user.LastLogin, &user.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &user, nil
}

func (r *UserRepo) GetByEmailAnyTenant(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, tenant_id, email, password_hash, name, role_id, is_active, last_login, created_at
		FROM users WHERE email = $1 AND is_active = true LIMIT 1`

	var user domain.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.TenantID, &user.Email, &user.PasswordHash,
		&user.Name, &user.RoleID, &user.IsActive, &user.LastLogin, &user.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &user, nil
}

func (r *UserRepo) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) (*domain.PaginatedResult[domain.User], error) {
	var total int64
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE tenant_id = $1`, tenantID).Scan(&total)
	if err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	query := `SELECT id, tenant_id, email, name, role_id, is_active, last_login, created_at
		FROM users WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, tenantID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.User, error) {
		var u domain.User
		err := row.Scan(&u.ID, &u.TenantID, &u.Email, &u.Name, &u.RoleID, &u.IsActive, &u.LastLogin, &u.CreatedAt)
		return u, err
	})
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &domain.PaginatedResult[domain.User]{
		Items:      users,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET password_hash = $2 WHERE id = $1`, id, passwordHash)
	return err
}

func (r *UserRepo) Update(ctx context.Context, user *domain.User) error {
	query := `UPDATE users SET email = $2, name = $3, role_id = $4, is_active = $5 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, user.ID, user.Email, user.Name, user.RoleID, user.IsActive)
	return err
}

func (r *UserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

func (r *UserRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET last_login = $2 WHERE id = $1`, id, time.Now())
	return err
}
