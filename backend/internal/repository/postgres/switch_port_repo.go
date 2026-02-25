package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
)

type SwitchPortRepo struct {
	db *pgxpool.Pool
}

func NewSwitchPortRepo(db *pgxpool.Pool) *SwitchPortRepo {
	return &SwitchPortRepo{db: db}
}

func (r *SwitchPortRepo) Create(ctx context.Context, port *domain.SwitchPort) error {
	port.ID = uuid.New()
	if port.PortMode == "" {
		port.PortMode = "access"
	}
	query := `INSERT INTO switch_ports (id, switch_id, server_id, port_index, port_name, speed_mbps, vlan_id, port_mode, native_vlan_id, trunk_vlans, admin_status, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := r.db.Exec(ctx, query,
		port.ID, port.SwitchID, port.ServerID, port.PortIndex, port.PortName,
		port.SpeedMbps, port.VlanID, port.PortMode, port.NativeVlanID, port.TrunkVlans,
		port.AdminStatus, port.Description)
	return err
}

func (r *SwitchPortRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.SwitchPort, error) {
	query := `SELECT id, switch_id, server_id, port_index, port_name, speed_mbps, vlan_id, port_mode, native_vlan_id, trunk_vlans, admin_status, oper_status, description, last_polled
		FROM switch_ports WHERE id = $1`
	row := r.db.QueryRow(ctx, query, id)
	return scanSwitchPort(row)
}

func (r *SwitchPortRepo) ListBySwitchID(ctx context.Context, switchID uuid.UUID) ([]domain.SwitchPort, error) {
	query := `SELECT id, switch_id, server_id, port_index, port_name, speed_mbps, vlan_id, port_mode, native_vlan_id, trunk_vlans, admin_status, oper_status, description, last_polled
		FROM switch_ports WHERE switch_id = $1 ORDER BY port_index`
	rows, err := r.db.Query(ctx, query, switchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectSwitchPorts(rows)
}

func (r *SwitchPortRepo) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]domain.SwitchPort, error) {
	query := `SELECT id, switch_id, server_id, port_index, port_name, speed_mbps, vlan_id, port_mode, native_vlan_id, trunk_vlans, admin_status, oper_status, description, last_polled
		FROM switch_ports WHERE server_id = $1 ORDER BY port_index`
	rows, err := r.db.Query(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectSwitchPorts(rows)
}

func (r *SwitchPortRepo) Update(ctx context.Context, port *domain.SwitchPort) error {
	query := `UPDATE switch_ports SET server_id=$2, port_index=$3, port_name=$4, speed_mbps=$5,
		vlan_id=$6, port_mode=$7, native_vlan_id=$8, trunk_vlans=$9,
		admin_status=$10, oper_status=$11, description=$12, last_polled=$13
		WHERE id=$1`
	_, err := r.db.Exec(ctx, query,
		port.ID, port.ServerID, port.PortIndex, port.PortName, port.SpeedMbps,
		port.VlanID, port.PortMode, port.NativeVlanID, port.TrunkVlans,
		port.AdminStatus, port.OperStatus, port.Description, port.LastPolled)
	return err
}

func (r *SwitchPortRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM switch_ports WHERE id = $1`, id)
	return err
}

func (r *SwitchPortRepo) UpsertBySwitchAndIndex(ctx context.Context, port *domain.SwitchPort) error {
	query := `INSERT INTO switch_ports (id, switch_id, port_index, port_name, speed_mbps, vlan_id, admin_status, oper_status, last_polled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (switch_id, port_index) DO UPDATE SET
			port_name = EXCLUDED.port_name,
			speed_mbps = EXCLUDED.speed_mbps,
			oper_status = EXCLUDED.oper_status,
			last_polled = EXCLUDED.last_polled,
			vlan_id = CASE WHEN switch_ports.vlan_id = 0 THEN EXCLUDED.vlan_id ELSE switch_ports.vlan_id END`
	_, err := r.db.Exec(ctx, query,
		port.ID, port.SwitchID, port.PortIndex, port.PortName,
		port.SpeedMbps, port.VlanID, port.AdminStatus, port.OperStatus, port.LastPolled)
	return err
}

func scanSwitchPort(row pgx.Row) (*domain.SwitchPort, error) {
	var p domain.SwitchPort
	err := row.Scan(&p.ID, &p.SwitchID, &p.ServerID, &p.PortIndex, &p.PortName,
		&p.SpeedMbps, &p.VlanID, &p.PortMode, &p.NativeVlanID, &p.TrunkVlans,
		&p.AdminStatus, &p.OperStatus, &p.Description, &p.LastPolled)
	if err == pgx.ErrNoRows {
		return nil, err
	}
	return &p, err
}

func collectSwitchPorts(rows pgx.Rows) ([]domain.SwitchPort, error) {
	var ports []domain.SwitchPort
	for rows.Next() {
		var p domain.SwitchPort
		if err := rows.Scan(&p.ID, &p.SwitchID, &p.ServerID, &p.PortIndex, &p.PortName,
			&p.SpeedMbps, &p.VlanID, &p.PortMode, &p.NativeVlanID, &p.TrunkVlans,
			&p.AdminStatus, &p.OperStatus, &p.Description, &p.LastPolled); err != nil {
			return nil, err
		}
		ports = append(ports, p)
	}
	return ports, nil
}
