import React, { useState } from 'react';
import { Card, Table, Button, Space, Modal, Form, Input, Select, Tag, Popconfirm, message } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { ipPoolAPI } from '../../api';
import type { IPPool, IPAddress } from '../../types';

const statusColors: Record<string, string> = { available: 'green', assigned: 'blue', reserved: 'orange' };

interface Props {
  pool: IPPool;
  addresses: IPAddress[];
  loading: boolean;
  onRefresh: () => void;
  onPoolChanged: () => void;
}

const IPOverviewTab: React.FC<Props> = ({ pool, addresses, loading, onRefresh, onPoolChanged }) => {
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<IPAddress | null>(null);
  const [form] = Form.useForm();

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({ status: 'available' });
    setModalOpen(true);
  };

  const openEdit = (record: IPAddress) => {
    setEditing(record);
    form.setFieldsValue(record);
    setModalOpen(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (editing) {
        const { data: resp } = await ipPoolAPI.updateAddress(pool.id, editing.id, values);
        if (resp.success) message.success('Address updated');
        else { message.error(resp.error); return; }
      } else {
        const { data: resp } = await ipPoolAPI.createAddress(pool.id, values);
        if (resp.success) message.success('Address added');
        else { message.error(resp.error); return; }
      }
      setModalOpen(false);
      onRefresh();
      onPoolChanged();
    } catch { /* validation */ }
  };

  const handleDelete = async (addrId: string) => {
    const { data: resp } = await ipPoolAPI.deleteAddress(pool.id, addrId);
    if (resp.success) {
      message.success('Address deleted');
      onRefresh();
      onPoolChanged();
    } else message.error(resp.error);
  };

  const columns: ColumnsType<IPAddress> = [
    { title: 'Address', dataIndex: 'address', key: 'address', sorter: (a, b) => a.address.localeCompare(b.address) },
    {
      title: 'Status', dataIndex: 'status', key: 'status',
      render: (v: string) => <Tag color={statusColors[v] || 'default'}>{v}</Tag>,
      filters: [{ text: 'Available', value: 'available' }, { text: 'Assigned', value: 'assigned' }, { text: 'Reserved', value: 'reserved' }],
      onFilter: (value, record) => record.status === value,
    },
    { title: 'Server ID', dataIndex: 'server_id', key: 'server_id', render: (v: string) => v ? <Tag>{v.slice(0, 8)}...</Tag> : '-', ellipsis: true },
    { title: 'Note', dataIndex: 'note', key: 'note', ellipsis: true },
    {
      title: 'Actions', key: 'actions', width: 120,
      render: (_, record) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => openEdit(record)} />
          <Popconfirm title="Delete this address?" onConfirm={() => handleDelete(record.id)}>
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <Card
      title={`${addresses.length} addresses (${pool.used_ips} used / ${pool.total_ips} total)`}
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={onRefresh}>Refresh</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>Add Address</Button>
        </Space>
      }
    >
      <Table columns={columns} dataSource={addresses} rowKey="id" loading={loading} size="small" />
      <Modal title={editing ? 'Edit Address' : 'Add Address'} open={modalOpen} onOk={handleSubmit} onCancel={() => setModalOpen(false)} okText={editing ? 'Update' : 'Add'}>
        <Form form={form} layout="vertical">
          {!editing && <Form.Item name="address" label="IP Address" rules={[{ required: true }]}><Input placeholder="10.0.0.10" /></Form.Item>}
          <Form.Item name="status" label="Status">
            <Select options={[{ label: 'Available', value: 'available' }, { label: 'Assigned', value: 'assigned' }, { label: 'Reserved', value: 'reserved' }]} />
          </Form.Item>
          <Form.Item name="server_id" label="Server ID (Optional)"><Input placeholder="Server UUID" allowClear /></Form.Item>
          <Form.Item name="note" label="Note"><Input placeholder="Note" /></Form.Item>
        </Form>
      </Modal>
    </Card>
  );
};

export default IPOverviewTab;
