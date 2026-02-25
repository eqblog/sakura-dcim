ALTER TABLE ip_pools DROP COLUMN IF EXISTS vlan_range_end;
ALTER TABLE ip_pools DROP COLUMN IF EXISTS vlan_range_start;
ALTER TABLE ip_pools DROP COLUMN IF EXISTS vlan_id;
ALTER TABLE ip_pools DROP COLUMN IF EXISTS switch_automation;
ALTER TABLE ip_pools DROP COLUMN IF EXISTS notes;
ALTER TABLE ip_pools DROP COLUMN IF EXISTS rdns_server;
ALTER TABLE ip_pools DROP COLUMN IF EXISTS priority;
