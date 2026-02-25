import React, { useEffect, useState, useCallback, useRef } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Tag,
  Select,
  Form,
  Input,
  Row,
  Col,
  Typography,
  message,
  Popconfirm,
  Alert,
  Tooltip,
} from 'antd';
import {
  SearchOutlined,
  PlayCircleOutlined,
  StopOutlined,
  CheckOutlined,
  CloseOutlined,
  EyeOutlined,
  DeleteOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { discoveryAPI, agentAPI } from '../../api';
import type { Agent, DiscoveredServer, PaginatedResult } from '../../types';
import ApproveModal from './ApproveModal';
import DetailDrawer from './DetailDrawer';

const { Title } = Typography;

const DiscoveryPage: React.FC = () => {
  // Agents
  const [agents, setAgents] = useState<Agent[]>([]);
  const [selectedAgentId, setSelectedAgentId] = useState<string>();

  // Discovery control
  const [discoveryActive, setDiscoveryActive] = useState(false);
  const [discoveryLoading, setDiscoveryLoading] = useState(false);
  const [startForm] = Form.useForm();

  // Discovered servers
  const [servers, setServers] = useState<DiscoveredServer[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [loading, setLoading] = useState(false);
  const [statusFilter, setStatusFilter] = useState<string>();

  // Modals
  const [approveServer, setApproveServer] = useState<DiscoveredServer | null>(null);
  const [detailServer, setDetailServer] = useState<DiscoveredServer | null>(null);

  // Auto-refresh timer
  const timerRef = useRef<ReturnType<typeof setInterval>>();

  // Fetch agents
  useEffect(() => {
    agentAPI.list({ page: 1, page_size: 200 }).then(({ data: resp }) => {
      if (resp.success) setAgents(resp.data?.items || []);
    });
  }, []);

  // Fetch discovery status when agent changes
  useEffect(() => {
    if (selectedAgentId) {
      discoveryAPI.status(selectedAgentId).then(({ data: resp }) => {
        if (resp.success && resp.data) {
          setDiscoveryActive(resp.data.active === true);
        }
      });
    } else {
      setDiscoveryActive(false);
    }
  }, [selectedAgentId]);

  // Fetch discovered servers
  const fetchServers = useCallback(() => {
    setLoading(true);
    const params: Record<string, any> = { page, page_size: pageSize };
    if (selectedAgentId) params.agent_id = selectedAgentId;
    if (statusFilter) params.status = statusFilter;

    discoveryAPI.listServers(params).then(({ data: resp }) => {
      if (resp.success && resp.data) {
        setServers(resp.data.items || []);
        setTotal(resp.data.total);
      }
    }).finally(() => setLoading(false));
  }, [page, pageSize, selectedAgentId, statusFilter]);

  useEffect(() => {
    fetchServers();
  }, [fetchServers]);

  // Auto-refresh while discovery is active
  useEffect(() => {
    if (discoveryActive) {
      timerRef.current = setInterval(fetchServers, 10000);
    } else if (timerRef.current) {
      clearInterval(timerRef.current);
    }
    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
    };
  }, [discoveryActive, fetchServers]);

  // Start discovery
  const handleStart = async () => {
    if (!selectedAgentId) {
      message.warning('Please select an agent first');
      return;
    }
    try {
      const values = await startForm.validateFields();
      setDiscoveryLoading(true);
      const { data: resp } = await discoveryAPI.start(selectedAgentId, values);
      if (resp.success) {
        message.success('Discovery started');
        setDiscoveryActive(true);
      } else {
        message.error(resp.error || 'Failed to start');
      }
    } catch {
      // validation error
    } finally {
      setDiscoveryLoading(false);
    }
  };

  // Stop discovery
  const handleStop = async () => {
    if (!selectedAgentId) return;
    setDiscoveryLoading(true);
    try {
      const { data: resp } = await discoveryAPI.stop(selectedAgentId);
      if (resp.success) {
        message.success('Discovery stopped');
        setDiscoveryActive(false);
      } else {
        message.error(resp.error || 'Failed to stop');
      }
    } finally {
      setDiscoveryLoading(false);
    }
  };

  // Reject
  const handleReject = async (id: string) => {
    const { data: resp } = await discoveryAPI.reject(id);
    if (resp.success) {
      message.success('Server rejected');
      fetchServers();
    } else {
      message.error(resp.error || 'Failed to reject');
    }
  };

  // Delete
  const handleDelete = async (id: string) => {
    const { data: resp } = await discoveryAPI.deleteServer(id);
    if (resp.success) {
      message.success('Deleted');
      fetchServers();
    } else {
      message.error(resp.error || 'Failed to delete');
    }
  };

  const statusColor: Record<string, string> = {
    pending: 'blue',
    approved: 'green',
    rejected: 'red',
  };

  const columns: ColumnsType<DiscoveredServer> = [
    {
      title: 'Status',
      dataIndex: 'status',
      width: 100,
      render: (s: string) => <Tag color={statusColor[s] || 'default'}>{s.toUpperCase()}</Tag>,
    },
    {
      title: 'MAC',
      dataIndex: 'mac_address',
      width: 150,
    },
    {
      title: 'System',
      key: 'system',
      render: (_, r) => `${r.system_vendor} ${r.system_product}`.trim() || '-',
    },
    {
      title: 'CPU',
      key: 'cpu',
      render: (_, r) => r.cpu_model ? `${r.cpu_model} (${r.cpu_cores}c)` : '-',
    },
    {
      title: 'RAM',
      dataIndex: 'ram_mb',
      width: 90,
      render: (v: number) => v > 0 ? `${Math.round(v / 1024)} GB` : '-',
    },
    {
      title: 'Disks',
      key: 'disks',
      width: 120,
      render: (_, r) => r.disk_count > 0 ? `${r.disk_count} (${r.disk_total_gb}GB)` : '-',
    },
    {
      title: 'NICs',
      dataIndex: 'nic_count',
      width: 60,
    },
    {
      title: 'BMC IP',
      dataIndex: 'bmc_ip',
      width: 130,
      render: (v: string) => v || '-',
    },
    {
      title: 'Discovered',
      dataIndex: 'discovered_at',
      width: 160,
      render: (v: string) => new Date(v).toLocaleString(),
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 160,
      render: (_, record) => (
        <Space size="small">
          <Tooltip title="Details">
            <Button size="small" icon={<EyeOutlined />} onClick={() => setDetailServer(record)} />
          </Tooltip>
          {record.status === 'pending' && (
            <>
              <Tooltip title="Approve">
                <Button size="small" type="primary" icon={<CheckOutlined />} onClick={() => setApproveServer(record)} />
              </Tooltip>
              <Tooltip title="Reject">
                <Popconfirm title="Reject this server?" onConfirm={() => handleReject(record.id)}>
                  <Button size="small" danger icon={<CloseOutlined />} />
                </Popconfirm>
              </Tooltip>
            </>
          )}
          <Tooltip title="Delete">
            <Popconfirm title="Delete this record?" onConfirm={() => handleDelete(record.id)}>
              <Button size="small" danger icon={<DeleteOutlined />} />
            </Popconfirm>
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Title level={4}>Hardware Discovery</Title>

      {/* Discovery Control Panel */}
      <Card title="Discovery Control" style={{ marginBottom: 24 }}>
        <Row gutter={16} align="middle">
          <Col span={6}>
            <label style={{ display: 'block', marginBottom: 4, fontWeight: 500 }}>Agent</label>
            <Select
              style={{ width: '100%' }}
              placeholder="Select agent"
              value={selectedAgentId}
              onChange={setSelectedAgentId}
              allowClear
            >
              {agents.map(a => (
                <Select.Option key={a.id} value={a.id}>
                  {a.name} ({a.location})
                  {a.status === 'online' ? ' ✓' : ' ✗'}
                </Select.Option>
              ))}
            </Select>
          </Col>
          <Col span={18}>
            {!discoveryActive ? (
              <Form form={startForm} layout="inline">
                <Form.Item name="dhcp_range_start" rules={[{ required: true, message: 'Required' }]}>
                  <Input placeholder="DHCP Start (e.g. 10.99.0.100)" style={{ width: 180 }} />
                </Form.Item>
                <Form.Item name="dhcp_range_end" rules={[{ required: true, message: 'Required' }]}>
                  <Input placeholder="DHCP End (e.g. 10.99.0.200)" style={{ width: 180 }} />
                </Form.Item>
                <Form.Item name="gateway" rules={[{ required: true, message: 'Required' }]}>
                  <Input placeholder="Gateway" style={{ width: 140 }} />
                </Form.Item>
                <Form.Item name="netmask" rules={[{ required: true, message: 'Required' }]}>
                  <Input placeholder="Netmask" style={{ width: 140 }} />
                </Form.Item>
                <Form.Item>
                  <Button
                    type="primary"
                    icon={<PlayCircleOutlined />}
                    loading={discoveryLoading}
                    onClick={handleStart}
                    disabled={!selectedAgentId}
                  >
                    Start Discovery
                  </Button>
                </Form.Item>
              </Form>
            ) : (
              <Space>
                <Alert
                  type="info"
                  showIcon
                  message="Discovery is active. Servers will appear below as they PXE boot and report hardware."
                  style={{ marginBottom: 0 }}
                />
                <Button
                  danger
                  icon={<StopOutlined />}
                  loading={discoveryLoading}
                  onClick={handleStop}
                >
                  Stop Discovery
                </Button>
              </Space>
            )}
          </Col>
        </Row>
      </Card>

      {/* Discovered Servers Table */}
      <Card
        title="Discovered Servers"
        extra={
          <Space>
            <Select
              style={{ width: 130 }}
              placeholder="Status"
              allowClear
              value={statusFilter}
              onChange={setStatusFilter}
            >
              <Select.Option value="pending">Pending</Select.Option>
              <Select.Option value="approved">Approved</Select.Option>
              <Select.Option value="rejected">Rejected</Select.Option>
            </Select>
            <Button icon={<ReloadOutlined />} onClick={fetchServers}>
              Refresh
            </Button>
          </Space>
        }
      >
        <Table
          rowKey="id"
          columns={columns}
          dataSource={servers}
          loading={loading}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            onChange: (p, ps) => { setPage(p); setPageSize(ps); },
          }}
          scroll={{ x: 1200 }}
          size="small"
        />
      </Card>

      {/* Approve Modal */}
      <ApproveModal
        open={!!approveServer}
        server={approveServer}
        agents={agents}
        onClose={() => setApproveServer(null)}
        onSuccess={() => { setApproveServer(null); fetchServers(); }}
      />

      {/* Detail Drawer */}
      <DetailDrawer
        open={!!detailServer}
        server={detailServer}
        onClose={() => setDetailServer(null)}
      />
    </div>
  );
};

export default DiscoveryPage;
