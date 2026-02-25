ALTER TABLE switch_ports DROP COLUMN IF EXISTS trunk_vlans;
ALTER TABLE switch_ports DROP COLUMN IF EXISTS native_vlan_id;
ALTER TABLE switch_ports DROP COLUMN IF EXISTS port_mode;
