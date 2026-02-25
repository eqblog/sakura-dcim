package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
)

type InstallTaskRepo struct {
	db *pgxpool.Pool
}

func NewInstallTaskRepo(db *pgxpool.Pool) *InstallTaskRepo {
	return &InstallTaskRepo{db: db}
}

func (r *InstallTaskRepo) Create(ctx context.Context, task *domain.InstallTask) error {
	task.ID = uuid.New()
	task.CreatedAt = time.Now()

	query := `INSERT INTO install_tasks (id, server_id, os_profile_id, disk_layout_id, raid_level, status, root_password_hash, ssh_keys, progress, log, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err := r.db.Exec(ctx, query,
		task.ID, task.ServerID, task.OSProfileID, task.DiskLayoutID,
		task.RAIDLevel, task.Status, task.RootPasswordHash,
		task.SSHKeys, task.Progress, task.Log, task.CreatedAt)
	return err
}

func (r *InstallTaskRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.InstallTask, error) {
	query := `SELECT id, server_id, os_profile_id, disk_layout_id, raid_level, status,
		root_password_hash, ssh_keys, progress, log, started_at, completed_at, created_at
		FROM install_tasks WHERE id = $1`

	var t domain.InstallTask
	err := r.db.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.ServerID, &t.OSProfileID, &t.DiskLayoutID,
		&t.RAIDLevel, &t.Status, &t.RootPasswordHash,
		&t.SSHKeys, &t.Progress, &t.Log, &t.StartedAt, &t.CompletedAt, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get install_task: %w", err)
	}
	return &t, nil
}

func (r *InstallTaskRepo) GetActiveByServerID(ctx context.Context, serverID uuid.UUID) (*domain.InstallTask, error) {
	query := `SELECT id, server_id, os_profile_id, disk_layout_id, raid_level, status,
		root_password_hash, ssh_keys, progress, log, started_at, completed_at, created_at
		FROM install_tasks WHERE server_id = $1
		ORDER BY created_at DESC LIMIT 1`

	var t domain.InstallTask
	err := r.db.QueryRow(ctx, query, serverID).Scan(
		&t.ID, &t.ServerID, &t.OSProfileID, &t.DiskLayoutID,
		&t.RAIDLevel, &t.Status, &t.RootPasswordHash,
		&t.SSHKeys, &t.Progress, &t.Log, &t.StartedAt, &t.CompletedAt, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get active install_task: %w", err)
	}
	return &t, nil
}

func (r *InstallTaskRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.InstallTaskStatus, progress int, log string) error {
	query := `UPDATE install_tasks SET status=$2, progress=$3, log=log || $4`
	args := []interface{}{id, status, progress, log}

	if status == domain.InstallStatusPXEBooting {
		now := time.Now()
		query += `, started_at=$5 WHERE id=$1`
		args = append(args, now)
	} else if status == domain.InstallStatusCompleted || status == domain.InstallStatusFailed {
		now := time.Now()
		query += `, completed_at=$5 WHERE id=$1`
		args = append(args, now)
	} else {
		query += ` WHERE id=$1`
	}

	_, err := r.db.Exec(ctx, query, args...)
	return err
}
