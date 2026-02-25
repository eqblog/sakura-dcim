ALTER TABLE ip_pools DROP COLUMN IF EXISTS trunk_vlans;
ALTER TABLE ip_pools DROP COLUMN IF EXISTS native_vlan_id;
ALTER TABLE ip_pools DROP COLUMN IF EXISTS vlan_mode;
