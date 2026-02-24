import React, { useEffect, useState } from 'react';
import { Card, Typography, Table, Button, Space, Modal, Form, Input, message, Popconfirm } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { diskLayoutAPI } from '../../api';
import type { DiskLayout } from '../../types';

const { Title } = Typography;
const { TextArea } = Input;

const DiskLayoutsPage: React.FC = () => {
  const [layouts, setLayouts] = useState<DiskLayout[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<DiskLayout | null>(null);
  const [form] = Form.useForm();

  const fetchLayouts = async () => {
    setLoading(true);
    try {
      const { data: resp } = await diskLayoutAPI.list();
      if (resp.success) setLayouts(resp.data || []);
    } catch { /* */ }
    setLoading(false);
  };

  useEffect(() => { fetchLayouts(); }, []);

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    setModalOpen(true);
  };

  const openEdit = (record: DiskLayout) => {
    setEditing(record);
    form.setFieldsValue({
      ...record,
      layout: JSON.stringify(record.layout, null, 2),
    });
    setModalOpen(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      // Parse layout JSON
      let layout = {};
      try {
        layout = JSON.parse(values.layout || '{}');
      } catch {
        message.error('Invalid JSON in layout field');
        return;
      }
      const payload = { ...values, layout };

      if (editing) {
        const { data: resp } = await diskLayoutAPI.update(editing.id, payload);
        if (resp.success) { message.success('Disk layout updated'); }
        else { message.error(resp.error); return; }
      } else {
        const { data: resp } = await diskLayoutAPI.create(payload);
        if (resp.success) { message.success('Disk layout created'); }
        else { message.error(resp.error); return; }
      }
      setModalOpen(false);
      fetchLayouts();
    } catch { /* validation */ }
  };

  const handleDelete = async (id: string) => {
    const { data: resp } = await diskLayoutAPI.delete(id);
    if (resp.success) { message.success('Deleted'); fetchLayouts(); }
    else { message.error(resp.error); }
  };

  const columns: ColumnsType<DiskLayout> = [
    { title: 'Name', dataIndex: 'name', key: 'name', sorter: (a, b) => a.name.localeCompare(b.name) },
    { title: 'Description', dataIndex: 'description', key: 'description', ellipsis: true },
    {
      title: 'Partitions', key: 'partitions',
      render: (_, record) => {
        const layout = record.layout as any;
        if (layout?.partitions) return `${layout.partitions.length} partitions`;
        return '-';
      },
    },
    { title: 'Created', dataIndex: 'created_at', key: 'created_at', render: (v: string) => new Date(v).toLocaleDateString() },
    {
      title: 'Actions', key: 'actions', width: 120,
      render: (_, record) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => openEdit(record)} />
          <Popconfirm title="Delete this layout?" onConfirm={() => handleDelete(record.id)}>
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4}>Disk Layouts</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>Add Layout</Button>
      </div>

      <Card>
        <Table columns={columns} dataSource={layouts} rowKey="id" loading={loading} size="small" />
      </Card>

      <Modal
        title={editing ? 'Edit Disk Layout' : 'Create Disk Layout'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        width={700}
        okText={editing ? 'Update' : 'Create'}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true }]}>
            <Input placeholder="Standard 2-partition (boot + root)" />
          </Form.Item>
          <Form.Item name="description" label="Description">
            <Input placeholder="Boot partition + root ext4" />
          </Form.Item>
          <Form.Item name="layout" label="Layout (JSON)" rules={[{ required: true }]}>
            <TextArea
              rows={12}
              placeholder={`{
  "partitions": [
    { "mount": "/boot", "size": "1G", "fs": "ext4" },
    { "mount": "/", "size": "100%FREE", "fs": "ext4" }
  ]
}`}
              style={{ fontFamily: 'monospace', fontSize: 12 }}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default DiskLayoutsPage;
