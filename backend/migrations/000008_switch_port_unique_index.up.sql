-- Add unique constraint on (switch_id, port_index) for SNMP port upsert
CREATE UNIQUE INDEX IF NOT EXISTS idx_switch_ports_switch_port_index ON switch_ports(switch_id, port_index);
