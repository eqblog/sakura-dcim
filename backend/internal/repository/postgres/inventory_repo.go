package postgres

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
)

type InventoryRepo struct {
	db *pgxpool.Pool
}

func NewInventoryRepo(db *pgxpool.Pool) *InventoryRepo {
	return &InventoryRepo{db: db}
}

func (r *InventoryRepo) Upsert(ctx context.Context, inv *domain.ServerInventory) error {
	detailsJSON, err := json.Marshal(inv.Details)
	if err != nil {
		return err
	}

	// Delete existing entry for this server+component, then insert fresh
	_, err = r.db.Exec(ctx,
		`DELETE FROM server_inventory WHERE server_id = $1 AND component = $2`,
		inv.ServerID, inv.Component)
	if err != nil {
		return err
	}

	return r.db.QueryRow(ctx,
		`INSERT INTO server_inventory (id, server_id, component, details, collected_at)
		 VALUES (gen_random_uuid(), $1, $2, $3, NOW())
		 RETURNING id, collected_at`,
		inv.ServerID, inv.Component, detailsJSON,
	).Scan(&inv.ID, &inv.CollectedAt)
}

func (r *InventoryRepo) ListByServerID(ctx context.Context, serverID uuid.UUID) ([]domain.ServerInventory, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, server_id, component, details, collected_at
		 FROM server_inventory WHERE server_id = $1
		 ORDER BY component`, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.ServerInventory
	for rows.Next() {
		var inv domain.ServerInventory
		var detailsRaw []byte
		if err := rows.Scan(&inv.ID, &inv.ServerID, &inv.Component, &detailsRaw, &inv.CollectedAt); err != nil {
			return nil, err
		}
		var details interface{}
		if json.Unmarshal(detailsRaw, &details) == nil {
			inv.Details = details
		}
		items = append(items, inv)
	}
	return items, rows.Err()
}

func (r *InventoryRepo) DeleteByServerID(ctx context.Context, serverID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM server_inventory WHERE server_id = $1`, serverID)
	return err
}
