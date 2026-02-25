import React, { useEffect, useState, useCallback } from 'react';
import { message } from 'antd';
import { ipPoolAPI, tenantAPI } from '../../api';
import type { IPPool, IPAddress, Tenant } from '../../types';
import PoolList from './PoolList';
import SubnetDetail from './SubnetDetail';

const IPManagementPage: React.FC = () => {
  const [pools, setPools] = useState<IPPool[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedPool, setSelectedPool] = useState<IPPool | null>(null);
  const [addresses, setAddresses] = useState<IPAddress[]>([]);
  const [addrLoading, setAddrLoading] = useState(false);
  const [tenants, setTenants] = useState<Tenant[]>([]);

  useEffect(() => {
    tenantAPI.list({ page: 1, page_size: 200 }).then(({ data: resp }) => {
      if (resp.success) setTenants(resp.data?.items || []);
    });
  }, []);

  const fetchPools = useCallback(async () => {
    setLoading(true);
    try {
      const { data: resp } = await ipPoolAPI.list();
      if (resp.success) setPools(resp.data || []);
    } catch { /* */ }
    setLoading(false);
  }, []);

  useEffect(() => { fetchPools(); }, [fetchPools]);

  const fetchAddresses = useCallback(async (pool: IPPool) => {
    setAddrLoading(true);
    try {
      const { data: resp } = await ipPoolAPI.listAddresses(pool.id);
      if (resp.success) setAddresses(resp.data || []);
    } catch { /* */ }
    setAddrLoading(false);
  }, []);

  const handleSelectPool = useCallback((pool: IPPool) => {
    setSelectedPool(pool);
    fetchAddresses(pool);
  }, [fetchAddresses]);

  const handleBack = useCallback(() => {
    setSelectedPool(null);
    setAddresses([]);
  }, []);

  const handlePoolSaved = useCallback(async () => {
    await fetchPools();
    if (selectedPool) {
      const { data: resp } = await ipPoolAPI.get(selectedPool.id);
      if (resp.success && resp.data) setSelectedPool(resp.data);
    }
  }, [fetchPools, selectedPool]);

  const handleDeletePool = useCallback(async (id: string) => {
    const { data: resp } = await ipPoolAPI.delete(id);
    if (resp.success) {
      message.success('Pool deleted');
      fetchPools();
      if (selectedPool?.id === id) handleBack();
    } else message.error(resp.error);
  }, [fetchPools, selectedPool, handleBack]);

  if (selectedPool) {
    return (
      <SubnetDetail
        pool={selectedPool}
        addresses={addresses}
        addrLoading={addrLoading}
        tenants={tenants}
        onBack={handleBack}
        onPoolSaved={handlePoolSaved}
        onRefreshAddresses={() => fetchAddresses(selectedPool)}
      />
    );
  }

  return (
    <PoolList
      pools={pools}
      loading={loading}
      tenants={tenants}
      onSelect={handleSelectPool}
      onRefresh={fetchPools}
      onDelete={handleDeletePool}
    />
  );
};

export default IPManagementPage;
