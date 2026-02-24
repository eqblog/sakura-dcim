package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
)

type SwitchRepo struct {
	db *pgxpool.Pool
}

func NewSwitchRepo(db *pgxpool.Pool) *SwitchRepo {
	return &SwitchRepo{db: db}
}

func (r *SwitchRepo) Create(ctx context.Context, sw *domain.Switch) error {
	sw.ID = uuid.New()
	now := time.Now()
	sw.CreatedAt = now
	sw.UpdatedAt = now
	query := `INSERT INTO switches (id, agent_id, name, ip, vendor, model, snmp_community, snmp_version, ssh_user, ssh_pass, ssh_port, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	_, err := r.db.Exec(ctx, query,
		sw.ID, sw.AgentID, sw.Name, sw.IP, sw.Vendor, sw.Model,
		sw.SNMPCommunity, sw.SNMPVersion, sw.SSHUser, sw.SSHPass, sw.SSHPort,
		sw.CreatedAt, sw.UpdatedAt)
	return err
}

func (r *SwitchRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Switch, error) {
	query := `SELECT id, agent_id, name, ip, vendor, model, snmp_community, snmp_version, ssh_user, ssh_pass, ssh_port, created_at, updated_at
		FROM switches WHERE id = $1`
	row := r.db.QueryRow(ctx, query, id)
	var sw domain.Switch
	err := row.Scan(&sw.ID, &sw.AgentID, &sw.Name, &sw.IP, &sw.Vendor, &sw.Model,
		&sw.SNMPCommunity, &sw.SNMPVersion, &sw.SSHUser, &sw.SSHPass, &sw.SSHPort,
		&sw.CreatedAt, &sw.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, err
	}
	return &sw, err
}

func (r *SwitchRepo) List(ctx context.Context) ([]domain.Switch, error) {
	query := `SELECT id, agent_id, name, ip, vendor, model, snmp_community, snmp_version, ssh_user, ssh_pass, ssh_port, created_at, updated_at
		FROM switches ORDER BY name`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var switches []domain.Switch
	for rows.Next() {
		var sw domain.Switch
		if err := rows.Scan(&sw.ID, &sw.AgentID, &sw.Name, &sw.IP, &sw.Vendor, &sw.Model,
			&sw.SNMPCommunity, &sw.SNMPVersion, &sw.SSHUser, &sw.SSHPass, &sw.SSHPort,
			&sw.CreatedAt, &sw.UpdatedAt); err != nil {
			return nil, err
		}
		switches = append(switches, sw)
	}
	return switches, nil
}

func (r *SwitchRepo) Update(ctx context.Context, sw *domain.Switch) error {
	sw.UpdatedAt = time.Now()
	query := `UPDATE switches SET name=$2, ip=$3, vendor=$4, model=$5, snmp_community=$6, snmp_version=$7,
		ssh_user=$8, ssh_pass=$9, ssh_port=$10, agent_id=$11, updated_at=$12 WHERE id=$1`
	_, err := r.db.Exec(ctx, query,
		sw.ID, sw.Name, sw.IP, sw.Vendor, sw.Model,
		sw.SNMPCommunity, sw.SNMPVersion, sw.SSHUser, sw.SSHPass, sw.SSHPort,
		sw.AgentID, sw.UpdatedAt)
	return err
}

func (r *SwitchRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM switches WHERE id = $1`, id)
	return err
}
