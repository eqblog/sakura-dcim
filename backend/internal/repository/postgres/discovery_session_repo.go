package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
)

type DiscoverySessionRepo struct {
	db *pgxpool.Pool
}

func NewDiscoverySessionRepo(db *pgxpool.Pool) *DiscoverySessionRepo {
	return &DiscoverySessionRepo{db: db}
}

func (r *DiscoverySessionRepo) Create(ctx context.Context, s *domain.DiscoverySession) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO discovery_sessions (agent_id, status, callback_token, dhcp_range, started_by)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, started_at`,
		s.AgentID, s.Status, s.CallbackToken, s.DHCPRange, s.StartedBy,
	).Scan(&s.ID, &s.StartedAt)
}

func (r *DiscoverySessionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.DiscoverySession, error) {
	var s domain.DiscoverySession
	err := r.db.QueryRow(ctx,
		`SELECT id, agent_id, status, callback_token, dhcp_range, started_by, started_at, stopped_at
		 FROM discovery_sessions WHERE id = $1`, id,
	).Scan(&s.ID, &s.AgentID, &s.Status, &s.CallbackToken, &s.DHCPRange, &s.StartedBy, &s.StartedAt, &s.StoppedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *DiscoverySessionRepo) GetActiveByAgentID(ctx context.Context, agentID uuid.UUID) (*domain.DiscoverySession, error) {
	var s domain.DiscoverySession
	err := r.db.QueryRow(ctx,
		`SELECT id, agent_id, status, callback_token, dhcp_range, started_by, started_at, stopped_at
		 FROM discovery_sessions WHERE agent_id = $1 AND status = 'active'
		 ORDER BY started_at DESC LIMIT 1`, agentID,
	).Scan(&s.ID, &s.AgentID, &s.Status, &s.CallbackToken, &s.DHCPRange, &s.StartedBy, &s.StartedAt, &s.StoppedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *DiscoverySessionRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.DiscoverySessionStatus) error {
	var stoppedAt *time.Time
	if status == domain.DiscoveryStatusStopped {
		now := time.Now()
		stoppedAt = &now
	}
	_, err := r.db.Exec(ctx,
		`UPDATE discovery_sessions SET status = $2, stopped_at = $3 WHERE id = $1`,
		id, status, stoppedAt)
	return err
}
