import React, { useEffect, useState } from 'react';
import { Row, Col, Card, Statistic, Typography, Tag, Table, Space } from 'antd';
import {
  CloudServerOutlined,
  ApiOutlined,
  CheckCircleOutlined,
  WarningOutlined,
  CloseCircleOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { serverAPI, agentAPI } from '../../api';
import type { Server, Agent } from '../../types';

const { Title } = Typography;

const statusColors: Record<string, string> = {
  active: 'green',
  online: 'green',
  provisioning: 'blue',
  reinstalling: 'orange',
  offline: 'default',
  error: 'red',
};

const DashboardPage: React.FC = () => {
  const [servers, setServers] = useState<Server[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    setLoading(true);
    try {
      const [serverResp, agentResp] = await Promise.all([
        serverAPI.list({ page: 1, page_size: 10 }),
        agentAPI.list({ page: 1, page_size: 100 }),
      ]);
      if (serverResp.data.success) setServers(serverResp.data.data?.items || []);
      if (agentResp.data.success) setAgents(agentResp.data.data?.items || []);
    } catch {
      // ignore
    }
    setLoading(false);
  };

  const serverStats = {
    total: servers.length,
    active: servers.filter((s) => s.status === 'active').length,
    offline: servers.filter((s) => s.status === 'offline').length,
    error: servers.filter((s) => s.status === 'error').length,
  };

  const agentStats = {
    total: agents.length,
    online: agents.filter((a) => a.status === 'online').length,
    offline: agents.filter((a) => a.status === 'offline').length,
  };

  const recentServerColumns = [
    { title: 'Hostname', dataIndex: 'hostname', key: 'hostname' },
    { title: 'IP', dataIndex: 'primary_ip', key: 'primary_ip' },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={statusColors[status] || 'default'}>{status.toUpperCase()}</Tag>
      ),
    },
    {
      title: 'CPU',
      key: 'cpu',
      render: (_: any, r: Server) => r.cpu_model ? `${r.cpu_model} (${r.cpu_cores}c)` : '-',
    },
    {
      title: 'RAM',
      key: 'ram',
      render: (_: any, r: Server) => r.ram_mb ? `${Math.round(r.ram_mb / 1024)} GB` : '-',
    },
  ];

  return (
    <div>
      <Title level={4}>Dashboard</Title>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Total Servers"
              value={serverStats.total}
              prefix={<CloudServerOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Active Servers"
              value={serverStats.active}
              prefix={<CheckCircleOutlined style={{ color: '#52c41a' }} />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Agents Online"
              value={agentStats.online}
              suffix={`/ ${agentStats.total}`}
              prefix={<ApiOutlined style={{ color: '#1890ff' }} />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Issues"
              value={serverStats.error + serverStats.offline}
              prefix={<WarningOutlined style={{ color: serverStats.error > 0 ? '#ff4d4f' : '#d9d9d9' }} />}
              valueStyle={{ color: serverStats.error > 0 ? '#ff4d4f' : undefined }}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={16}>
          <Card title="Recent Servers" extra={<a href="/servers">View All</a>}>
            <Table
              dataSource={servers}
              columns={recentServerColumns}
              rowKey="id"
              loading={loading}
              pagination={false}
              size="small"
            />
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title="Agent Status">
            <Space direction="vertical" style={{ width: '100%' }}>
              {agents.map((agent) => (
                <div
                  key={agent.id}
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    padding: '8px 0',
                    borderBottom: '1px solid #f0f0f0',
                  }}
                >
                  <div>
                    <div style={{ fontWeight: 500 }}>{agent.name}</div>
                    <div style={{ fontSize: 12, color: '#999' }}>{agent.location}</div>
                  </div>
                  <Tag
                    icon={agent.status === 'online' ? <SyncOutlined spin /> : <CloseCircleOutlined />}
                    color={statusColors[agent.status]}
                  >
                    {agent.status}
                  </Tag>
                </div>
              ))}
              {agents.length === 0 && !loading && (
                <div style={{ textAlign: 'center', color: '#999', padding: 20 }}>
                  No agents registered
                </div>
              )}
            </Space>
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default DashboardPage;
