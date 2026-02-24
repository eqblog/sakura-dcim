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

type AgentRepo struct {
	db *pgxpool.Pool
}

func NewAgentRepo(db *pgxpool.Pool) *AgentRepo {
	return &AgentRepo{db: db}
}

func (r *AgentRepo) Create(ctx context.Context, agent *domain.Agent) error {
	capsJSON, _ := json.Marshal(agent.Capabilities)
	query := `INSERT INTO agents (id, name, location, token_hash, status, version, capabilities, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.Exec(ctx, query,
		agent.ID, agent.Name, agent.Location, agent.TokenHash,
		agent.Status, agent.Version, capsJSON, time.Now())
	return err
}

func (r *AgentRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error) {
	query := `SELECT id, name, location, token_hash, status, last_seen, version, capabilities, created_at
		FROM agents WHERE id = $1`

	var agent domain.Agent
	var capsJSON []byte
	err := r.db.QueryRow(ctx, query, id).Scan(
		&agent.ID, &agent.Name, &agent.Location, &agent.TokenHash,
		&agent.Status, &agent.LastSeen, &agent.Version, &capsJSON, &agent.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}
	_ = json.Unmarshal(capsJSON, &agent.Capabilities)
	return &agent, nil
}

func (r *AgentRepo) List(ctx context.Context, page, pageSize int) (*domain.PaginatedResult[domain.Agent], error) {
	var total int64
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM agents`).Scan(&total)

	offset := (page - 1) * pageSize
	query := `SELECT id, name, location, status, last_seen, version, capabilities, created_at
		FROM agents ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, query, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	agents, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Agent, error) {
		var a domain.Agent
		var capsJSON []byte
		err := row.Scan(&a.ID, &a.Name, &a.Location, &a.Status, &a.LastSeen, &a.Version, &capsJSON, &a.CreatedAt)
		if err == nil {
			_ = json.Unmarshal(capsJSON, &a.Capabilities)
		}
		return a, err
	})
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &domain.PaginatedResult[domain.Agent]{
		Items:      agents,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (r *AgentRepo) Update(ctx context.Context, agent *domain.Agent) error {
	capsJSON, _ := json.Marshal(agent.Capabilities)
	_, err := r.db.Exec(ctx,
		`UPDATE agents SET name = $2, location = $3, version = $4, capabilities = $5 WHERE id = $1`,
		agent.ID, agent.Name, agent.Location, agent.Version, capsJSON)
	return err
}

func (r *AgentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM agents WHERE id = $1`, id)
	return err
}

func (r *AgentRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.AgentStatus) error {
	_, err := r.db.Exec(ctx, `UPDATE agents SET status = $2 WHERE id = $1`, id, status)
	return err
}

func (r *AgentRepo) UpdateLastSeen(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE agents SET last_seen = $2, status = 'online' WHERE id = $1`, id, time.Now())
	return err
}
