package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
)

type IPPoolRepo struct {
	db *pgxpool.Pool
}

func NewIPPoolRepo(db *pgxpool.Pool) *IPPoolRepo {
	return &IPPoolRepo{db: db}
}

func (r *IPPoolRepo) Create(ctx context.Context, pool *domain.IPPool) error {
	if pool.Nameservers == nil {
		pool.Nameservers = []string{}
	}
	return r.db.QueryRow(ctx,
		`INSERT INTO ip_pools (id, tenant_id, network, gateway, netmask, vrf, nameservers, description)
		 VALUES (gen_random_uuid(), $1, $2::cidr, $3::inet, $4, $5, $6, $7)
		 RETURNING id`,
		pool.TenantID, pool.Network, pool.Gateway, pool.Netmask, pool.VRF, pool.Nameservers, pool.Description,
	).Scan(&pool.ID)
}

func (r *IPPoolRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.IPPool, error) {
	pool := &domain.IPPool{}
	err := r.db.QueryRow(ctx,
		`SELECT p.id, p.tenant_id, p.network::text, p.gateway::text, p.netmask, p.vrf, p.nameservers, p.description,
		        COALESCE((SELECT COUNT(*) FROM ip_addresses WHERE pool_id = p.id), 0) AS total_ips,
		        COALESCE((SELECT COUNT(*) FROM ip_addresses WHERE pool_id = p.id AND status != 'available'), 0) AS used_ips
		 FROM ip_pools p WHERE p.id = $1`, id,
	).Scan(&pool.ID, &pool.TenantID, &pool.Network, &pool.Gateway, &pool.Netmask, &pool.VRF, &pool.Nameservers, &pool.Description, &pool.TotalIPs, &pool.UsedIPs)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func (r *IPPoolRepo) List(ctx context.Context, tenantID *uuid.UUID) ([]domain.IPPool, error) {
	query := `SELECT p.id, p.tenant_id, p.network::text, p.gateway::text, p.netmask, p.vrf, p.nameservers, p.description,
	                  COALESCE((SELECT COUNT(*) FROM ip_addresses WHERE pool_id = p.id), 0) AS total_ips,
	                  COALESCE((SELECT COUNT(*) FROM ip_addresses WHERE pool_id = p.id AND status != 'available'), 0) AS used_ips
	           FROM ip_pools p`
	args := []any{}
	if tenantID != nil {
		query += ` WHERE p.tenant_id = $1`
		args = append(args, *tenantID)
	}
	query += ` ORDER BY p.network`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pools []domain.IPPool
	for rows.Next() {
		var p domain.IPPool
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Network, &p.Gateway, &p.Netmask, &p.VRF, &p.Nameservers, &p.Description, &p.TotalIPs, &p.UsedIPs); err != nil {
			return nil, err
		}
		pools = append(pools, p)
	}
	return pools, rows.Err()
}

func (r *IPPoolRepo) Update(ctx context.Context, pool *domain.IPPool) error {
	if pool.Nameservers == nil {
		pool.Nameservers = []string{}
	}
	_, err := r.db.Exec(ctx,
		`UPDATE ip_pools SET network = $2::cidr, gateway = $3::inet, netmask = $4, vrf = $5, nameservers = $6, description = $7, tenant_id = $8
		 WHERE id = $1`,
		pool.ID, pool.Network, pool.Gateway, pool.Netmask, pool.VRF, pool.Nameservers, pool.Description, pool.TenantID)
	return err
}

func (r *IPPoolRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM ip_pools WHERE id = $1`, id)
	return err
}
