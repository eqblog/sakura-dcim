import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Table,
  Button,
  Modal,
  Form,
  Input,
  Tag,
  Space,
  Popconfirm,
  Typography,
  Checkbox,
  Divider,
  App,
  Tooltip,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  LockOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { roleAPI } from '../../api';
import type { Role } from '../../types';

const { Title } = Typography;

// ---------------------------------------------------------------------------
// Permission definitions grouped by category
// ---------------------------------------------------------------------------

interface PermissionGroup {
  category: string;
  permissions: string[];
}

const PERMISSION_GROUPS: PermissionGroup[] = [
  {
    category: 'Server',
    permissions: [
      'server.view',
      'server.create',
      'server.edit',
      'server.delete',
      'server.power',
    ],
  },
  {
    category: 'IPMI',
    permissions: ['ipmi.kvm', 'ipmi.sensors'],
  },
  {
    category: 'OS',
    permissions: ['os.reinstall', 'os.profile.manage'],
  },
  {
    category: 'RAID',
    permissions: ['raid.manage'],
  },
  {
    category: 'Bandwidth',
    permissions: ['bandwidth.view'],
  },
  {
    category: 'Switch',
    permissions: ['switch.manage'],
  },
  {
    category: 'Inventory',
    permissions: ['inventory.view', 'inventory.scan'],
  },
  {
    category: 'User',
    permissions: ['user.view', 'user.manage'],
  },
  {
    category: 'Role',
    permissions: ['role.manage'],
  },
  {
    category: 'Tenant',
    permissions: ['tenant.view', 'tenant.manage'],
  },
  {
    category: 'Agent',
    permissions: ['agent.manage'],
  },
  {
    category: 'IP',
    permissions: ['ip.manage'],
  },
  {
    category: 'Audit',
    permissions: ['audit.view'],
  },
  {
    category: 'Settings',
    permissions: ['settings.manage'],
  },
  {
    category: 'Script',
    permissions: ['script.manage'],
  },
  {
    category: 'Disk Layout',
    permissions: ['disk_layout.manage'],
  },
];

const ALL_PERMISSIONS = PERMISSION_GROUPS.flatMap((g) => g.permissions);

// ---------------------------------------------------------------------------
// Category colour map for tags
// ---------------------------------------------------------------------------

const CATEGORY_COLORS: Record<string, string> = {
  server: 'blue',
  ipmi: 'purple',
  os: 'cyan',
  raid: 'orange',
  bandwidth: 'green',
  switch: 'geekblue',
  inventory: 'gold',
  user: 'magenta',
  role: 'red',
  tenant: 'lime',
  agent: 'volcano',
  ip: 'processing',
  audit: 'default',
  settings: 'warning',
  script: 'success',
  disk_layout: '#87d068',
};

function permColor(perm: string): string {
  const prefix = perm.split('.')[0];
  return CATEGORY_COLORS[prefix] ?? 'default';
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

const RolesPage: React.FC = () => {
  const { message } = App.useApp();

  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingRole, setEditingRole] = useState<Role | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const [form] = Form.useForm();

  // -----------------------------------------------------------------------
  // Fetch
  // -----------------------------------------------------------------------

  const fetchRoles = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await roleAPI.list();
      setRoles(data.data ?? []);
    } catch {
      message.error('Failed to load roles');
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => {
    fetchRoles();
  }, [fetchRoles]);

  // -----------------------------------------------------------------------
  // Create / Edit
  // -----------------------------------------------------------------------

  const openCreate = () => {
    setEditingRole(null);
    form.resetFields();
    form.setFieldsValue({ name: '', permissions: [] });
    setModalOpen(true);
  };

  const openEdit = (role: Role) => {
    setEditingRole(role);
    form.setFieldsValue({
      name: role.name,
      permissions: role.permissions,
    });
    setModalOpen(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setSubmitting(true);

      if (editingRole) {
        await roleAPI.update(editingRole.id, values);
        message.success('Role updated');
      } else {
        await roleAPI.create(values);
        message.success('Role created');
      }

      setModalOpen(false);
      fetchRoles();
    } catch (err: any) {
      if (err?.errorFields) return; // validation error – let antd show it
      message.error(
        err?.response?.data?.error || 'Operation failed',
      );
    } finally {
      setSubmitting(false);
    }
  };

  // -----------------------------------------------------------------------
  // Delete
  // -----------------------------------------------------------------------

  const handleDelete = async (id: string) => {
    try {
      await roleAPI.delete(id);
      message.success('Role deleted');
      fetchRoles();
    } catch {
      message.error('Failed to delete role');
    }
  };

  // -----------------------------------------------------------------------
  // Select-all helpers for permission groups
  // -----------------------------------------------------------------------

  const PermissionGroupCheckboxes: React.FC = () => {
    const selected: string[] = Form.useWatch('permissions', form) ?? [];

    const toggleAll = (group: PermissionGroup, checked: boolean) => {
      const current = new Set(selected);
      group.permissions.forEach((p) =>
        checked ? current.add(p) : current.delete(p),
      );
      form.setFieldsValue({ permissions: Array.from(current) });
    };

    return (
      <>
        {PERMISSION_GROUPS.map((group) => {
          const allChecked = group.permissions.every((p) =>
            selected.includes(p),
          );
          const someChecked =
            !allChecked &&
            group.permissions.some((p) => selected.includes(p));

          return (
            <div key={group.category} style={{ marginBottom: 16 }}>
              <Checkbox
                indeterminate={someChecked}
                checked={allChecked}
                onChange={(e) => toggleAll(group, e.target.checked)}
                style={{ fontWeight: 600, marginBottom: 4 }}
              >
                {group.category}
              </Checkbox>

              <div style={{ paddingLeft: 24 }}>
                <Form.Item name="permissions" noStyle>
                  <Checkbox.Group
                    value={selected}
                    onChange={(vals) =>
                      form.setFieldsValue({ permissions: vals })
                    }
                  >
                    {group.permissions.map((perm) => (
                      <Checkbox
                        key={perm}
                        value={perm}
                        style={{ display: 'block', marginLeft: 0, marginBottom: 2 }}
                      >
                        {perm}
                      </Checkbox>
                    ))}
                  </Checkbox.Group>
                </Form.Item>
              </div>
            </div>
          );
        })}
      </>
    );
  };

  // -----------------------------------------------------------------------
  // Select all / deselect all
  // -----------------------------------------------------------------------

  const selectAllPermissions = () =>
    form.setFieldsValue({ permissions: [...ALL_PERMISSIONS] });

  const deselectAllPermissions = () =>
    form.setFieldsValue({ permissions: [] });

  // -----------------------------------------------------------------------
  // Table columns
  // -----------------------------------------------------------------------

  const columns: ColumnsType<Role> = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: Role) => (
        <Space>
          {name}
          {record.is_system && (
            <Tooltip title="System role — cannot be modified or deleted">
              <LockOutlined style={{ color: '#999' }} />
            </Tooltip>
          )}
        </Space>
      ),
    },
    {
      title: 'System',
      dataIndex: 'is_system',
      key: 'is_system',
      width: 100,
      render: (val: boolean) =>
        val ? <Tag color="red">System</Tag> : <Tag>Custom</Tag>,
    },
    {
      title: 'Permissions',
      dataIndex: 'permissions',
      key: 'permissions',
      render: (perms: string[]) =>
        perms && perms.length > 0 ? (
          <Space size={[4, 4]} wrap>
            {perms.map((p) => (
              <Tag key={p} color={permColor(p)}>
                {p}
              </Tag>
            ))}
          </Space>
        ) : (
          <Typography.Text type="secondary">None</Typography.Text>
        ),
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 160,
      render: (_: unknown, record: Role) => (
        <Space>
          <Button
            type="link"
            icon={<EditOutlined />}
            disabled={record.is_system}
            onClick={() => openEdit(record)}
          >
            Edit
          </Button>
          <Popconfirm
            title="Delete this role?"
            description="This action cannot be undone."
            onConfirm={() => handleDelete(record.id)}
            okText="Delete"
            okButtonProps={{ danger: true }}
            disabled={record.is_system}
          >
            <Button
              type="link"
              danger
              icon={<DeleteOutlined />}
              disabled={record.is_system}
            >
              Delete
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  // -----------------------------------------------------------------------
  // Render
  // -----------------------------------------------------------------------

  return (
    <div>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 16,
        }}
      >
        <Title level={4} style={{ margin: 0 }}>
          Roles
        </Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          Create Role
        </Button>
      </div>

      <Card>
        <Table<Role>
          rowKey="id"
          columns={columns}
          dataSource={roles}
          loading={loading}
          pagination={false}
        />
      </Card>

      {/* Create / Edit Modal */}
      <Modal
        title={editingRole ? 'Edit Role' : 'Create Role'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitting}
        destroyOnClose
        width={640}
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{ name: '', permissions: [] }}
          preserve={false}
        >
          <Form.Item
            name="name"
            label="Name"
            rules={[{ required: true, message: 'Please enter a role name' }]}
          >
            <Input placeholder="e.g. Operator" />
          </Form.Item>

          <Divider titlePlacement="start" plain>
            Permissions
          </Divider>

          <Space style={{ marginBottom: 12 }}>
            <Button size="small" onClick={selectAllPermissions}>
              Select All
            </Button>
            <Button size="small" onClick={deselectAllPermissions}>
              Deselect All
            </Button>
          </Space>

          <PermissionGroupCheckboxes />
        </Form>
      </Modal>
    </div>
  );
};

export default RolesPage;
