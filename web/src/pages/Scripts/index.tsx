import React, { useEffect, useState } from 'react';
import { Card, Typography, Table, Button, Space, Modal, Form, Input, InputNumber, message, Popconfirm, Tag } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { scriptAPI } from '../../api';
import type { Script } from '../../types';

const { Title } = Typography;
const { TextArea } = Input;

const ScriptsPage: React.FC = () => {
  const [scripts, setScripts] = useState<Script[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Script | null>(null);
  const [form] = Form.useForm();

  const fetchScripts = async () => {
    setLoading(true);
    try {
      const { data: resp } = await scriptAPI.list();
      if (resp.success) setScripts(resp.data || []);
    } catch { /* */ }
    setLoading(false);
  };

  useEffect(() => { fetchScripts(); }, []);

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({ run_order: 0 });
    setModalOpen(true);
  };

  const openEdit = (record: Script) => {
    setEditing(record);
    form.setFieldsValue(record);
    setModalOpen(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (editing) {
        const { data: resp } = await scriptAPI.update(editing.id, values);
        if (resp.success) { message.success('Script updated'); }
        else { message.error(resp.error); return; }
      } else {
        const { data: resp } = await scriptAPI.create(values);
        if (resp.success) { message.success('Script created'); }
        else { message.error(resp.error); return; }
      }
      setModalOpen(false);
      fetchScripts();
    } catch { /* validation */ }
  };

  const handleDelete = async (id: string) => {
    const { data: resp } = await scriptAPI.delete(id);
    if (resp.success) { message.success('Deleted'); fetchScripts(); }
    else { message.error(resp.error); }
  };

  const columns: ColumnsType<Script> = [
    { title: 'Name', dataIndex: 'name', key: 'name', sorter: (a, b) => a.name.localeCompare(b.name) },
    { title: 'Description', dataIndex: 'description', key: 'description', ellipsis: true },
    { title: 'Order', dataIndex: 'run_order', key: 'run_order', width: 80, sorter: (a, b) => a.run_order - b.run_order },
    {
      title: 'Content', key: 'content', ellipsis: true,
      render: (_, record) => {
        const lines = record.content.split('\n').length;
        return <Tag>{lines} lines</Tag>;
      },
    },
    { title: 'Created', dataIndex: 'created_at', key: 'created_at', render: (v: string) => new Date(v).toLocaleDateString() },
    {
      title: 'Actions', key: 'actions', width: 120,
      render: (_, record) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => openEdit(record)} />
          <Popconfirm title="Delete this script?" onConfirm={() => handleDelete(record.id)}>
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4}>Post-Install Scripts</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>Add Script</Button>
      </div>

      <Card>
        <Table columns={columns} dataSource={scripts} rowKey="id" loading={loading} size="small" />
      </Card>

      <Modal
        title={editing ? 'Edit Script' : 'Create Script'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        width={800}
        okText={editing ? 'Update' : 'Create'}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true }]}>
            <Input placeholder="Install Docker" />
          </Form.Item>
          <Form.Item name="description" label="Description">
            <Input placeholder="Installs Docker CE on the target server" />
          </Form.Item>
          <Form.Item name="run_order" label="Run Order">
            <InputNumber min={0} max={999} style={{ width: 120 }} />
          </Form.Item>
          <Form.Item name="content" label="Script Content" rules={[{ required: true }]}>
            <TextArea
              rows={16}
              placeholder="#!/bin/bash&#10;apt-get update&#10;apt-get install -y docker.io"
              style={{ fontFamily: 'monospace', fontSize: 12 }}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default ScriptsPage;
