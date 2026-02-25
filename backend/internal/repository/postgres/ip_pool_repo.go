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

const ipPoolSelectFields = `p.id, p.tenant_id, p.network::text, host(p.gateway), p.netmask, p.vrf, p.nameservers, p.description,
	p.priority, p.rdns_server, p.notes, p.switch_automation, p.vlan_id, p.vlan_range_start, p.vlan_range_end,
	p.vlan_mode, p.native_vlan_id, p.trunk_vlans,
	p.parent_id, p.pool_type,
	COALESCE((SELECT COUNT(*) FROM ip_addresses WHERE pool_id = p.id), 0),
	COALESCE((SELECT COUNT(*) FROM ip_addresses WHERE pool_id = p.id AND status != 'available'), 0),
	COALESCE((SELECT COUNT(*) FROM ip_pools c WHERE c.parent_id = p.id), 0)`

func scanIPPool(scan func(dest ...any) error) (*domain.IPPool, error) {
	p := &domain.IPPool{}
	err := scan(&p.ID, &p.TenantID, &p.Network, &p.Gateway, &p.Netmask, &p.VRF, &p.Nameservers, &p.Description,
		&p.Priority, &p.RDNSServer, &p.Notes, &p.SwitchAutomation, &p.VlanID, &p.VlanRangeStart, &p.VlanRangeEnd,
		&p.VlanMode, &p.NativeVlanID, &p.TrunkVlans,
		&p.ParentID, &p.PoolType,
		&p.TotalIPs, &p.UsedIPs, &p.ChildCount)
	return p, err
}

func (r *IPPoolRepo) Create(ctx context.Context, pool *domain.IPPool) error {
	if pool.Nameservers == nil {
		pool.Nameservers = []string{}
	}
	if pool.PoolType == "" {
		pool.PoolType = "ip_pool"
	}
	return r.db.QueryRow(ctx,
		`INSERT INTO ip_pools (id, tenant_id, network, gateway, netmask, vrf, nameservers, description,
		                       priority, rdns_server, notes, switch_automation, vlan_id, vlan_range_start, vlan_range_end,
		                       vlan_mode, native_vlan_id, trunk_vlans,
		                       parent_id, pool_type)
		 VALUES (gen_random_uuid(), $1, $2::cidr, $3::inet, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		 RETURNING id`,
		pool.TenantID, pool.Network, pool.Gateway, pool.Netmask, pool.VRF, pool.Nameservers, pool.Description,
		pool.Priority, pool.RDNSServer, pool.Notes, pool.SwitchAutomation, pool.VlanID, pool.VlanRangeStart, pool.VlanRangeEnd,
		pool.VlanMode, pool.NativeVlanID, pool.TrunkVlans,
		pool.ParentID, pool.PoolType,
	).Scan(&pool.ID)
}

func (r *IPPoolRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.IPPool, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+ipPoolSelectFields+` FROM ip_pools p WHERE p.id = $1`, id)
	return scanIPPool(row.Scan)
}

func (r *IPPoolRepo) List(ctx context.Context, tenantID *uuid.UUID) ([]domain.IPPool, error) {
	query := `SELECT ` + ipPoolSelectFields + ` FROM ip_pools p WHERE p.parent_id IS NULL`
	args := []any{}
	if tenantID != nil {
		query += ` AND p.tenant_id = $1`
		args = append(args, *tenantID)
	}
	query += ` ORDER BY p.network`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pools := make([]domain.IPPool, 0)
	for rows.Next() {
		p, err := scanIPPool(rows.Scan)
		if err != nil {
			return nil, err
		}
		pools = append(pools, *p)
	}
	return pools, rows.Err()
}

func (r *IPPoolRepo) ListByParentID(ctx context.Context, parentID uuid.UUID) ([]domain.IPPool, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+ipPoolSelectFields+` FROM ip_pools p WHERE p.parent_id = $1 ORDER BY p.network`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pools := make([]domain.IPPool, 0)
	for rows.Next() {
		p, err := scanIPPool(rows.Scan)
		if err != nil {
			return nil, err
		}
		pools = append(pools, *p)
	}
	return pools, rows.Err()
}

func (r *IPPoolRepo) Update(ctx context.Context, pool *domain.IPPool) error {
	if pool.Nameservers == nil {
		pool.Nameservers = []string{}
	}
	_, err := r.db.Exec(ctx,
		`UPDATE ip_pools SET network = $2::cidr, gateway = $3::inet, netmask = $4, vrf = $5, nameservers = $6, description = $7, tenant_id = $8,
		        priority = $9, rdns_server = $10, notes = $11, switch_automation = $12, vlan_id = $13, vlan_range_start = $14, vlan_range_end = $15,
		        vlan_mode = $16, native_vlan_id = $17, trunk_vlans = $18,
		        pool_type = $19
		 WHERE id = $1`,
		pool.ID, pool.Network, pool.Gateway, pool.Netmask, pool.VRF, pool.Nameservers, pool.Description, pool.TenantID,
		pool.Priority, pool.RDNSServer, pool.Notes, pool.SwitchAutomation, pool.VlanID, pool.VlanRangeStart, pool.VlanRangeEnd,
		pool.VlanMode, pool.NativeVlanID, pool.TrunkVlans,
		pool.PoolType)
	return err
}

func (r *IPPoolRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM ip_pools WHERE id = $1`, id)
	return err
}
