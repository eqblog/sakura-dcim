import React, { useState } from 'react';
import { Table, Button, Space, Tag, Popconfirm, message } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, ThunderboltOutlined, PoweroffOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { switchAPI } from '../../api';
import type { Switch, SwitchPort, SwitchBandwidthMap } from '../../types';
import PortEditModal from './PortEditModal';

const formatBytes = (b: number): string => {
  if (!b) return '0 B';
  if (b >= 1e12) return `${(b / 1e12).toFixed(1)} TB`;
  if (b >= 1e9) return `${(b / 1e9).toFixed(1)} GB`;
  if (b >= 1e6) return `${(b / 1e6).toFixed(1)} MB`;
  if (b >= 1e3) return `${(b / 1e3).toFixed(1)} KB`;
  return `${b} B`;
};

const formatSpeed = (v: number): string => (v >= 1000 ? `${v / 1000}G` : `${v}M`);

const portModeTag = (mode: string) => {
  switch (mode) {
    case 'trunk': return <Tag color="purple">Trunk</Tag>;
    case 'trunk_native': return <Tag color="volcano">Trunk+Native</Tag>;
    default: return <Tag color="cyan">Access</Tag>;
  }
};

interface Props {
  sw: Switch;
  ports: SwitchPort[];
  loading: boolean;
  bandwidth: SwitchBandwidthMap;
  onRefresh: () => void;
}

const InterfacesTab: React.FC<Props> = ({ sw, ports, loading, bandwidth, onRefresh }) => {
  const [modalOpen, setModalOpen] = useState(false);
  const [editingPort, setEditingPort] = useState<SwitchPort | null>(null);
  const [adminLoading, setAdminLoading] = useState<string | null>(null);

  const handleToggleAdmin = async (port: SwitchPort) => {
    const newStatus = port.admin_status === 'up' ? 'down' : 'up';
    setAdminLoading(port.id);
    try {
      const { data: resp } = await switchAPI.togglePortAdmin(sw.id, port.id, newStatus);
      if (resp.success) { message.success(`Port ${newStatus === 'up' ? 'enabled' : 'disabled'}`); onRefresh(); }
      else message.error(resp.error);
    } catch { message.error('Failed to toggle port'); }
    setAdminLoading(null);
  };

  const handleProvision = async (portId: string) => {
    try {
      const { data: resp } = await switchAPI.provisionPort(sw.id, portId);
      if (resp.success) message.success('Port provisioned');
      else message.error(resp.error);
    } catch { message.error('Provision failed'); }
  };

  const handleDelete = async (portId: string) => {
    const { data: resp } = await switchAPI.deletePort(sw.id, portId);
    if (resp.success) { message.success('Deleted'); onRefresh(); }
    else message.error(resp.error);
  };

  const columns: ColumnsType<SwitchPort> = [
    {
      title: 'Port', key: 'port', width: 180, sorter: (a, b) => a.port_index - b.port_index,
      render: (_, r) => (
        <div>
          <div style={{ fontWeight: 500 }}>{r.port_name}</div>
          {r.description && <div style={{ fontSize: 12, color: '#888' }}>{r.description}</div>}
        </div>
      ),
    },
    {
      title: 'Assignment', key: 'assignment', width: 140,
      render: (_, r) => r.server_id ? <Tag color="blue">{r.server_id.slice(0, 8)}...</Tag> : <span style={{ color: '#999' }}>Not assigned</span>,
    },
    { title: 'Speed', dataIndex: 'speed_mbps', key: 'speed', width: 80, render: (v: number) => formatSpeed(v) },
    { title: 'Type', key: 'type', width: 110, render: (_, r) => portModeTag(r.port_mode) },
    {
      title: 'Status', key: 'status', width: 130,
      render: (_, r) => (
        <Space>
          <Tag color={r.oper_status === 'up' ? 'green' : 'red'}>{r.oper_status === 'up' ? 'Up' : 'Down'}</Tag>
          <Tag color={r.admin_status === 'up' ? 'green' : 'orange'}>{r.admin_status === 'up' ? 'Enabled' : 'Disabled'}</Tag>
        </Space>
      ),
    },
    {
      title: 'Traffic Today', key: 'traffic_today', width: 140,
      render: (_, r) => {
        const bw = bandwidth[r.id];
        if (!bw) return '-';
        return <span style={{ fontSize: 12 }}>{formatBytes(bw.traffic_today_in)} / {formatBytes(bw.traffic_today_out)}</span>;
      },
    },
    {
      title: 'Traffic Month', key: 'traffic_month', width: 140,
      render: (_, r) => {
        const bw = bandwidth[r.id];
        if (!bw) return '-';
        return <span style={{ fontSize: 12 }}>{formatBytes(bw.traffic_month_in)} / {formatBytes(bw.traffic_month_out)}</span>;
      },
    },
    {
      title: 'Actions', key: 'actions', width: 160,
      render: (_, record) => (
        <Space>
          <Button size="small" icon={<PoweroffOutlined />} loading={adminLoading === record.id}
            danger={record.admin_status === 'up'} type={record.admin_status === 'up' ? 'default' : 'primary'}
            title={record.admin_status === 'up' ? 'Shutdown' : 'Enable'} onClick={() => handleToggleAdmin(record)} />
          <Popconfirm title="Provision this port?" onConfirm={() => handleProvision(record.id)}>
            <Button size="small" icon={<ThunderboltOutlined />} title="Provision" />
          </Popconfirm>
          <Button size="small" icon={<EditOutlined />} onClick={() => { setEditingPort(record); setModalOpen(true); }} />
          <Popconfirm title="Delete this port?" onConfirm={() => handleDelete(record.id)}>
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 12 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => { setEditingPort(null); setModalOpen(true); }}>Add Port</Button>
      </div>
      <Table columns={columns} dataSource={ports} rowKey="id" loading={loading} size="small" pagination={{ pageSize: 50 }} />
      <PortEditModal
        open={modalOpen}
        port={editingPort}
        switchId={sw.id}
        onClose={() => setModalOpen(false)}
        onSuccess={() => { setModalOpen(false); onRefresh(); }}
      />
    </div>
  );
};

export default InterfacesTab;
