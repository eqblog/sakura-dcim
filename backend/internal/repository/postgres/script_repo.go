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

type ScriptRepo struct {
	db *pgxpool.Pool
}

func NewScriptRepo(db *pgxpool.Pool) *ScriptRepo {
	return &ScriptRepo{db: db}
}

func (r *ScriptRepo) Create(ctx context.Context, script *domain.Script) error {
	script.ID = uuid.New()
	script.CreatedAt = time.Now()

	query := `INSERT INTO scripts (id, name, description, content, run_order, os_profile_ids, tags, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.Exec(ctx, query,
		script.ID, script.Name, script.Description, script.Content,
		script.RunOrder, script.OSProfileIDs, script.Tags, script.CreatedAt)
	return err
}

func (r *ScriptRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Script, error) {
	query := `SELECT id, name, description, content, run_order, os_profile_ids, tags, created_at
		FROM scripts WHERE id = $1`

	var s domain.Script
	err := r.db.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.Name, &s.Description, &s.Content,
		&s.RunOrder, &s.OSProfileIDs, &s.Tags, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get script: %w", err)
	}
	return &s, nil
}

func (r *ScriptRepo) List(ctx context.Context) ([]domain.Script, error) {
	query := `SELECT id, name, description, content, run_order, os_profile_ids, tags, created_at
		FROM scripts ORDER BY run_order, name`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Script, error) {
		var s domain.Script
		err := row.Scan(&s.ID, &s.Name, &s.Description, &s.Content,
			&s.RunOrder, &s.OSProfileIDs, &s.Tags, &s.CreatedAt)
		return s, err
	})
}

func (r *ScriptRepo) ListByOSProfileID(ctx context.Context, osProfileID uuid.UUID) ([]domain.Script, error) {
	query := `SELECT id, name, description, content, run_order, os_profile_ids, tags, created_at
		FROM scripts WHERE $1 = ANY(os_profile_ids) ORDER BY run_order, name`

	rows, err := r.db.Query(ctx, query, osProfileID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Script, error) {
		var s domain.Script
		err := row.Scan(&s.ID, &s.Name, &s.Description, &s.Content,
			&s.RunOrder, &s.OSProfileIDs, &s.Tags, &s.CreatedAt)
		return s, err
	})
}

func (r *ScriptRepo) Update(ctx context.Context, script *domain.Script) error {
	query := `UPDATE scripts SET name=$2, description=$3, content=$4, run_order=$5, os_profile_ids=$6, tags=$7 WHERE id=$1`
	_, err := r.db.Exec(ctx, query,
		script.ID, script.Name, script.Description, script.Content,
		script.RunOrder, script.OSProfileIDs, script.Tags)
	return err
}

func (r *ScriptRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM scripts WHERE id = $1`, id)
	return err
}
