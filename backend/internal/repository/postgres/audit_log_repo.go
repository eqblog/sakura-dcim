package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
)

type AuditLogRepo struct {
	db *pgxpool.Pool
}

func NewAuditLogRepo(db *pgxpool.Pool) *AuditLogRepo {
	return &AuditLogRepo{db: db}
}

func (r *AuditLogRepo) Create(ctx context.Context, log *domain.AuditLog) error {
	detailsJSON, _ := json.Marshal(log.Details)
	query := `INSERT INTO audit_logs (id, tenant_id, user_id, action, resource_type, resource_id, details, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.db.Exec(ctx, query,
		log.ID, log.TenantID, log.UserID, log.Action,
		log.ResourceType, log.ResourceID, detailsJSON,
		log.IPAddress, log.UserAgent, time.Now())
	return err
}

func (r *AuditLogRepo) List(ctx context.Context, params domain.AuditLogListParams) (*domain.PaginatedResult[domain.AuditLog], error) {
	var conditions []string
	var args []any
	argIdx := 1

	if params.TenantID != nil {
		conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
		args = append(args, *params.TenantID)
		argIdx++
	}
	if params.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
		args = append(args, *params.UserID)
		argIdx++
	}
	if params.Action != "" {
		conditions = append(conditions, fmt.Sprintf("action ILIKE $%d", argIdx))
		args = append(args, "%"+params.Action+"%")
		argIdx++
	}
	if params.StartTime != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *params.StartTime)
		argIdx++
	}
	if params.EndTime != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *params.EndTime)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int64
	_ = r.db.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", where), args...).Scan(&total)

	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}
	offset := (params.Page - 1) * params.PageSize

	query := fmt.Sprintf(`SELECT id, tenant_id, user_id, action, resource_type, resource_id, details, ip_address, user_agent, created_at
		FROM audit_logs %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, params.PageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.AuditLog, error) {
		var l domain.AuditLog
		var detailsJSON []byte
		err := row.Scan(
			&l.ID, &l.TenantID, &l.UserID, &l.Action, &l.ResourceType,
			&l.ResourceID, &detailsJSON, &l.IPAddress, &l.UserAgent, &l.CreatedAt,
		)
		if err == nil {
			_ = json.Unmarshal(detailsJSON, &l.Details)
		}
		return l, err
	})
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	return &domain.PaginatedResult[domain.AuditLog]{
		Items:      logs,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}
