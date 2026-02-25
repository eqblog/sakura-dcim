package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
)

type DiscoveredServerRepo struct {
	db *pgxpool.Pool
}

func NewDiscoveredServerRepo(db *pgxpool.Pool) *DiscoveredServerRepo {
	return &DiscoveredServerRepo{db: db}
}

func (r *DiscoveredServerRepo) Upsert(ctx context.Context, ds *domain.DiscoveredServer) error {
	rawJSON, err := json.Marshal(ds.RawInventory)
	if err != nil {
		rawJSON = []byte("{}")
	}

	return r.db.QueryRow(ctx,
		`INSERT INTO discovered_servers (
			session_id, agent_id, mac_address, ip_address, status,
			system_vendor, system_product, system_serial,
			cpu_model, cpu_cores, cpu_sockets, ram_mb,
			disk_count, disk_total_gb, nic_count,
			raw_inventory, bmc_ip
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		ON CONFLICT (session_id, mac_address) DO UPDATE SET
			ip_address = EXCLUDED.ip_address,
			system_vendor = EXCLUDED.system_vendor,
			system_product = EXCLUDED.system_product,
			system_serial = EXCLUDED.system_serial,
			cpu_model = EXCLUDED.cpu_model,
			cpu_cores = EXCLUDED.cpu_cores,
			cpu_sockets = EXCLUDED.cpu_sockets,
			ram_mb = EXCLUDED.ram_mb,
			disk_count = EXCLUDED.disk_count,
			disk_total_gb = EXCLUDED.disk_total_gb,
			nic_count = EXCLUDED.nic_count,
			raw_inventory = EXCLUDED.raw_inventory,
			bmc_ip = EXCLUDED.bmc_ip,
			updated_at = NOW()
		RETURNING id, discovered_at, updated_at`,
		ds.SessionID, ds.AgentID, ds.MACAddress, ds.IPAddress, ds.Status,
		ds.SystemVendor, ds.SystemProduct, ds.SystemSerial,
		ds.CPUModel, ds.CPUCores, ds.CPUSockets, ds.RAMMB,
		ds.DiskCount, ds.DiskTotalGB, ds.NICCount,
		rawJSON, ds.BMCIP,
	).Scan(&ds.ID, &ds.DiscoveredAt, &ds.UpdatedAt)
}

func (r *DiscoveredServerRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.DiscoveredServer, error) {
	var ds domain.DiscoveredServer
	var rawJSON []byte
	err := r.db.QueryRow(ctx,
		`SELECT id, session_id, agent_id, mac_address, ip_address, status,
			system_vendor, system_product, system_serial,
			cpu_model, cpu_cores, cpu_sockets, ram_mb,
			disk_count, disk_total_gb, nic_count,
			raw_inventory, bmc_ip, approved_by, server_id,
			discovered_at, updated_at
		 FROM discovered_servers WHERE id = $1`, id,
	).Scan(
		&ds.ID, &ds.SessionID, &ds.AgentID, &ds.MACAddress, &ds.IPAddress, &ds.Status,
		&ds.SystemVendor, &ds.SystemProduct, &ds.SystemSerial,
		&ds.CPUModel, &ds.CPUCores, &ds.CPUSockets, &ds.RAMMB,
		&ds.DiskCount, &ds.DiskTotalGB, &ds.NICCount,
		&rawJSON, &ds.BMCIP, &ds.ApprovedBy, &ds.ServerID,
		&ds.DiscoveredAt, &ds.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	var inv interface{}
	if json.Unmarshal(rawJSON, &inv) == nil {
		ds.RawInventory = inv
	}
	return &ds, nil
}

func (r *DiscoveredServerRepo) List(ctx context.Context, params domain.DiscoveredServerListParams) (*domain.PaginatedResult[domain.DiscoveredServer], error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}

	where := "WHERE 1=1"
	args := []interface{}{}
	argN := 1

	if params.AgentID != nil {
		where += fmt.Sprintf(" AND agent_id = $%d", argN)
		args = append(args, *params.AgentID)
		argN++
	}
	if params.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", argN)
		args = append(args, string(*params.Status))
		argN++
	}
	if params.Search != "" {
		where += fmt.Sprintf(" AND (mac_address ILIKE $%d OR system_vendor ILIKE $%d OR system_product ILIKE $%d OR system_serial ILIKE $%d)", argN, argN, argN, argN)
		args = append(args, "%"+params.Search+"%")
		argN++
	}

	var total int64
	countQuery := "SELECT COUNT(*) FROM discovered_servers " + where
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	offset := (params.Page - 1) * params.PageSize
	query := fmt.Sprintf(
		`SELECT id, session_id, agent_id, mac_address, ip_address, status,
			system_vendor, system_product, system_serial,
			cpu_model, cpu_cores, cpu_sockets, ram_mb,
			disk_count, disk_total_gb, nic_count,
			raw_inventory, bmc_ip, approved_by, server_id,
			discovered_at, updated_at
		 FROM discovered_servers %s
		 ORDER BY discovered_at DESC
		 LIMIT $%d OFFSET $%d`, where, argN, argN+1)
	args = append(args, params.PageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.DiscoveredServer
	for rows.Next() {
		var ds domain.DiscoveredServer
		var rawJSON []byte
		if err := rows.Scan(
			&ds.ID, &ds.SessionID, &ds.AgentID, &ds.MACAddress, &ds.IPAddress, &ds.Status,
			&ds.SystemVendor, &ds.SystemProduct, &ds.SystemSerial,
			&ds.CPUModel, &ds.CPUCores, &ds.CPUSockets, &ds.RAMMB,
			&ds.DiskCount, &ds.DiskTotalGB, &ds.NICCount,
			&rawJSON, &ds.BMCIP, &ds.ApprovedBy, &ds.ServerID,
			&ds.DiscoveredAt, &ds.UpdatedAt,
		); err != nil {
			return nil, err
		}
		var inv interface{}
		if json.Unmarshal(rawJSON, &inv) == nil {
			ds.RawInventory = inv
		}
		items = append(items, ds)
	}

	return &domain.PaginatedResult[domain.DiscoveredServer]{
		Items:      items,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: int(math.Ceil(float64(total) / float64(params.PageSize))),
	}, rows.Err()
}

func (r *DiscoveredServerRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.DiscoveredServerStatus) error {
	_, err := r.db.Exec(ctx,
		`UPDATE discovered_servers SET status = $2, updated_at = NOW() WHERE id = $1`,
		id, status)
	return err
}

func (r *DiscoveredServerRepo) SetServerID(ctx context.Context, id uuid.UUID, serverID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE discovered_servers SET server_id = $2, updated_at = NOW() WHERE id = $1`,
		id, serverID)
	return err
}

func (r *DiscoveredServerRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM discovered_servers WHERE id = $1`, id)
	return err
}
