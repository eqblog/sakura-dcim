import React, { useState } from 'react';
import { Card, Typography, Table, Button, Space, Modal, Form, Input, Select, Popconfirm, Progress, message } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { ipPoolAPI } from '../../api';
import type { IPPool, Tenant } from '../../types';

const { Title } = Typography;

interface Props {
  pools: IPPool[];
  loading: boolean;
  tenants: Tenant[];
  onSelect: (pool: IPPool) => void;
  onRefresh: () => void;
  onDelete: (id: string) => void;
}

const PoolList: React.FC<Props> = ({ pools, loading, tenants, onSelect, onRefresh, onDelete }) => {
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<IPPool | null>(null);
  const [form] = Form.useForm();

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    setModalOpen(true);
  };

  const openEdit = (record: IPPool) => {
    setEditing(record);
    form.setFieldsValue({ ...record, nameservers: record.nameservers?.join(', ') || '' });
    setModalOpen(true);
  };

  const handleSubmit = async () => {
    try {
      const raw = await form.validateFields();
      const values = {
        ...raw,
        nameservers: raw.nameservers ? String(raw.nameservers).split(',').map((s: string) => s.trim()).filter(Boolean) : [],
      };
      if (editing) {
        const { data: resp } = await ipPoolAPI.update(editing.id, values);
        if (resp.success) message.success('Pool updated');
        else { message.error(resp.error); return; }
      } else {
        const { data: resp } = await ipPoolAPI.create(values);
        if (resp.success) message.success('Pool created');
        else { message.error(resp.error); return; }
      }
      setModalOpen(false);
      onRefresh();
    } catch { /* validation */ }
  };

  const columns: ColumnsType<IPPool> = [
    { title: 'Network', dataIndex: 'network', key: 'network', sorter: (a, b) => a.network.localeCompare(b.network) },
    { title: 'Gateway', dataIndex: 'gateway', key: 'gateway' },
    { title: 'Netmask', dataIndex: 'netmask', key: 'netmask' },
    {
      title: 'VRF', dataIndex: 'vrf', key: 'vrf', render: (v: string) => v || '-',
      filters: [...new Set(pools.map(p => p.vrf).filter(Boolean))].map(v => ({ text: v, value: v })),
      onFilter: (value, record) => record.vrf === value,
    },
    { title: 'Description', dataIndex: 'description', key: 'description', ellipsis: true },
    {
      title: 'Usage', key: 'usage', width: 180, render: (_, r) => {
        const pct = r.total_ips > 0 ? Math.round((r.used_ips / r.total_ips) * 100) : 0;
        return (
          <Space direction="vertical" size={0} style={{ width: '100%' }}>
            <Progress percent={pct} size="small" strokeColor={pct > 80 ? '#ff4d4f' : undefined} />
            <span style={{ fontSize: 12 }}>{r.used_ips} / {r.total_ips} used</span>
          </Space>
        );
      },
    },
    {
      title: 'Actions', key: 'actions', width: 220,
      render: (_, record) => (
        <Space>
          <Button size="small" type="link" onClick={() => onSelect(record)}>Manage</Button>
          <Button size="small" icon={<EditOutlined />} onClick={(e) => { e.stopPropagation(); openEdit(record); }} />
          <Popconfirm title="Delete this pool and all its addresses?" onConfirm={() => onDelete(record.id)}>
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4}>IP Pools</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>Add Pool</Button>
      </div>
      <Card>
        <Table columns={columns} dataSource={pools} rowKey="id" loading={loading} size="small"
          onRow={(record) => ({ onClick: () => onSelect(record), style: { cursor: 'pointer' } })} />
      </Card>
      <Modal title={editing ? 'Edit Pool' : 'Create Pool'} open={modalOpen} onOk={handleSubmit} onCancel={() => setModalOpen(false)} okText={editing ? 'Update' : 'Create'}>
        <Form form={form} layout="vertical">
          <Form.Item name="network" label="Network (CIDR)" rules={[{ required: true }]}><Input placeholder="10.0.0.0/24" /></Form.Item>
          <Form.Item name="gateway" label="Gateway" rules={[{ required: true }]}><Input placeholder="10.0.0.1" /></Form.Item>
          <Form.Item name="netmask" label="Netmask"><Input placeholder="255.255.255.0" /></Form.Item>
          <Form.Item name="vrf" label="VRF"><Input placeholder="default (empty = no VRF)" /></Form.Item>
          <Form.Item name="nameservers" label="Nameservers"><Input placeholder="8.8.8.8, 8.8.4.4 (comma-separated)" /></Form.Item>
          <Form.Item name="tenant_id" label="Tenant">
            <Select allowClear placeholder="Select tenant (optional)" options={tenants.map((t) => ({ label: t.name, value: t.id }))} />
          </Form.Item>
          <Form.Item name="description" label="Description"><Input placeholder="Production subnet" /></Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default PoolList;
