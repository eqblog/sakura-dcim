-- Fix legacy pools that have empty pool_type (should be 'ip_pool')
UPDATE ip_pools SET pool_type = 'ip_pool' WHERE pool_type = '';
