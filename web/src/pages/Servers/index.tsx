import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Table,
  Card,
  Button,
  Space,
  Tag,
  Input,
  Select,
  Typography,
  Tooltip,
  Popconfirm,
  message,
  Modal,
  Form,
} from 'antd';
import {
  PlusOutlined,
  SearchOutlined,
  ReloadOutlined,
  PoweroffOutlined,
  DesktopOutlined,
  DeleteOutlined,
  EditOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { serverAPI, agentAPI } from '../../api';
import type { Server, ServerStatus, PaginatedResult, Agent } from '../../types';

const { Title } = Typography;

const statusColors: Record<ServerStatus, string> = {
  active: 'green',
  provisioning: 'blue',
  reinstalling: 'orange',
  offline: 'default',
  error: 'red',
};

const ServerListPage: React.FC = () => {
  const navigate = useNavigate();
  const [data, setData] = useState<PaginatedResult<Server> | null>(null);
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<string | undefined>();
  const [page, setPage] = useState(1);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [createForm] = Form.useForm();
  const [creating, setCreating] = useState(false);
  const [agents, setAgents] = useState<Agent[]>([]);

  useEffect(() => {
    agentAPI.list({ page: 1, page_size: 200 }).then(({ data: resp }) => {
      if (resp.success) setAgents(resp.data?.items || []);
    });
  }, []);

  useEffect(() => {
    fetchServers();
  }, [page, statusFilter]);

  const fetchServers = async () => {
    setLoading(true);
    try {
      const params: Record<string, any> = { page, page_size: 20 };
      if (search) params.search = search;
      if (statusFilter) params.status = statusFilter;

      const { data: resp } = await serverAPI.list(params);
      if (resp.success) {
        setData(resp.data || null);
      }
    } catch {
      message.error('Failed to load servers');
    }
    setLoading(false);
  };

  const handleSearch = () => {
    setPage(1);
    fetchServers();
  };

  const handleDelete = async (id: string) => {
    try {
      await serverAPI.delete(id);
      message.success('Server deleted');
      fetchServers();
    } catch {
      message.error('Failed to delete server');
    }
  };

  const handleCreate = async (values: any) => {
    setCreating(true);
    try {
      await serverAPI.create(values);
      message.success('Server created');
      setCreateModalOpen(false);
      createForm.resetFields();
      fetchServers();
    } catch {
      message.error('Failed to create server');
    }
    setCreating(false);
  };

  const columns: ColumnsType<Server> = [
    {
      title: 'Hostname',
      dataIndex: 'hostname',
      key: 'hostname',
      render: (text, record) => (
        <a onClick={() => navigate(`/servers/${record.id}`)}>{text || '-'}</a>
      ),
    },
    {
      title: 'Label',
      dataIndex: 'label',
      key: 'label',
      ellipsis: true,
    },
    {
      title: 'IP Address',
      dataIndex: 'primary_ip',
      key: 'primary_ip',
      render: (ip) => ip || '-',
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (status: ServerStatus) => (
        <Tag color={statusColors[status]}>{status.toUpperCase()}</Tag>
      ),
    },
    {
      title: 'CPU',
      key: 'cpu',
      width: 200,
      render: (_, record) =>
        record.cpu_model ? `${record.cpu_model} (${record.cpu_cores}c)` : '-',
    },
    {
      title: 'RAM',
      key: 'ram',
      width: 100,
      render: (_, record) =>
        record.ram_mb ? `${Math.round(record.ram_mb / 1024)} GB` : '-',
    },
    {
      title: 'Tags',
      dataIndex: 'tags',
      key: 'tags',
      render: (tags: string[]) => (
        <Space size={2} wrap>
          {(tags || []).map((tag) => (
            <Tag key={tag} color="blue">{tag}</Tag>
          ))}
        </Space>
      ),
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 150,
      render: (_, record) => (
        <Space size="small">
          <Tooltip title="KVM Console">
            <Button
              type="text"
              size="small"
              icon={<DesktopOutlined />}
              onClick={() => navigate(`/servers/${record.id}?tab=kvm`)}
            />
          </Tooltip>
          <Tooltip title="Power">
            <Button
              type="text"
              size="small"
              icon={<PoweroffOutlined />}
              onClick={() => navigate(`/servers/${record.id}?tab=power`)}
            />
          </Tooltip>
          <Tooltip title="Edit">
            <Button
              type="text"
              size="small"
              icon={<EditOutlined />}
              onClick={() => navigate(`/servers/${record.id}`)}
            />
          </Tooltip>
          <Popconfirm title="Delete this server?" onConfirm={() => handleDelete(record.id)}>
            <Button type="text" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <div>
          <Title level={4} style={{ margin: 0 }}>Servers</Title>
          <Typography.Text type="secondary" style={{ fontSize: 13 }}>Manage your dedicated server infrastructure</Typography.Text>
        </div>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateModalOpen(true)}>
          Add Server
        </Button>
      </div>

      <Card>
        <Space style={{ marginBottom: 16 }}>
          <Input
            placeholder="Search hostname, IP..."
            prefix={<SearchOutlined />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            onPressEnter={handleSearch}
            style={{ width: 280 }}
          />
          <Select
            placeholder="Status"
            allowClear
            value={statusFilter}
            onChange={(v) => { setStatusFilter(v); setPage(1); }}
            style={{ width: 140 }}
            options={[
              { value: 'active', label: 'Active' },
              { value: 'provisioning', label: 'Provisioning' },
              { value: 'reinstalling', label: 'Reinstalling' },
              { value: 'offline', label: 'Offline' },
              { value: 'error', label: 'Error' },
            ]}
          />
          <Button icon={<ReloadOutlined />} onClick={fetchServers}>Refresh</Button>
        </Space>

        <Table
          dataSource={data?.items || []}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={{
            current: data?.page || 1,
            pageSize: data?.page_size || 20,
            total: data?.total || 0,
            showSizeChanger: false,
            onChange: setPage,
          }}
          size="middle"
        />
      </Card>

      <Modal
        title="Add Server"
        open={createModalOpen}
        onCancel={() => setCreateModalOpen(false)}
        footer={null}
      >
        <Form form={createForm} layout="vertical" onFinish={handleCreate}>
          <Form.Item name="hostname" label="Hostname" rules={[{ required: true }]}>
            <Input placeholder="server01.example.com" />
          </Form.Item>
          <Form.Item name="label" label="Label">
            <Input placeholder="Web Server 1" />
          </Form.Item>
          <Form.Item name="agent_id" label="Agent">
            <Select
              allowClear
              placeholder="Select agent"
              options={agents.map((a) => ({ label: `${a.name} (${a.location || 'No location'})`, value: a.id }))}
            />
          </Form.Item>
          <Form.Item name="primary_ip" label="Primary IP">
            <Input placeholder="192.168.1.100" />
          </Form.Item>
          <Form.Item name="ipmi_ip" label="IPMI IP">
            <Input placeholder="10.0.0.100" />
          </Form.Item>
          <Form.Item name="ipmi_user" label="IPMI User">
            <Input placeholder="ADMIN" />
          </Form.Item>
          <Form.Item name="ipmi_pass" label="IPMI Password">
            <Input.Password placeholder="IPMI password" />
          </Form.Item>
          <Form.Item name="mac_address" label="MAC Address">
            <Input placeholder="AA:BB:CC:DD:EE:FF" />
          </Form.Item>
          <Form.Item name="bmc_type" label="BMC Type" initialValue="generic">
            <Select
              options={[
                { value: 'generic', label: 'Generic IPMI' },
                { value: 'dell_idrac', label: 'Dell iDRAC' },
                { value: 'hp_ilo', label: 'HPE iLO' },
                { value: 'supermicro', label: 'Supermicro IPMI' },
                { value: 'lenovo_xcc', label: 'Lenovo XClarity' },
                { value: 'huawei_ibmc', label: 'Huawei iBMC' },
              ]}
            />
          </Form.Item>
          <Form.Item name="tags" label="Tags">
            <Select mode="tags" placeholder="Press Enter to add tag" />
          </Form.Item>
          <Form.Item name="notes" label="Notes">
            <Input.TextArea rows={3} placeholder="Additional notes" />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={creating}>Create</Button>
              <Button onClick={() => setCreateModalOpen(false)}>Cancel</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default ServerListPage;
