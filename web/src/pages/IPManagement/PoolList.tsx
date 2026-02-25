import React, { useState } from 'react';
import { Card, Typography, Table, Button, Space, Modal, Form, Input, Select, Popconfirm, Progress, Tag, Radio, Checkbox, Tooltip, message } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, InfoCircleOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { ipPoolAPI } from '../../api';
import type { IPPool, Tenant } from '../../types';

const { Title } = Typography;

interface Props {
  pools: IPPool[];
  loading: boolean;
  tenants: Tenant[];
  parentId?: string;
  onSelect: (pool: IPPool) => void;
  onRefresh: () => void;
  onDelete: (id: string) => void;
}

// Convert CIDR prefix length to netmask string
function prefixToNetmask(prefix: number): string {
  if (prefix < 0 || prefix > 32) return '';
  const mask = prefix === 0 ? 0 : (~0 << (32 - prefix)) >>> 0;
  return [
    (mask >>> 24) & 0xff,
    (mask >>> 16) & 0xff,
    (mask >>> 8) & 0xff,
    mask & 0xff,
  ].join('.');
}

// Auto-fill gateway and netmask from CIDR
function autoFillFromCIDR(cidr: string, form: any) {
  const match = cidr.match(/^(\d+\.\d+\.\d+\.\d+)\/(\d+)$/);
  if (!match) return;
  const parts = match[1].split('.').map(Number);
  if (parts.some(p => p < 0 || p > 255)) return;
  const prefix = parseInt(match[2]);
  if (prefix < 0 || prefix > 32) return;
  // Gateway = network address + 1 (first usable IP)
  const netInt = ((parts[0] << 24) | (parts[1] << 16) | (parts[2] << 8) | parts[3]) >>> 0;
  const gwInt = netInt + 1;
  const gateway = [
    (gwInt >>> 24) & 0xff,
    (gwInt >>> 16) & 0xff,
    (gwInt >>> 8) & 0xff,
    gwInt & 0xff,
  ].join('.');
  form.setFieldsValue({
    gateway,
    netmask: prefixToNetmask(prefix),
  });
}

const PoolList: React.FC<Props> = ({ pools, loading, tenants, parentId, onSelect, onRefresh, onDelete }) => {
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<IPPool | null>(null);
  const [form] = Form.useForm();
  const poolType = Form.useWatch('pool_type', form);

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({ pool_type: 'ip_pool', reserve_gateway: true });
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
        parent_id: parentId || undefined,
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
      title: 'Type', dataIndex: 'pool_type', key: 'pool_type', width: 100,
      render: (v: string) => v === 'subnet'
        ? <Tag color="blue">Subnet</Tag>
        : <Tag color="green">IP Pool</Tag>,
      filters: [{ text: 'Subnet', value: 'subnet' }, { text: 'IP Pool', value: 'ip_pool' }],
      onFilter: (value, record) => record.pool_type === value,
    },
    {
      title: 'VRF', dataIndex: 'vrf', key: 'vrf', render: (v: string) => v || '-',
      filters: [...new Set(pools.map(p => p.vrf).filter(Boolean))].map(v => ({ text: v, value: v })),
      onFilter: (value, record) => record.vrf === value,
    },
    { title: 'Description', dataIndex: 'description', key: 'description', ellipsis: true },
    {
      title: 'Usage', key: 'usage', width: 180, render: (_, r) => {
        if (r.pool_type === 'subnet') {
          return <span>{r.child_count} subnet{r.child_count !== 1 ? 's' : ''}</span>;
        }
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
          <span onClick={(e) => e.stopPropagation()}>
            <Popconfirm title="Delete this pool and all its contents?" onConfirm={() => onDelete(record.id)}>
              <Button size="small" danger icon={<DeleteOutlined />} />
            </Popconfirm>
          </span>
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
      <Modal title={editing ? 'Edit Pool' : 'Create Pool'} open={modalOpen} onOk={handleSubmit} onCancel={() => setModalOpen(false)} okText={editing ? 'Update' : 'Create'} width={520}>
        <Form form={form} layout="vertical">
          <Form.Item
            name="pool_type"
            label={<Space>Pool Type <Tooltip title="Choose how this subnet will be used"><InfoCircleOutlined /></Tooltip></Space>}
          >
            <Radio.Group>
              <Space direction="vertical">
                <Radio value="ip_pool">Generate IPs for the new subnet</Radio>
                <Radio value="subnet">Subnet will be divided into smaller subnets</Radio>
              </Space>
            </Radio.Group>
          </Form.Item>
          {poolType === 'ip_pool' && !editing && (
            <Form.Item name="reserve_gateway" valuePropName="checked">
              <Checkbox>Reserve Gateway IP</Checkbox>
            </Form.Item>
          )}
          <Form.Item name="network" label="Network (CIDR)" rules={[{ required: true }]}>
            <Input
              placeholder="10.0.0.0/24"
              onBlur={(e) => autoFillFromCIDR(e.target.value, form)}
            />
          </Form.Item>
          <Form.Item name="gateway" label="Gateway" rules={[{ required: true }]}><Input placeholder="10.0.0.1 (auto-filled from CIDR)" /></Form.Item>
          <Form.Item name="netmask" label="Netmask"><Input placeholder="255.255.255.0 (auto-filled from CIDR)" /></Form.Item>
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
