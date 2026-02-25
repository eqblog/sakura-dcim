import React, { useEffect, useState } from 'react';
import {
  Table,
  Card,
  Button,
  Space,
  Tag,
  Typography,
  Modal,
  Form,
  Input,
  Select,
  message,
  Popconfirm,
  Alert,
} from 'antd';
import {
  PlusOutlined,
  ReloadOutlined,
  SyncOutlined,
  CopyOutlined,
  EditOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import { agentAPI } from '../../api';
import type { Agent, PaginatedResult } from '../../types';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';

dayjs.extend(relativeTime);

const { Title, Text, Paragraph } = Typography;

const AgentListPage: React.FC = () => {
  const [data, setData] = useState<PaginatedResult<Agent> | null>(null);
  const [loading, setLoading] = useState(false);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [createForm] = Form.useForm();
  const [creating, setCreating] = useState(false);
  const [newAgentToken, setNewAgentToken] = useState<string | null>(null);

  // Edit modal
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [editForm] = Form.useForm();
  const [editingAgent, setEditingAgent] = useState<Agent | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    fetchAgents();
  }, []);

  const fetchAgents = async () => {
    setLoading(true);
    try {
      const { data: resp } = await agentAPI.list({ page: 1, page_size: 100 });
      if (resp.success) setData(resp.data || null);
    } catch {
      message.error('Failed to load agents');
    }
    setLoading(false);
  };

  const handleCreate = async (values: any) => {
    setCreating(true);
    try {
      const { data: resp } = await agentAPI.create(values);
      if (resp.success && resp.data) {
        setNewAgentToken(resp.data.token);
        message.success('Agent created');
        createForm.resetFields();
        fetchAgents();
      }
    } catch {
      message.error('Failed to create agent');
    }
    setCreating(false);
  };

  const handleEdit = (agent: Agent) => {
    setEditingAgent(agent);
    editForm.setFieldsValue({
      name: agent.name,
      location: agent.location,
      capabilities: agent.capabilities || [],
    });
    setEditModalOpen(true);
  };

  const handleUpdate = async (values: any) => {
    if (!editingAgent) return;
    setSaving(true);
    try {
      const { data: resp } = await agentAPI.update(editingAgent.id, values);
      if (resp.success) {
        message.success('Agent updated');
        setEditModalOpen(false);
        setEditingAgent(null);
        fetchAgents();
      } else {
        message.error(resp.error || 'Update failed');
      }
    } catch {
      message.error('Failed to update agent');
    }
    setSaving(false);
  };

  const handleDelete = async (id: string) => {
    try {
      await agentAPI.delete(id);
      message.success('Agent deleted');
      fetchAgents();
    } catch {
      message.error('Failed to delete agent');
    }
  };

  const columns = [
    { title: 'Name', dataIndex: 'name', key: 'name' },
    { title: 'Location', dataIndex: 'location', key: 'location' },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag
          icon={status === 'online' ? <SyncOutlined spin /> : undefined}
          color={status === 'online' ? 'green' : status === 'error' ? 'red' : 'default'}
        >
          {status.toUpperCase()}
        </Tag>
      ),
    },
    { title: 'Version', dataIndex: 'version', key: 'version', render: (v: string) => v || '-' },
    {
      title: 'Capabilities',
      dataIndex: 'capabilities',
      key: 'capabilities',
      render: (caps: string[]) => (
        <Space size={2} wrap>
          {(caps || []).map((c) => <Tag key={c}>{c}</Tag>)}
        </Space>
      ),
    },
    {
      title: 'Last Seen',
      dataIndex: 'last_seen',
      key: 'last_seen',
      render: (t: string) => t ? dayjs(t).fromNow() : 'Never',
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 120,
      render: (_: any, record: Agent) => (
        <Space size="small">
          <Button type="text" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)} />
          <Popconfirm title="Delete this agent?" onConfirm={() => handleDelete(record.id)}>
            <Button type="text" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>Agents</Title>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={fetchAgents}>Refresh</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => { setCreateModalOpen(true); setNewAgentToken(null); }}>
            Add Agent
          </Button>
        </Space>
      </div>

      <Card>
        <Table
          dataSource={data?.items || []}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={false}
          size="middle"
        />
      </Card>

      <Modal
        title="Add Agent"
        open={createModalOpen}
        onCancel={() => { setCreateModalOpen(false); setNewAgentToken(null); }}
        footer={newAgentToken ? [
          <Button key="close" onClick={() => { setCreateModalOpen(false); setNewAgentToken(null); }}>Close</Button>
        ] : null}
      >
        {newAgentToken ? (
          <div>
            <Alert
              title="Agent Created Successfully"
              description="Save this token now. It will not be shown again."
              type="success"
              showIcon
              style={{ marginBottom: 16 }}
            />
            <Text strong>Agent Token:</Text>
            <Paragraph
              copyable={{ icon: <CopyOutlined /> }}
              code
              style={{ wordBreak: 'break-all', marginTop: 8 }}
            >
              {newAgentToken}
            </Paragraph>
          </div>
        ) : (
          <Form form={createForm} layout="vertical" onFinish={handleCreate}>
            <Form.Item name="name" label="Name" rules={[{ required: true }]}>
              <Input placeholder="DC1 Agent" />
            </Form.Item>
            <Form.Item name="location" label="Location">
              <Input placeholder="Frankfurt, DE" />
            </Form.Item>
            <Form.Item name="capabilities" label="Capabilities">
              <Select
                mode="multiple"
                placeholder="Select capabilities"
                options={[
                  { value: 'ipmi', label: 'IPMI' },
                  { value: 'pxe', label: 'PXE' },
                  { value: 'kvm', label: 'KVM' },
                  { value: 'snmp', label: 'SNMP' },
                ]}
              />
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" htmlType="submit" loading={creating}>Create</Button>
                <Button onClick={() => setCreateModalOpen(false)}>Cancel</Button>
              </Space>
            </Form.Item>
          </Form>
        )}
      </Modal>

      {/* Edit Modal */}
      <Modal
        title="Edit Agent"
        open={editModalOpen}
        onCancel={() => { setEditModalOpen(false); setEditingAgent(null); }}
        footer={null}
        destroyOnClose
      >
        <Form form={editForm} layout="vertical" onFinish={handleUpdate}>
          <Form.Item name="name" label="Name" rules={[{ required: true }]}>
            <Input placeholder="DC1 Agent" />
          </Form.Item>
          <Form.Item name="location" label="Location">
            <Input placeholder="Frankfurt, DE" />
          </Form.Item>
          <Form.Item name="capabilities" label="Capabilities">
            <Select
              mode="multiple"
              placeholder="Select capabilities"
              options={[
                { value: 'ipmi', label: 'IPMI' },
                { value: 'pxe', label: 'PXE' },
                { value: 'kvm', label: 'KVM' },
                { value: 'snmp', label: 'SNMP' },
              ]}
            />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => { setEditModalOpen(false); setEditingAgent(null); }}>Cancel</Button>
              <Button type="primary" htmlType="submit" loading={saving}>Save</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default AgentListPage;
