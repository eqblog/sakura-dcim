import React, { useEffect, useState } from 'react';
import { useParams, useNavigate, useSearchParams } from 'react-router-dom';
import { Card, Tabs, Button, Spin, Typography, Tag, Space, message } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { serverAPI } from '../../api';
import type { Server, ServerStatus } from '../../types';
import OverviewTab from './tabs/OverviewTab';
import PowerTab from './tabs/PowerTab';
import KvmTab from './tabs/KvmTab';
import ReinstallTab from './tabs/ReinstallTab';
import BandwidthTab from './tabs/BandwidthTab';
import InventoryTab from './tabs/InventoryTab';

const { Title } = Typography;

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
    { key: 'overview', label: 'Overview', children: <OverviewTab server={server} /> },
    { key: 'power', label: 'Power', children: <PowerTab /> },
    { key: 'kvm', label: 'KVM', children: <KvmTab /> },
    { key: 'reinstall', label: 'Reinstall', children: <ReinstallTab /> },
    { key: 'bandwidth', label: 'Bandwidth', children: <BandwidthTab /> },
    { key: 'inventory', label: 'Inventory', children: <InventoryTab /> },
  ];

  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 16 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/servers')}>
          Back
        </Button>
        <Space>
          <Title level={4} style={{ margin: 0 }}>
            {server.hostname || server.label || server.id}
          </Title>
          <Tag color={statusColors[server.status]}>{server.status.toUpperCase()}</Tag>
        </Space>
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
