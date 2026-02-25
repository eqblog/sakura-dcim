import React, { useState } from 'react';
import { Card, Table, Button, Space, Modal, Form, Input, Select, Radio, Checkbox, Tag, Tooltip, Popconfirm, Progress, message } from 'antd';
import { PlusOutlined, DeleteOutlined, InfoCircleOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { ipPoolAPI } from '../../api';
import type { IPPool, Tenant } from '../../types';

interface Props {
  parentPool: IPPool;
  childPools: IPPool[];
  loading: boolean;
  tenants: Tenant[];
  onSelect: (pool: IPPool) => void;
  onRefresh: () => void;
  onDelete: (id: string) => void;
}

function prefixToNetmask(prefix: number): string {
  if (prefix < 0 || prefix > 32) return '';
  const mask = prefix === 0 ? 0 : (~0 << (32 - prefix)) >>> 0;
  return [(mask >>> 24) & 0xff, (mask >>> 16) & 0xff, (mask >>> 8) & 0xff, mask & 0xff].join('.');
}

function autoFillFromCIDR(cidr: string, form: any) {
  const match = cidr.match(/^(\d+\.\d+\.\d+\.\d+)\/(\d+)$/);
  if (!match) return;
  const parts = match[1].split('.').map(Number);
  if (parts.some(p => p < 0 || p > 255)) return;
  const prefix = parseInt(match[2]);
  if (prefix < 0 || prefix > 32) return;
  const netInt = ((parts[0] << 24) | (parts[1] << 16) | (parts[2] << 8) | parts[3]) >>> 0;
  const gwInt = netInt + 1;
  form.setFieldsValue({
    gateway: [(gwInt >>> 24) & 0xff, (gwInt >>> 16) & 0xff, (gwInt >>> 8) & 0xff, gwInt & 0xff].join('.'),
    netmask: prefixToNetmask(prefix),
  });
}

const ChildSubnetsTab: React.FC<Props> = ({ parentPool, childPools, loading, tenants, onSelect, onRefresh, onDelete }) => {
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm();
  const poolType = Form.useWatch('pool_type', form);

  const openCreate = () => {
    form.resetFields();
    form.setFieldsValue({ pool_type: 'ip_pool', reserve_gateway: true });
    setModalOpen(true);
  };

  const handleSubmit = async () => {
    try {
      const raw = await form.validateFields();
      const values = {
        ...raw,
        nameservers: raw.nameservers ? String(raw.nameservers).split(',').map((s: string) => s.trim()).filter(Boolean) : [],
        parent_id: parentPool.id,
      };
      const { data: resp } = await ipPoolAPI.create(values);
      if (resp.success) {
        message.success('Subnet created');
        setModalOpen(false);
        onRefresh();
      } else message.error(resp.error);
    } catch { /* validation */ }
  };

  const columns: ColumnsType<IPPool> = [
    { title: 'Network', dataIndex: 'network', key: 'network', sorter: (a, b) => a.network.localeCompare(b.network) },
    { title: 'Gateway', dataIndex: 'gateway', key: 'gateway' },
    {
      title: 'Type', dataIndex: 'pool_type', key: 'pool_type', width: 100,
      render: (v: string) => v === 'subnet'
        ? <Tag color="blue">Subnet</Tag>
        : <Tag color="green">IP Pool</Tag>,
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
      title: 'Actions', key: 'actions', width: 160,
      render: (_, record) => (
        <Space>
          <Button size="small" type="link" onClick={() => onSelect(record)}>Manage</Button>
          <span onClick={(e) => e.stopPropagation()}>
            <Popconfirm title="Delete this subnet and all its contents?" onConfirm={() => onDelete(record.id)}>
              <Button size="small" danger icon={<DeleteOutlined />} />
            </Popconfirm>
          </span>
        </Space>
      ),
    },
  ];

  return (
    <Card
      title={`${childPools.length} subnets in ${parentPool.network}`}
      extra={<Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>Add Subnet</Button>}
    >
      <Table columns={columns} dataSource={childPools} rowKey="id" loading={loading} size="small"
        onRow={(record) => ({ onClick: () => onSelect(record), style: { cursor: 'pointer' } })} />
      <Modal title={`Add Subnet to ${parentPool.network}`} open={modalOpen} onOk={handleSubmit} onCancel={() => setModalOpen(false)} okText="Create" width={520}>
        <Form form={form} layout="vertical">
          <Form.Item
            name="pool_type"
            label={<Space>Select Subnet Action <Tooltip title="Choose how this subnet will be used"><InfoCircleOutlined /></Tooltip></Space>}
          >
            <Radio.Group>
              <Space direction="vertical">
                <Radio value="ip_pool">Generate IPs for the new subnet</Radio>
                <Radio value="subnet">Subnet will be divided into smaller subnets</Radio>
              </Space>
            </Radio.Group>
          </Form.Item>
          {poolType === 'ip_pool' && (
            <Form.Item name="reserve_gateway" valuePropName="checked">
              <Checkbox>Reserve Gateway IP</Checkbox>
            </Form.Item>
          )}
          <Form.Item name="network" label="Network (CIDR)" rules={[{ required: true }]}>
            <Input placeholder={`e.g. subnet of ${parentPool.network}`} onBlur={(e) => autoFillFromCIDR(e.target.value, form)} />
          </Form.Item>
          <Form.Item name="gateway" label="Gateway" rules={[{ required: true }]}>
            <Input placeholder="10.0.0.1 (auto-filled from CIDR)" />
          </Form.Item>
          <Form.Item name="netmask" label="Netmask"><Input placeholder="255.255.255.0 (auto-filled from CIDR)" /></Form.Item>
          <Form.Item name="nameservers" label="Nameservers"><Input placeholder="8.8.8.8, 8.8.4.4" /></Form.Item>
          <Form.Item name="tenant_id" label="Tenant">
            <Select allowClear placeholder="Select tenant" options={tenants.map((t) => ({ label: t.name, value: t.id }))} />
          </Form.Item>
          <Form.Item name="description" label="Description"><Input placeholder="Description" /></Form.Item>
        </Form>
      </Modal>
    </Card>
  );
};

export default ChildSubnetsTab;
