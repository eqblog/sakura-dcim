package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
)

type IPAddressRepo struct {
	db *pgxpool.Pool
}

func NewIPAddressRepo(db *pgxpool.Pool) *IPAddressRepo {
	return &IPAddressRepo{db: db}
}

func (r *IPAddressRepo) Create(ctx context.Context, addr *domain.IPAddress) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO ip_addresses (id, pool_id, address, server_id, status, note)
		 VALUES (gen_random_uuid(), $1, $2::inet, $3, $4, $5)
		 RETURNING id`,
		addr.PoolID, addr.Address, addr.ServerID, addr.Status, addr.Note,
	).Scan(&addr.ID)
}

func (r *IPAddressRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.IPAddress, error) {
	addr := &domain.IPAddress{}
	err := r.db.QueryRow(ctx,
		`SELECT id, pool_id, host(address), server_id, status, note
		 FROM ip_addresses WHERE id = $1`, id,
	).Scan(&addr.ID, &addr.PoolID, &addr.Address, &addr.ServerID, &addr.Status, &addr.Note)
	if err != nil {
		return nil, err
	}
	return addr, nil
}

func (r *IPAddressRepo) ListByPoolID(ctx context.Context, poolID uuid.UUID) ([]domain.IPAddress, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, pool_id, host(address), server_id, status, note
		 FROM ip_addresses WHERE pool_id = $1
		 ORDER BY address`, poolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var addrs []domain.IPAddress
	for rows.Next() {
		var a domain.IPAddress
		if err := rows.Scan(&a.ID, &a.PoolID, &a.Address, &a.ServerID, &a.Status, &a.Note); err != nil {
			return nil, err
		}
		addrs = append(addrs, a)
	}
	return addrs, rows.Err()
}

func (r *IPAddressRepo) ListByServerID(ctx context.Context, serverID uuid.UUID) ([]domain.IPAddress, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, pool_id, host(address), server_id, status, note
		 FROM ip_addresses WHERE server_id = $1
		 ORDER BY address`, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var addrs []domain.IPAddress
	for rows.Next() {
		var a domain.IPAddress
		if err := rows.Scan(&a.ID, &a.PoolID, &a.Address, &a.ServerID, &a.Status, &a.Note); err != nil {
			return nil, err
		}
		addrs = append(addrs, a)
	}
	return addrs, rows.Err()
}

func (r *IPAddressRepo) CountAssignedByPoolAndServer(ctx context.Context, poolID uuid.UUID, serverID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM ip_addresses WHERE pool_id = $1 AND server_id = $2 AND status = 'assigned'`,
		poolID, serverID).Scan(&count)
	return count, err
}

func (r *IPAddressRepo) Update(ctx context.Context, addr *domain.IPAddress) error {
	_, err := r.db.Exec(ctx,
		`UPDATE ip_addresses SET server_id = $2, status = $3, note = $4
		 WHERE id = $1`,
		addr.ID, addr.ServerID, addr.Status, addr.Note)
	return err
}

func (r *IPAddressRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM ip_addresses WHERE id = $1`, id)
	return err
}

func (r *IPAddressRepo) GetNextAvailable(ctx context.Context, poolID uuid.UUID) (*domain.IPAddress, error) {
	addr := &domain.IPAddress{}
	err := r.db.QueryRow(ctx,
		`SELECT id, pool_id, host(address), server_id, status, note
		 FROM ip_addresses WHERE pool_id = $1 AND status = 'available'
		 ORDER BY address LIMIT 1`, poolID,
	).Scan(&addr.ID, &addr.PoolID, &addr.Address, &addr.ServerID, &addr.Status, &addr.Note)
	if err != nil {
		return nil, fmt.Errorf("no available IP in pool: %w", err)
	}
	return addr, nil
}
