package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
)

type ServerRepo struct {
	db *pgxpool.Pool
}

func NewServerRepo(db *pgxpool.Pool) *ServerRepo {
	return &ServerRepo{db: db}
}

// inetOrNil converts an empty string to nil for PostgreSQL INET columns.
func inetOrNil(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func (r *ServerRepo) Create(ctx context.Context, server *domain.Server) error {
	query := `INSERT INTO servers (id, tenant_id, agent_id, hostname, label, status, primary_ip, ipmi_ip, ipmi_user, ipmi_pass, bmc_type, tags, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $14)`
	now := time.Now()
	_, err := r.db.Exec(ctx, query,
		server.ID, server.TenantID, server.AgentID, server.Hostname, server.Label,
		server.Status, inetOrNil(server.PrimaryIP), inetOrNil(server.IPMIIP), server.IPMIUser, server.IPMIPass,
		server.BMCType, server.Tags, server.Notes, now)
	if err == nil {
		server.CreatedAt = now
		server.UpdatedAt = now
	}
	return err
}

func (r *ServerRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Server, error) {
	query := `SELECT id, tenant_id, agent_id, hostname, label, status,
		COALESCE(primary_ip::text,''), COALESCE(ipmi_ip::text,''), ipmi_user, ipmi_pass, bmc_type,
		cpu_model, cpu_cores, ram_mb, tags, notes, created_at, updated_at
		FROM servers WHERE id = $1`

	var s domain.Server
	err := r.db.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.TenantID, &s.AgentID, &s.Hostname, &s.Label, &s.Status,
		&s.PrimaryIP, &s.IPMIIP, &s.IPMIUser, &s.IPMIPass, &s.BMCType,
		&s.CPUModel, &s.CPUCores, &s.RAMMB, &s.Tags, &s.Notes,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get server: %w", err)
	}
	return &s, nil
}

func (r *ServerRepo) List(ctx context.Context, params domain.ServerListParams) (*domain.PaginatedResult[domain.Server], error) {
	var conditions []string
	var args []any
	argIdx := 1

	if params.TenantID != nil {
		conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
		args = append(args, *params.TenantID)
		argIdx++
	}
	if params.AgentID != nil {
		conditions = append(conditions, fmt.Sprintf("agent_id = $%d", argIdx))
		args = append(args, *params.AgentID)
		argIdx++
	}
	if params.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *params.Status)
		argIdx++
	}
	if params.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(hostname ILIKE $%d OR label ILIKE $%d OR primary_ip::text ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+params.Search+"%")
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count
	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM servers %s", where)
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	// Fetch
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}
	offset := (params.Page - 1) * params.PageSize

	selectQuery := fmt.Sprintf(`SELECT id, tenant_id, agent_id, hostname, label, status,
		COALESCE(primary_ip::text,''), COALESCE(ipmi_ip::text,''), bmc_type,
		cpu_model, cpu_cores, ram_mb, tags, notes, created_at, updated_at
		FROM servers %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, params.PageSize, offset)

	rows, err := r.db.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	servers, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Server, error) {
		var s domain.Server
		err := row.Scan(
			&s.ID, &s.TenantID, &s.AgentID, &s.Hostname, &s.Label, &s.Status,
			&s.PrimaryIP, &s.IPMIIP, &s.BMCType, &s.CPUModel, &s.CPUCores, &s.RAMMB,
			&s.Tags, &s.Notes, &s.CreatedAt, &s.UpdatedAt,
		)
		return s, err
	})
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	return &domain.PaginatedResult[domain.Server]{
		Items:      servers,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (r *ServerRepo) Update(ctx context.Context, server *domain.Server) error {
	query := `UPDATE servers SET hostname = $2, label = $3, agent_id = $4, primary_ip = $5,
		ipmi_ip = $6, ipmi_user = $7, ipmi_pass = $8, bmc_type = $9, tags = $10, notes = $11, updated_at = $12
		WHERE id = $1`
	now := time.Now()
	_, err := r.db.Exec(ctx, query,
		server.ID, server.Hostname, server.Label, server.AgentID,
		inetOrNil(server.PrimaryIP), inetOrNil(server.IPMIIP), server.IPMIUser, server.IPMIPass,
		server.BMCType, server.Tags, server.Notes, now)
	if err == nil {
		server.UpdatedAt = now
	}
	return err
}

func (r *ServerRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM servers WHERE id = $1`, id)
	return err
}

func (r *ServerRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ServerStatus) error {
	_, err := r.db.Exec(ctx, `UPDATE servers SET status = $2, updated_at = $3 WHERE id = $1`, id, status, time.Now())
	return err
}
