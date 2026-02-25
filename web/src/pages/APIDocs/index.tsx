import React from 'react';
import { Card, Typography, Table, Tag, Space, Collapse } from 'antd';
import { ApiOutlined } from '@ant-design/icons';

const { Title, Text, Paragraph } = Typography;

const methodColors: Record<string, string> = {
  GET: 'green',
  POST: 'blue',
  PUT: 'orange',
  DELETE: 'red',
  WS: 'purple',
};

interface Endpoint {
  method: string;
  path: string;
  description: string;
  auth: boolean;
  permission?: string;
}

const endpoints: Record<string, Endpoint[]> = {
  Authentication: [
    { method: 'POST', path: '/api/v1/auth/login', description: 'Login with email and password', auth: false },
    { method: 'POST', path: '/api/v1/auth/refresh', description: 'Refresh access token', auth: false },
    { method: 'GET', path: '/api/v1/auth/me', description: 'Get current user info (includes tenant branding)', auth: true },
    { method: 'POST', path: '/api/v1/auth/logout', description: 'Logout', auth: true },
    { method: 'GET', path: '/api/v1/auth/branding', description: 'Public tenant branding by domain or slug', auth: false },
  ],
  Servers: [
    { method: 'GET', path: '/api/v1/servers', description: 'List servers (search, filter, paginate)', auth: true, permission: 'server.view' },
    { method: 'POST', path: '/api/v1/servers', description: 'Create a new server', auth: true, permission: 'server.create' },
    { method: 'GET', path: '/api/v1/servers/:id', description: 'Get server details', auth: true, permission: 'server.view' },
    { method: 'PUT', path: '/api/v1/servers/:id', description: 'Update server', auth: true, permission: 'server.edit' },
    { method: 'DELETE', path: '/api/v1/servers/:id', description: 'Delete server', auth: true, permission: 'server.delete' },
    { method: 'POST', path: '/api/v1/servers/:id/power', description: 'Power control (on/off/reset/cycle)', auth: true, permission: 'server.power' },
    { method: 'GET', path: '/api/v1/servers/:id/power', description: 'Power status', auth: true, permission: 'server.power' },
    { method: 'GET', path: '/api/v1/servers/:id/sensors', description: 'IPMI sensor readings', auth: true, permission: 'ipmi.sensors' },
    { method: 'POST', path: '/api/v1/servers/:id/kvm', description: 'Start KVM session', auth: true, permission: 'ipmi.kvm' },
    { method: 'DELETE', path: '/api/v1/servers/:id/kvm', description: 'Stop KVM session', auth: true, permission: 'ipmi.kvm' },
    { method: 'POST', path: '/api/v1/servers/:id/reinstall', description: 'Start OS reinstall', auth: true, permission: 'os.reinstall' },
    { method: 'GET', path: '/api/v1/servers/:id/reinstall/status', description: 'Get install task status', auth: true, permission: 'os.reinstall' },
    { method: 'GET', path: '/api/v1/servers/:id/bandwidth', description: 'Bandwidth stats', auth: true, permission: 'bandwidth.view' },
    { method: 'GET', path: '/api/v1/servers/:id/inventory', description: 'Hardware inventory', auth: true, permission: 'inventory.view' },
    { method: 'POST', path: '/api/v1/servers/:id/inventory/scan', description: 'Trigger inventory scan', auth: true, permission: 'inventory.scan' },
  ],
  Agents: [
    { method: 'GET', path: '/api/v1/agents', description: 'List all agents', auth: true, permission: 'agent.manage' },
    { method: 'POST', path: '/api/v1/agents', description: 'Register a new agent', auth: true, permission: 'agent.manage' },
    { method: 'WS', path: '/api/v1/agents/ws', description: 'Agent WebSocket connection', auth: false },
  ],
  'OS Management': [
    { method: 'GET', path: '/api/v1/os-profiles', description: 'List OS profiles', auth: true, permission: 'os_profile.manage' },
    { method: 'POST', path: '/api/v1/os-profiles', description: 'Create OS profile', auth: true, permission: 'os_profile.manage' },
    { method: 'GET', path: '/api/v1/disk-layouts', description: 'List disk layouts', auth: true, permission: 'disk_layout.manage' },
    { method: 'POST', path: '/api/v1/disk-layouts', description: 'Create disk layout', auth: true, permission: 'disk_layout.manage' },
    { method: 'GET', path: '/api/v1/scripts', description: 'List post-install scripts', auth: true, permission: 'script.manage' },
    { method: 'POST', path: '/api/v1/scripts', description: 'Create script', auth: true, permission: 'script.manage' },
  ],
  Network: [
    { method: 'GET', path: '/api/v1/switches', description: 'List switches', auth: true, permission: 'switch.manage' },
    { method: 'POST', path: '/api/v1/switches', description: 'Create switch', auth: true, permission: 'switch.manage' },
    { method: 'GET', path: '/api/v1/switches/:id/ports', description: 'List switch ports', auth: true, permission: 'switch.manage' },
    { method: 'POST', path: '/api/v1/switches/:id/ports/:portId/provision', description: 'Auto-provision port', auth: true, permission: 'switch.manage' },
    { method: 'GET', path: '/api/v1/ip-pools', description: 'List IP pools', auth: true, permission: 'ip.manage' },
    { method: 'POST', path: '/api/v1/ip-pools', description: 'Create IP pool', auth: true, permission: 'ip.manage' },
    { method: 'POST', path: '/api/v1/ip-pools/:id/assign', description: 'Auto-assign next available IP', auth: true, permission: 'ip.manage' },
  ],
  Administration: [
    { method: 'GET', path: '/api/v1/users', description: 'List users', auth: true, permission: 'user.manage' },
    { method: 'POST', path: '/api/v1/users', description: 'Create user', auth: true, permission: 'user.manage' },
    { method: 'GET', path: '/api/v1/roles', description: 'List roles', auth: true, permission: 'role.manage' },
    { method: 'GET', path: '/api/v1/tenants', description: 'List tenants', auth: true, permission: 'tenant.manage' },
    { method: 'GET', path: '/api/v1/tenants/:id/tree', description: 'Get tenant hierarchy tree', auth: true, permission: 'tenant.manage' },
    { method: 'GET', path: '/api/v1/audit-logs', description: 'Search audit logs', auth: true, permission: 'audit.view' },
  ],
  'WebSocket Protocol': [
    { method: 'WS', path: 'agent.heartbeat', description: 'Agent → Panel: periodic health + version', auth: false },
    { method: 'WS', path: 'ipmi.power.*', description: 'Panel → Agent: power on/off/reset/cycle/status', auth: false },
    { method: 'WS', path: 'ipmi.sensors', description: 'Panel → Agent: sensor readings', auth: false },
    { method: 'WS', path: 'ipmi.sol', description: 'Panel → Agent: Serial Over LAN control', auth: false },
    { method: 'WS', path: 'ipmi.kvm.start/stop', description: 'Panel → Agent: KVM Docker container lifecycle', auth: false },
    { method: 'WS', path: 'pxe.prepare/status', description: 'Bidirectional: PXE boot lifecycle', auth: false },
    { method: 'WS', path: 'inventory.scan/pxe', description: 'Panel → Agent: hardware scan or PXE inventory', auth: false },
    { method: 'WS', path: 'switch.provision/status', description: 'Panel → Agent: switch port auto-config', auth: false },
    { method: 'WS', path: 'snmp.poll', description: 'Panel → Agent: bandwidth counter polling', auth: false },
  ],
  Monitoring: [
    { method: 'GET', path: '/health', description: 'Health check', auth: false },
    { method: 'GET', path: '/metrics', description: 'Prometheus metrics endpoint', auth: false },
  ],
};

const APIDocsPage: React.FC = () => {
  const collapseItems = Object.entries(endpoints).map(([group, eps]) => ({
    key: group,
    label: (
      <Space>
        <Text strong>{group}</Text>
        <Tag>{eps.length} endpoints</Tag>
      </Space>
    ),
    children: (
      <Table
        dataSource={eps}
        rowKey={(r) => `${r.method}-${r.path}`}
        pagination={false}
        size="small"
        columns={[
          {
            title: 'Method',
            dataIndex: 'method',
            key: 'method',
            width: 80,
            render: (m: string) => <Tag color={methodColors[m] || 'default'}>{m}</Tag>,
          },
          {
            title: 'Path',
            dataIndex: 'path',
            key: 'path',
            render: (p: string) => <Text code style={{ fontSize: 12 }}>{p}</Text>,
          },
          {
            title: 'Description',
            dataIndex: 'description',
            key: 'description',
          },
          {
            title: 'Auth',
            dataIndex: 'auth',
            key: 'auth',
            width: 60,
            render: (a: boolean) => a ? <Tag color="blue">JWT</Tag> : <Tag>Public</Tag>,
          },
          {
            title: 'Permission',
            dataIndex: 'permission',
            key: 'permission',
            width: 140,
            render: (p?: string) => p ? <Tag color="orange">{p}</Tag> : '-',
          },
        ]}
      />
    ),
  }));

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>
          <ApiOutlined style={{ marginRight: 8 }} />
          API Documentation
        </Title>
      </div>

      <Card>
        <Paragraph>
          Base URL: <Text code>{window.location.origin}/api/v1</Text>
        </Paragraph>
        <Paragraph>
          Authentication: Bearer token in <Text code>Authorization</Text> header. Obtain via <Text code>POST /auth/login</Text>.
        </Paragraph>
        <Paragraph>
          All responses follow the format: <Text code>{'{ "success": true, "data": ... }'}</Text>
        </Paragraph>

        <Collapse items={collapseItems} defaultActiveKey={['Authentication', 'Servers']} />
      </Card>
    </div>
  );
};

export default APIDocsPage;
