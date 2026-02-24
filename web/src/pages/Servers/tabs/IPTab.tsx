import React, { useEffect, useState } from 'react';
import { Table, Button, Modal, Select, Space, Tag, Typography, message } from 'antd';
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons';
import { ipPoolAPI } from '../../../api';
import type { IPPool, IPAddress } from '../../../types';

const { Text } = Typography;

interface IPTabProps {
  serverId: string;
}

const IPTab: React.FC<IPTabProps> = ({ serverId }) => {
  const [addresses, setAddresses] = useState<IPAddress[]>([]);
  const [pools, setPools] = useState<IPPool[]>([]);
  const [loading, setLoading] = useState(false);
  const [assignModalOpen, setAssignModalOpen] = useState(false);
  const [selectedPool, setSelectedPool] = useState<string>('');
  const [assigning, setAssigning] = useState(false);

  useEffect(() => {
    fetchAssignedIPs();
    fetchPools();
  }, [serverId]);

  const fetchAssignedIPs = async () => {
    setLoading(true);
    try {
      const { data: resp } = await ipPoolAPI.list();
      if (resp.success && resp.data) {
        const allAddrs: IPAddress[] = [];
        for (const pool of resp.data) {
          const { data: addrResp } = await ipPoolAPI.listAddresses(pool.id);
          if (addrResp.success && addrResp.data) {
            const serverAddrs = addrResp.data.filter(
              (a: IPAddress) => a.server_id === serverId
            );
            allAddrs.push(...serverAddrs);
          }
        }
        setAddresses(allAddrs);
      }
    } catch { /* ignore */ }
    setLoading(false);
  };

  const fetchPools = async () => {
    try {
      const { data: resp } = await ipPoolAPI.list();
      if (resp.success && resp.data) {
        setPools(resp.data);
      }
    } catch { /* ignore */ }
  };

  const handleAssign = async () => {
    if (!selectedPool) return;
    setAssigning(true);
    try {
      const { data: resp } = await ipPoolAPI.assignNext(selectedPool, serverId);
      if (resp.success) {
        message.success(`IP ${resp.data?.address} assigned`);
        setAssignModalOpen(false);
        setSelectedPool('');
        fetchAssignedIPs();
      } else {
        message.error(resp.error || 'Failed to assign IP');
      }
    } catch {
      message.error('Failed to assign IP');
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
        const pool = pools.find(p => p.id === poolId);
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
        onCancel={() => setAssignModalOpen(false)}
        onOk={handleAssign}
        confirmLoading={assigning}
        okText="Assign Next Available"
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          <Text>Select an IP pool to assign the next available address from:</Text>
          <Select
            placeholder="Select IP Pool"
            value={selectedPool || undefined}
            onChange={setSelectedPool}
            style={{ width: '100%' }}
            options={pools.map(p => ({
              label: `${p.network} (${p.total_ips - p.used_ips} available)`,
              value: p.id,
              disabled: p.total_ips - p.used_ips <= 0,
            }))}
          />
        </Space>
      </Modal>
    </div>
  );
};

export default IPTab;
