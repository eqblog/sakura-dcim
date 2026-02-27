import React, { useEffect, useState } from 'react';
import { useParams, useNavigate, useSearchParams } from 'react-router-dom';
import { Card, Tabs, Button, Spin, Typography, Tag, Space, message, theme } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { serverAPI } from '../../api';
import type { Server, ServerStatus } from '../../types';
import OverviewTab from './tabs/OverviewTab';
import PowerTab from './tabs/PowerTab';
import KvmTab from './tabs/KvmTab';
import ReinstallTab from './tabs/ReinstallTab';
import SensorsTab from './tabs/SensorsTab';
import BandwidthTab from './tabs/BandwidthTab';
import NetworkTab from './tabs/NetworkTab';
import InventoryTab from './tabs/InventoryTab';
import IPTab from './tabs/IPTab';
import ProvisionTab from './tabs/ProvisionTab';

const { Title, Text } = Typography;

const statusColors: Record<ServerStatus, string> = {
  active: 'green',
  provisioning: 'blue',
  reinstalling: 'orange',
  offline: 'default',
  error: 'red',
};

const ServerDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { token } = theme.useToken();
  const [searchParams, setSearchParams] = useSearchParams();
  const [server, setServer] = useState<Server | null>(null);
  const [loading, setLoading] = useState(true);

  const activeTab = searchParams.get('tab') || 'overview';

  useEffect(() => {
    if (id) fetchServer();
  }, [id]);

  const fetchServer = async () => {
    setLoading(true);
    try {
      const { data: resp } = await serverAPI.get(id!);
      if (resp.success && resp.data) {
        setServer(resp.data);
      } else {
        message.error(resp.error || 'Server not found');
      }
    } catch {
      message.error('Failed to load server details');
    }
    setLoading(false);
  };

  const handleTabChange = (key: string) => {
    if (key === 'overview') {
      setSearchParams({});
    } else {
      setSearchParams({ tab: key });
    }
  };

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }

  if (!server) {
    return (
      <div>
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/servers')}>
          Back to Servers
        </Button>
        <Card style={{ marginTop: 16, textAlign: 'center' }}>
          <Title level={4}>Server not found</Title>
        </Card>
      </div>
    );
  }

  const tabItems = [
    { key: 'overview', label: 'Overview', children: <OverviewTab server={server} onUpdated={setServer} /> },
    { key: 'power', label: 'Power', children: <PowerTab serverId={server.id} /> },
    { key: 'sensors', label: 'Sensors', children: <SensorsTab serverId={server.id} /> },
    { key: 'kvm', label: 'KVM', children: <KvmTab serverId={server.id} /> },
    { key: 'provision', label: 'Provision', children: <ProvisionTab serverId={server.id} server={server} /> },
    { key: 'reinstall', label: 'Reinstall', children: <ReinstallTab serverId={server.id} /> },
    { key: 'bandwidth', label: 'Bandwidth', children: <BandwidthTab serverId={server.id} /> },
    { key: 'network', label: 'Network', children: <NetworkTab serverId={server.id} /> },
    { key: 'inventory', label: 'Inventory', children: <InventoryTab serverId={server.id} /> },
    { key: 'ips', label: 'IP Addresses', children: <IPTab serverId={server.id} /> },
  ];

  return (
    <div>
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 16,
          marginBottom: 24,
          paddingBottom: 16,
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
        }}
      >
        <Button
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate('/servers')}
          type="text"
        />
        <div style={{ flex: 1 }}>
          <Space align="center">
            <Title level={4} style={{ margin: 0 }}>
              {server.hostname || server.label || server.id}
            </Title>
            <Tag color={statusColors[server.status]}>{server.status.toUpperCase()}</Tag>
          </Space>
          {server.primary_ip && (
            <div>
              <Text type="secondary" style={{ fontSize: 13 }}>
                {server.primary_ip}
                {server.label && server.hostname ? ` — ${server.label}` : ''}
              </Text>
            </div>
          )}
        </div>
      </div>

      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={handleTabChange}
          items={tabItems}
        />
      </Card>
    </div>
  );
};

export default ServerDetailPage;
