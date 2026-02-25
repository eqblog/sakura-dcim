ALTER TABLE ip_pools ADD COLUMN parent_id UUID REFERENCES ip_pools(id) ON DELETE CASCADE;
ALTER TABLE ip_pools ADD COLUMN pool_type VARCHAR(20) NOT NULL DEFAULT 'ip_pool';

CREATE INDEX idx_ip_pools_parent ON ip_pools(parent_id);
