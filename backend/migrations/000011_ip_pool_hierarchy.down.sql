DROP INDEX IF EXISTS idx_ip_pools_parent;
ALTER TABLE ip_pools DROP COLUMN IF EXISTS pool_type;
ALTER TABLE ip_pools DROP COLUMN IF EXISTS parent_id;
