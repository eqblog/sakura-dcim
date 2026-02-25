import React, { useEffect, useState } from 'react';
import { Card, Typography, Table, Button, Space, Modal, Form, Input, Select, Tag, message, Popconfirm, Progress } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { ipPoolAPI, tenantAPI } from '../../api';
import type { IPPool, IPAddress, Tenant } from '../../types';

const { Title } = Typography;

const statusColors: Record<string, string> = {
  available: 'green',
  assigned: 'blue',
  reserved: 'orange',
};

const IPManagementPage: React.FC = () => {
  const [pools, setPools] = useState<IPPool[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<IPPool | null>(null);
  const [form] = Form.useForm();

  // Address state
  const [selectedPool, setSelectedPool] = useState<IPPool | null>(null);
  const [addresses, setAddresses] = useState<IPAddress[]>([]);
  const [addrLoading, setAddrLoading] = useState(false);
  const [addrModalOpen, setAddrModalOpen] = useState(false);
  const [editingAddr, setEditingAddr] = useState<IPAddress | null>(null);
  const [addrForm] = Form.useForm();
  const [tenants, setTenants] = useState<Tenant[]>([]);

  useEffect(() => {
    tenantAPI.list({ page: 1, page_size: 200 }).then(({ data: resp }) => {
      if (resp.success) setTenants(resp.data?.items || []);
    });
  }, []);

  const fetchPools = async () => {
    setLoading(true);
    try {
      const { data: resp } = await ipPoolAPI.list();
      if (resp.success) setPools(resp.data || []);
    } catch { /* */ }
    setLoading(false);
  };

  useEffect(() => { fetchPools(); }, []);

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    setModalOpen(true);
  };

  const openEdit = (record: IPPool) => {
    setEditing(record);
    form.setFieldsValue({
      ...record,
      nameservers: record.nameservers?.join(', ') || '',
    });
    setModalOpen(true);
  };

  const handleSubmit = async () => {
    try {
      const raw = await form.validateFields();
      const values = {
        ...raw,
        nameservers: raw.nameservers
          ? String(raw.nameservers).split(',').map((s: string) => s.trim()).filter(Boolean)
          : [],
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
      fetchPools();
    } catch { /* validation */ }
  };

  const handleDelete = async (id: string) => {
    const { data: resp } = await ipPoolAPI.delete(id);
    if (resp.success) {
      message.success('Pool deleted');
      fetchPools();
      if (selectedPool?.id === id) {
        setSelectedPool(null);
        setAddresses([]);
      }
    } else message.error(resp.error);
  };

  // Address methods
  const fetchAddresses = async (pool: IPPool) => {
    setSelectedPool(pool);
    setAddrLoading(true);
    try {
      const { data: resp } = await ipPoolAPI.listAddresses(pool.id);
      if (resp.success) setAddresses(resp.data || []);
    } catch { /* */ }
    setAddrLoading(false);
  };

  const openCreateAddr = () => {
    setEditingAddr(null);
    addrForm.resetFields();
    addrForm.setFieldsValue({ status: 'available' });
    setAddrModalOpen(true);
  };

  const openEditAddr = (record: IPAddress) => {
    setEditingAddr(record);
    addrForm.setFieldsValue(record);
    setAddrModalOpen(true);
  };

  const handleAddrSubmit = async () => {
    if (!selectedPool) return;
    try {
      const values = await addrForm.validateFields();
      if (editingAddr) {
        const { data: resp } = await ipPoolAPI.updateAddress(selectedPool.id, editingAddr.id, values);
        if (resp.success) message.success('Address updated');
        else { message.error(resp.error); return; }
      } else {
        const { data: resp } = await ipPoolAPI.createAddress(selectedPool.id, values);
        if (resp.success) message.success('Address added');
        else { message.error(resp.error); return; }
      }
      setAddrModalOpen(false);
      fetchAddresses(selectedPool);
      fetchPools();
    } catch { /* validation */ }
  };

  const handleDeleteAddr = async (addrId: string) => {
    if (!selectedPool) return;
    const { data: resp } = await ipPoolAPI.deleteAddress(selectedPool.id, addrId);
    if (resp.success) {
      message.success('Address deleted');
      fetchAddresses(selectedPool);
      fetchPools();
    } else message.error(resp.error);
  };

  const poolColumns: ColumnsType<IPPool> = [
    { title: 'Network', dataIndex: 'network', key: 'network', sorter: (a, b) => a.network.localeCompare(b.network) },
    { title: 'Gateway', dataIndex: 'gateway', key: 'gateway' },
    { title: 'Netmask', dataIndex: 'netmask', key: 'netmask' },
    { title: 'VRF', dataIndex: 'vrf', key: 'vrf', render: (v: string) => v || '-',
      filters: [...new Set(pools.map(p => p.vrf).filter(Boolean))].map(v => ({ text: v, value: v })),
      onFilter: (value, record) => record.vrf === value,
    },
    { title: 'Description', dataIndex: 'description', key: 'description', ellipsis: true },
    { title: 'Usage', key: 'usage', width: 180, render: (_, r) => {
      const pct = r.total_ips > 0 ? Math.round((r.used_ips / r.total_ips) * 100) : 0;
      return (
        <Space direction="vertical" size={0} style={{ width: '100%' }}>
          <Progress percent={pct} size="small" strokeColor={pct > 80 ? '#ff4d4f' : undefined} />
          <span style={{ fontSize: 12 }}>{r.used_ips} / {r.total_ips} used</span>
        </Space>
      );
    }},
    {
      title: 'Actions', key: 'actions', width: 200,
      render: (_, record) => (
        <Space>
          <Button size="small" onClick={() => fetchAddresses(record)}>Addresses</Button>
          <Button size="small" icon={<EditOutlined />} onClick={() => openEdit(record)} />
          <Popconfirm title="Delete this pool and all its addresses?" onConfirm={() => handleDelete(record.id)}>
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const addrColumns: ColumnsType<IPAddress> = [
    { title: 'Address', dataIndex: 'address', key: 'address', sorter: (a, b) => a.address.localeCompare(b.address) },
    { title: 'Status', dataIndex: 'status', key: 'status', render: (v: string) => <Tag color={statusColors[v] || 'default'}>{v}</Tag>,
      filters: [
        { text: 'Available', value: 'available' },
        { text: 'Assigned', value: 'assigned' },
        { text: 'Reserved', value: 'reserved' },
      ],
      onFilter: (value, record) => record.status === value,
    },
    { title: 'Server ID', dataIndex: 'server_id', key: 'server_id', render: (v: string) => v ? <Tag>{v.slice(0, 8)}...</Tag> : '-', ellipsis: true },
    { title: 'Note', dataIndex: 'note', key: 'note', ellipsis: true },
    {
      title: 'Actions', key: 'actions', width: 120,
      render: (_, record) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => openEditAddr(record)} />
          <Popconfirm title="Delete this address?" onConfirm={() => handleDeleteAddr(record.id)}>
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
        <Table columns={poolColumns} dataSource={pools} rowKey="id" loading={loading} size="small" />
      </Card>

      {selectedPool && (
        <Card
          title={`Addresses — ${selectedPool.network} (${selectedPool.gateway})`}
          style={{ marginTop: 16 }}
          extra={
            <Space>
              <Button icon={<ReloadOutlined />} onClick={() => fetchAddresses(selectedPool)}>Refresh</Button>
              <Button type="primary" icon={<PlusOutlined />} onClick={openCreateAddr}>Add Address</Button>
            </Space>
          }
        >
          <Table columns={addrColumns} dataSource={addresses} rowKey="id" loading={addrLoading} size="small" />
        </Card>
      )}

      {/* Pool Modal */}
      <Modal title={editing ? 'Edit Pool' : 'Create Pool'} open={modalOpen} onOk={handleSubmit} onCancel={() => setModalOpen(false)} okText={editing ? 'Update' : 'Create'}>
        <Form form={form} layout="vertical">
          <Form.Item name="network" label="Network (CIDR)" rules={[{ required: true }]}>
            <Input placeholder="10.0.0.0/24" />
          </Form.Item>
          <Form.Item name="gateway" label="Gateway" rules={[{ required: true }]}>
            <Input placeholder="10.0.0.1" />
          </Form.Item>
          <Form.Item name="netmask" label="Netmask">
            <Input placeholder="255.255.255.0" />
          </Form.Item>
          <Form.Item name="vrf" label="VRF">
            <Input placeholder="default (empty = no VRF)" />
          </Form.Item>
          <Form.Item name="nameservers" label="Nameservers">
            <Input placeholder="8.8.8.8, 8.8.4.4 (comma-separated)" />
          </Form.Item>
          <Form.Item name="tenant_id" label="Tenant">
            <Select
              allowClear
              placeholder="Select tenant (optional)"
              options={tenants.map((t) => ({ label: t.name, value: t.id }))}
            />
          </Form.Item>
          <Form.Item name="description" label="Description">
            <Input placeholder="Production subnet" />
          </Form.Item>
        </Form>
      </Modal>

      {/* Address Modal */}
      <Modal title={editingAddr ? 'Edit Address' : 'Add Address'} open={addrModalOpen} onOk={handleAddrSubmit} onCancel={() => setAddrModalOpen(false)} okText={editingAddr ? 'Update' : 'Add'}>
        <Form form={addrForm} layout="vertical">
          {!editingAddr && (
            <Form.Item name="address" label="IP Address" rules={[{ required: true }]}>
              <Input placeholder="10.0.0.10" />
            </Form.Item>
          )}
          <Form.Item name="status" label="Status">
            <Select options={[
              { label: 'Available', value: 'available' },
              { label: 'Assigned', value: 'assigned' },
              { label: 'Reserved', value: 'reserved' },
            ]} />
          </Form.Item>
          <Form.Item name="server_id" label="Server ID (Optional)">
            <Input placeholder="Server UUID" allowClear />
          </Form.Item>
          <Form.Item name="note" label="Note">
            <Input placeholder="Note" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default IPManagementPage;
