import React, { useEffect, useState, useCallback } from 'react';
import {
  Table,
  Card,
  Button,
  Space,
  Typography,
  Modal,
  Form,
  Input,
  ColorPicker,
  message,
  Popconfirm,
  Tooltip,
} from 'antd';
import {
  PlusOutlined,
  ReloadOutlined,
  DeleteOutlined,
  EditOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { tenantAPI } from '../../api';
import type { Tenant, PaginatedResult } from '../../types';
import dayjs from 'dayjs';

const { Title } = Typography;

const TenantsPage: React.FC = () => {
  const [data, setData] = useState<PaginatedResult<Tenant> | null>(null);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);

  // Create modal
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [createForm] = Form.useForm();
  const [creating, setCreating] = useState(false);

  // Edit modal
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [editForm] = Form.useForm();
  const [editing, setEditing] = useState(false);
  const [editingTenant, setEditingTenant] = useState<Tenant | null>(null);

  const fetchTenants = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, any> = { page, page_size: pageSize };
      const { data: resp } = await tenantAPI.list(params);
      if (resp.success) {
        setData(resp.data || null);
      }
    } catch {
      message.error('Failed to load tenants');
    }
    setLoading(false);
  }, [page, pageSize]);

  useEffect(() => {
    fetchTenants();
  }, [fetchTenants]);

  const handleCreate = async (values: any) => {
    setCreating(true);
    try {
      const payload = {
        ...values,
        primary_color:
          typeof values.primary_color === 'string'
            ? values.primary_color
            : values.primary_color?.toHexString?.() || undefined,
      };
      const { data: resp } = await tenantAPI.create(payload);
      if (resp.success) {
        message.success('Tenant created');
        setCreateModalOpen(false);
        createForm.resetFields();
        fetchTenants();
      } else {
        message.error(resp.error || 'Failed to create tenant');
      }
    } catch {
      message.error('Failed to create tenant');
    }
    setCreating(false);
  };

  const handleEdit = (tenant: Tenant) => {
    setEditingTenant(tenant);
    editForm.setFieldsValue({
      name: tenant.name,
      slug: tenant.slug,
      custom_domain: tenant.custom_domain || '',
      primary_color: tenant.primary_color || undefined,
    });
    setEditModalOpen(true);
  };

  const handleUpdate = async (values: any) => {
    if (!editingTenant) return;
    setEditing(true);
    try {
      const payload = {
        ...values,
        primary_color:
          typeof values.primary_color === 'string'
            ? values.primary_color
            : values.primary_color?.toHexString?.() || undefined,
      };
      const { data: resp } = await tenantAPI.update(editingTenant.id, payload);
      if (resp.success) {
        message.success('Tenant updated');
        setEditModalOpen(false);
        setEditingTenant(null);
        editForm.resetFields();
        fetchTenants();
      } else {
        message.error(resp.error || 'Failed to update tenant');
      }
    } catch {
      message.error('Failed to update tenant');
    }
    setEditing(false);
  };

  const handleDelete = async (id: string) => {
    try {
      const { data: resp } = await tenantAPI.delete(id);
      if (resp.success) {
        message.success('Tenant deleted');
        fetchTenants();
      } else {
        message.error(resp.error || 'Failed to delete tenant');
      }
    } catch {
      message.error('Failed to delete tenant');
    }
  };

  const columns: ColumnsType<Tenant> = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      ellipsis: true,
    },
    {
      title: 'Slug',
      dataIndex: 'slug',
      key: 'slug',
      render: (slug: string) => <code>{slug}</code>,
    },
    {
      title: 'Custom Domain',
      dataIndex: 'custom_domain',
      key: 'custom_domain',
      render: (domain: string) => domain || '-',
    },
    {
      title: 'Created At',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (t: string) => (t ? dayjs(t).format('YYYY-MM-DD HH:mm') : '-'),
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 120,
      render: (_, record) => (
        <Space size="small">
          <Tooltip title="Edit">
            <Button
              type="text"
              size="small"
              icon={<EditOutlined />}
              onClick={() => handleEdit(record)}
            />
          </Tooltip>
          <Popconfirm
            title="Delete this tenant?"
            description="This action cannot be undone."
            onConfirm={() => handleDelete(record.id)}
            okText="Delete"
            okButtonProps={{ danger: true }}
          >
            <Tooltip title="Delete">
              <Button type="text" size="small" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const tenantFormFields = (
    <>
      <Form.Item
        name="name"
        label="Name"
        rules={[{ required: true, message: 'Please enter tenant name' }]}
      >
        <Input placeholder="Acme Corp" />
      </Form.Item>
      <Form.Item
        name="slug"
        label="Slug"
        rules={[
          { required: true, message: 'Please enter tenant slug' },
          { pattern: /^[a-z0-9-]+$/, message: 'Slug must be lowercase alphanumeric with hyphens' },
        ]}
      >
        <Input placeholder="acme-corp" />
      </Form.Item>
      <Form.Item name="custom_domain" label="Custom Domain">
        <Input placeholder="panel.acme.com" />
      </Form.Item>
      <Form.Item name="primary_color" label="Primary Color">
        <ColorPicker format="hex" showText />
      </Form.Item>
    </>
  );

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>Tenants</Title>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={fetchTenants}>Refresh</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateModalOpen(true)}>
            Add Tenant
          </Button>
        </Space>
      </div>

      <Card>
        <Table
          dataSource={data?.items || []}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={{
            current: data?.page || 1,
            pageSize: data?.page_size || pageSize,
            total: data?.total || 0,
            showSizeChanger: false,
            showTotal: (total) => `Total ${total} tenants`,
            onChange: (p) => setPage(p),
          }}
          size="middle"
        />
      </Card>

      {/* Create Modal */}
      <Modal
        title="Create Tenant"
        open={createModalOpen}
        onCancel={() => {
          setCreateModalOpen(false);
          createForm.resetFields();
        }}
        footer={null}
        destroyOnClose
      >
        <Form form={createForm} layout="vertical" onFinish={handleCreate}>
          {tenantFormFields}
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => { setCreateModalOpen(false); createForm.resetFields(); }}>
                Cancel
              </Button>
              <Button type="primary" htmlType="submit" loading={creating}>
                Create
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* Edit Modal */}
      <Modal
        title="Edit Tenant"
        open={editModalOpen}
        onCancel={() => {
          setEditModalOpen(false);
          setEditingTenant(null);
          editForm.resetFields();
        }}
        footer={null}
        destroyOnClose
      >
        <Form form={editForm} layout="vertical" onFinish={handleUpdate}>
          {tenantFormFields}
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => { setEditModalOpen(false); setEditingTenant(null); editForm.resetFields(); }}>
                Cancel
              </Button>
              <Button type="primary" htmlType="submit" loading={editing}>
                Save
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default TenantsPage;
