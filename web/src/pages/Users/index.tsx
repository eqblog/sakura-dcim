import React, { useEffect, useState, useCallback } from 'react';
import { Table, Card, Button, Space, Tag, Typography, message, Popconfirm } from 'antd';
import { PlusOutlined, ReloadOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { userAPI, roleAPI } from '../../api';
import type { User, Role, PaginatedResult } from '../../types';
import UserFormModal from './UserFormModal';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';

dayjs.extend(relativeTime);
const { Title } = Typography;

const UsersPage: React.FC = () => {
  const [data, setData] = useState<PaginatedResult<User> | null>(null);
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const fetchUsers = useCallback(async (p = page) => {
    setLoading(true);
    try {
      const { data: resp } = await userAPI.list({ page: p, page_size: 20 });
      if (resp.success) setData(resp.data || null);
    } catch {
      message.error('Failed to load users');
    }
    setLoading(false);
  }, [page]);

  const fetchRoles = async () => {
    try {
      const { data: resp } = await roleAPI.list();
      if (resp.success && resp.data) setRoles(resp.data);
    } catch { /* ignore */ }
  };

  useEffect(() => { fetchRoles(); }, []);
  useEffect(() => { fetchUsers(page); }, [page]);

  const openCreate = () => { setEditingUser(null); setModalOpen(true); };
  const openEdit = (user: User) => { setEditingUser(user); setModalOpen(true); };
  const closeModal = () => { setModalOpen(false); setEditingUser(null); };

  const handleSubmit = async (values: any) => {
    setSubmitting(true);
    try {
      const payload = { ...values };
      if (editingUser && !payload.password) delete payload.password;
      if (editingUser) {
        await userAPI.update(editingUser.id, payload);
        message.success('User updated');
      } else {
        await userAPI.create(payload);
        message.success('User created');
      }
      closeModal();
      fetchUsers();
    } catch {
      message.error(editingUser ? 'Failed to update user' : 'Failed to create user');
    }
    setSubmitting(false);
  };

  const handleDelete = async (id: string) => {
    try {
      await userAPI.delete(id);
      message.success('User deleted');
      fetchUsers();
    } catch {
      message.error('Failed to delete user');
    }
  };

  const roleMap = Object.fromEntries(roles.map((r) => [r.id, r.name]));

  const columns = [
    { title: 'Name', dataIndex: 'name', key: 'name' },
    { title: 'Email', dataIndex: 'email', key: 'email' },
    {
      title: 'Role', dataIndex: 'role_id', key: 'role',
      render: (rid: string, record: User) => record.role?.name || roleMap[rid] || '-',
    },
    {
      title: 'Status', dataIndex: 'is_active', key: 'status',
      render: (active: boolean) => (
        <Tag color={active ? 'green' : 'default'}>{active ? 'Active' : 'Inactive'}</Tag>
      ),
    },
    {
      title: 'Last Login', dataIndex: 'last_login', key: 'last_login',
      render: (t: string) => (t ? dayjs(t).fromNow() : 'Never'),
    },
    {
      title: 'Actions', key: 'actions',
      render: (_: any, record: User) => (
        <Space size="small">
          <Button type="text" size="small" icon={<EditOutlined />} onClick={() => openEdit(record)}>
            Edit
          </Button>
          <Popconfirm title="Delete this user?" onConfirm={() => handleDelete(record.id)}>
            <Button type="text" size="small" danger icon={<DeleteOutlined />}>Delete</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>Users</Title>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => fetchUsers()}>Refresh</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>Add User</Button>
        </Space>
      </div>
      <Card>
        <Table
          dataSource={data?.items || []}
          columns={columns}
          rowKey="id"
          loading={loading}
          size="middle"
          pagination={{
            current: data?.page || 1,
            pageSize: data?.page_size || 20,
            total: data?.total || 0,
            onChange: setPage,
          }}
        />
      </Card>
      <UserFormModal
        open={modalOpen}
        user={editingUser}
        roles={roles}
        confirmLoading={submitting}
        onOk={handleSubmit}
        onCancel={closeModal}
      />
    </div>
  );
};

export default UsersPage;
