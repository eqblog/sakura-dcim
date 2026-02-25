ALTER TABLE ip_pools ADD COLUMN vlan_allocation VARCHAR(20) NOT NULL DEFAULT 'fixed';

-- Prevent duplicate CIDRs at the same parent level
CREATE UNIQUE INDEX idx_ip_pools_unique_network
  ON ip_pools (network, COALESCE(parent_id, '00000000-0000-0000-0000-000000000000'));
