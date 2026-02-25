import React, { useEffect, useState, useMemo } from 'react';
import { Table, Button, Modal, Select, Space, Tag, Typography, Divider, message } from 'antd';
import { PlusOutlined, DeleteOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { ipPoolAPI } from '../../../api';
import type { IPPool, IPAddress } from '../../../types';

const { Text } = Typography;

interface IPTabProps {
  serverId: string;
}

const IPTab: React.FC<IPTabProps> = ({ serverId }) => {
  const [addresses, setAddresses] = useState<IPAddress[]>([]);
  const [pools, setPools] = useState<IPPool[]>([]);
  const [allPools, setAllPools] = useState<IPPool[]>([]);
  const [loading, setLoading] = useState(false);
  const [assignModalOpen, setAssignModalOpen] = useState(false);
  const [selectedPool, setSelectedPool] = useState<string>('');
  const [selectedVRF, setSelectedVRF] = useState<string>('');
  const [assigning, setAssigning] = useState(false);

  useEffect(() => {
    fetchAssignedIPs();
    fetchPools();
  }, [serverId]);

  const fetchAssignedIPs = async () => {
    setLoading(true);
    try {
      const { data: resp } = await ipPoolAPI.listAddressesByServer(serverId);
      if (resp.success && resp.data) {
        setAddresses(resp.data);
      }
    } catch { /* ignore */ }
    setLoading(false);
  };

  const fetchPools = async () => {
    try {
      // Fetch all assignable pools (includes nested, only ip_pool type with available IPs)
      const { data: resp } = await ipPoolAPI.listAssignable();
      if (resp.success && resp.data) {
        setPools(resp.data);
      }
      // Also fetch all pools for display lookup
      const { data: allResp } = await ipPoolAPI.list();
      if (allResp.success && allResp.data) {
        setAllPools(allResp.data);
      }
    } catch { /* ignore */ }
  };

  const vrfOptions = useMemo(() => {
    const vrfs = [...new Set(pools.map(p => p.vrf).filter(Boolean))];
    return vrfs.map(v => ({ label: v, value: v }));
  }, [pools]);

  const filteredPools = useMemo(() => {
    if (!selectedVRF) return pools;
    return pools.filter(p => p.vrf === selectedVRF);
  }, [pools, selectedVRF]);

  const handleAssign = async () => {
    if (!selectedPool) return;
    setAssigning(true);
    try {
      const { data: resp } = await ipPoolAPI.assignNext(selectedPool, serverId);
      if (resp.success) {
        message.success(`IP ${resp.data?.address} assigned`);
        setAssignModalOpen(false);
        setSelectedPool('');
        setSelectedVRF('');
        fetchAssignedIPs();
        fetchPools();
      } else {
        message.error(resp.error || 'Failed to assign IP');
      }
    } catch {
      message.error('Failed to assign IP');
    }
    setAssigning(false);
  };

  const handleAutoAssign = async () => {
    setAssigning(true);
    try {
      const { data: resp } = await ipPoolAPI.autoAssign(serverId, undefined, selectedVRF || undefined);
      if (resp.success) {
        message.success(`IP ${resp.data?.address} auto-assigned`);
        setAssignModalOpen(false);
        setSelectedPool('');
        setSelectedVRF('');
        fetchAssignedIPs();
        fetchPools();
      } else {
        message.error(resp.error || 'No available IP pool found');
      }
    } catch {
      message.error('Failed to auto-assign IP');
    }
    setAssigning(false);
  };

  const handleUnassign = async (addr: IPAddress) => {
    try {
      const { data: resp } = await ipPoolAPI.updateAddress(addr.pool_id, addr.id, {
        server_id: null,
        status: 'available',
      });
      if (resp.success) {
        message.success('IP unassigned');
        fetchAssignedIPs();
        fetchPools();
      }
    } catch {
      message.error('Failed to unassign IP');
    }
  };

  const columns = [
    {
      title: 'Address',
      dataIndex: 'address',
      key: 'address',
      render: (ip: string) => <Text code>{ip}</Text>,
    },
    {
      title: 'Pool',
      dataIndex: 'pool_id',
      key: 'pool_id',
      render: (poolId: string) => {
        const pool = allPools.find(p => p.id === poolId) || pools.find(p => p.id === poolId);
        return pool ? <Tag>{pool.network}</Tag> : poolId.slice(0, 8);
      },
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (s: string) => (
        <Tag color={s === 'assigned' ? 'blue' : s === 'reserved' ? 'orange' : 'default'}>{s}</Tag>
      ),
    },
    {
      title: 'Note',
      dataIndex: 'note',
      key: 'note',
    },
    {
      title: '',
      key: 'actions',
      width: 80,
      render: (_: any, record: IPAddress) => (
        <Button
          type="text"
          danger
          size="small"
          icon={<DeleteOutlined />}
          onClick={() => handleUnassign(record)}
        >
          Unassign
        </Button>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Text strong>Assigned IP Addresses</Text>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setAssignModalOpen(true)}>
          Assign IP
        </Button>
      </div>

      <Table
        dataSource={addresses}
        columns={columns}
        rowKey="id"
        loading={loading}
        pagination={false}
        size="small"
      />

      <Modal
        title="Assign IP Address"
        open={assignModalOpen}
        onCancel={() => { setAssignModalOpen(false); setSelectedPool(''); setSelectedVRF(''); }}
        footer={[
          <Button key="cancel" onClick={() => { setAssignModalOpen(false); setSelectedPool(''); setSelectedVRF(''); }}>
            Cancel
          </Button>,
          <Button
            key="auto"
            icon={<ThunderboltOutlined />}
            loading={assigning}
            onClick={handleAutoAssign}
          >
            Auto-assign
          </Button>,
          <Button
            key="assign"
            type="primary"
            disabled={!selectedPool}
            loading={assigning}
            onClick={handleAssign}
          >
            Assign from Selected Pool
          </Button>,
        ]}
      >
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          <div>
            <Text type="secondary">Filter by VRF (optional):</Text>
            <Select
              placeholder="All VRFs"
              value={selectedVRF || undefined}
              onChange={(v) => { setSelectedVRF(v || ''); setSelectedPool(''); }}
              allowClear
              style={{ width: '100%', marginTop: 4 }}
              options={vrfOptions}
            />
          </div>

          <Divider style={{ margin: '8px 0' }} />

          <div>
            <Text>Select a specific IP pool:</Text>
            <Select
              placeholder="Select IP Pool"
              value={selectedPool || undefined}
              onChange={setSelectedPool}
              style={{ width: '100%', marginTop: 4 }}
              options={filteredPools.map(p => ({
                label: `${p.network}${p.vrf ? ` [${p.vrf}]` : ''} (${p.total_ips - p.used_ips} available)`,
                value: p.id,
                disabled: p.total_ips - p.used_ips <= 0,
              }))}
            />
          </div>

          <Text type="secondary" style={{ fontSize: 12 }}>
            Or click "Auto-assign" to automatically pick from the best available pool{selectedVRF ? ` in VRF "${selectedVRF}"` : ''}.
          </Text>
        </Space>
      </Modal>
    </div>
  );
};

export default IPTab;
