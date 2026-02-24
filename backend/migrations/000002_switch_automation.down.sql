ALTER TABLE switch_ports
    DROP COLUMN IF EXISTS vlan_id,
    DROP COLUMN IF EXISTS admin_status,
    DROP COLUMN IF EXISTS oper_status,
    DROP COLUMN IF EXISTS last_polled;

ALTER TABLE switches
    DROP COLUMN IF EXISTS vendor,
    DROP COLUMN IF EXISTS ssh_user,
    DROP COLUMN IF EXISTS ssh_pass,
    DROP COLUMN IF EXISTS ssh_port,
    DROP COLUMN IF EXISTS model,
    DROP COLUMN IF EXISTS updated_at;
