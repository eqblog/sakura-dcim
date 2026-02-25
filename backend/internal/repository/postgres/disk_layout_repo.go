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

type DiskLayoutRepo struct {
	db *pgxpool.Pool
}

func NewDiskLayoutRepo(db *pgxpool.Pool) *DiskLayoutRepo {
	return &DiskLayoutRepo{db: db}
}

func (r *DiskLayoutRepo) Create(ctx context.Context, layout *domain.DiskLayout) error {
	layout.ID = uuid.New()
	layout.CreatedAt = time.Now()

	layoutJSON, err := json.Marshal(layout.Layout)
	if err != nil {
		return fmt.Errorf("marshal layout: %w", err)
	}

	query := `INSERT INTO disk_layouts (id, name, description, layout, tags, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = r.db.Exec(ctx, query,
		layout.ID, layout.Name, layout.Description, layoutJSON, layout.Tags, layout.CreatedAt)
	return err
}

func (r *DiskLayoutRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.DiskLayout, error) {
	query := `SELECT id, name, description, layout, tags, created_at FROM disk_layouts WHERE id = $1`

	var d domain.DiskLayout
	var layoutRaw []byte
	err := r.db.QueryRow(ctx, query, id).Scan(
		&d.ID, &d.Name, &d.Description, &layoutRaw, &d.Tags, &d.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get disk_layout: %w", err)
	}
	if layoutRaw != nil {
		_ = json.Unmarshal(layoutRaw, &d.Layout)
	}
	return &d, nil
}

func (r *DiskLayoutRepo) List(ctx context.Context) ([]domain.DiskLayout, error) {
	query := `SELECT id, name, description, layout, tags, created_at FROM disk_layouts ORDER BY name`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.DiskLayout, error) {
		var d domain.DiskLayout
		var layoutRaw []byte
		err := row.Scan(&d.ID, &d.Name, &d.Description, &layoutRaw, &d.Tags, &d.CreatedAt)
		if err == nil && layoutRaw != nil {
			_ = json.Unmarshal(layoutRaw, &d.Layout)
		}
		return d, err
	})
}

func (r *DiskLayoutRepo) Update(ctx context.Context, layout *domain.DiskLayout) error {
	layoutJSON, err := json.Marshal(layout.Layout)
	if err != nil {
		return fmt.Errorf("marshal layout: %w", err)
	}
	query := `UPDATE disk_layouts SET name=$2, description=$3, layout=$4, tags=$5 WHERE id=$1`
	_, err = r.db.Exec(ctx, query,
		layout.ID, layout.Name, layout.Description, layoutJSON, layout.Tags)
	return err
}

func (r *DiskLayoutRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM disk_layouts WHERE id = $1`, id)
	return err
}
