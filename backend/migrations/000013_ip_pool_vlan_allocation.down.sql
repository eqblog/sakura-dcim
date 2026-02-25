DROP INDEX IF EXISTS idx_ip_pools_unique_network;
ALTER TABLE ip_pools DROP COLUMN IF EXISTS vlan_allocation;
