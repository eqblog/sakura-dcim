ALTER TABLE switch_ports ADD COLUMN port_mode VARCHAR(20) NOT NULL DEFAULT 'access';
ALTER TABLE switch_ports ADD COLUMN native_vlan_id INT NOT NULL DEFAULT 0;
ALTER TABLE switch_ports ADD COLUMN trunk_vlans TEXT NOT NULL DEFAULT '';
