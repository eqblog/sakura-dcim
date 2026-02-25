import React, { useEffect, useState, useCallback } from 'react';
import { message } from 'antd';
import { ipPoolAPI, tenantAPI } from '../../api';
import type { IPPool, IPAddress, Tenant } from '../../types';
import PoolList from './PoolList';
import SubnetDetail from './SubnetDetail';

const IPManagementPage: React.FC = () => {
  const [pools, setPools] = useState<IPPool[]>([]);
  const [loading, setLoading] = useState(false);
  const [poolStack, setPoolStack] = useState<IPPool[]>([]);
  const [childPools, setChildPools] = useState<IPPool[]>([]);
  const [addresses, setAddresses] = useState<IPAddress[]>([]);
  const [addrLoading, setAddrLoading] = useState(false);
  const [tenants, setTenants] = useState<Tenant[]>([]);

  const currentPool = poolStack.length > 0 ? poolStack[poolStack.length - 1] : null;

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

  const fetchPoolData = useCallback(async (pool: IPPool) => {
    setAddrLoading(true);
    try {
      if (pool.pool_type === 'subnet') {
        const { data: resp } = await ipPoolAPI.listChildren(pool.id);
        if (resp.success) setChildPools(resp.data || []);
        setAddresses([]);
      } else {
        const { data: resp } = await ipPoolAPI.listAddresses(pool.id);
        if (resp.success) setAddresses(resp.data || []);
        setChildPools([]);
      }
    } catch { /* */ }
    setAddrLoading(false);
  }, []);

  const handleSelectPool = useCallback((pool: IPPool) => {
    setPoolStack([pool]);
    fetchPoolData(pool);
  }, [fetchPoolData]);

  const handleDrillDown = useCallback((pool: IPPool) => {
    setPoolStack(prev => [...prev, pool]);
    fetchPoolData(pool);
  }, [fetchPoolData]);

  const handleBreadcrumbNav = useCallback((index: number) => {
    if (index < 0) {
      setPoolStack([]);
      setChildPools([]);
      setAddresses([]);
      return;
    }
    const newStack = poolStack.slice(0, index + 1);
    setPoolStack(newStack);
    fetchPoolData(newStack[newStack.length - 1]);
  }, [poolStack, fetchPoolData]);

  const handleBack = useCallback(() => {
    if (poolStack.length <= 1) {
      setPoolStack([]);
      setChildPools([]);
      setAddresses([]);
    } else {
      const newStack = poolStack.slice(0, -1);
      setPoolStack(newStack);
      fetchPoolData(newStack[newStack.length - 1]);
    }
  }, [poolStack, fetchPoolData]);

  const handlePoolSaved = useCallback(async () => {
    await fetchPools();
    if (currentPool) {
      const { data: resp } = await ipPoolAPI.get(currentPool.id);
      if (resp.success && resp.data) {
        setPoolStack(prev => [...prev.slice(0, -1), resp.data!]);
        fetchPoolData(resp.data);
      }
    }
  }, [fetchPools, currentPool, fetchPoolData]);

  const handleDeletePool = useCallback(async (id: string) => {
    const { data: resp } = await ipPoolAPI.delete(id);
    if (resp.success) {
      message.success('Pool deleted');
      fetchPools();
      if (currentPool?.id === id) handleBack();
      else if (currentPool) fetchPoolData(currentPool);
    } else message.error(resp.error);
  }, [fetchPools, currentPool, handleBack, fetchPoolData]);

  if (currentPool) {
    return (
      <SubnetDetail
        pool={currentPool}
        poolStack={poolStack}
        childPools={childPools}
        addresses={addresses}
        addrLoading={addrLoading}
        tenants={tenants}
        onBack={handleBack}
        onBreadcrumbNav={handleBreadcrumbNav}
        onDrillDown={handleDrillDown}
        onPoolSaved={handlePoolSaved}
        onRefreshAddresses={() => fetchPoolData(currentPool)}
        onDeleteChild={handleDeletePool}
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
