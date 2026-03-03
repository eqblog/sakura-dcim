import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { Spin, Typography, message } from 'antd';
import { serverAPI } from '../api';
import type { Server } from '../types';
import KvmTab from './Servers/tabs/KvmTab';

const { Title } = Typography;

const KvmPopupPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const [server, setServer] = useState<Server | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!id) return;
    (async () => {
      try {
        const { data: resp } = await serverAPI.get(id);
        if (resp.success && resp.data) setServer(resp.data);
        else message.error(resp.error || 'Server not found');
      } catch {
        message.error('Failed to load server');
      }
      setLoading(false);
    })();
  }, [id]);

  useEffect(() => {
    if (server) {
      document.title = `KVM — ${server.hostname || server.label || server.id}`;
    }
    return () => { document.title = 'Sakura DCIM'; };
  }, [server]);

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
        <Spin size="large" />
      </div>
    );
  }

  if (!server) {
    return (
      <div style={{ textAlign: 'center', padding: 60 }}>
        <Title level={4}>Server not found</Title>
      </div>
    );
  }

  return (
    <div style={{ padding: 16, height: '100vh', display: 'flex', flexDirection: 'column' }}>
      <div style={{ marginBottom: 8 }}>
        <Title level={5} style={{ margin: 0 }}>
          KVM — {server.hostname || server.label || server.id}
        </Title>
      </div>
      <div style={{ flex: 1 }}>
        <KvmTab server={server} />
      </div>
    </div>
  );
};

export default KvmPopupPage;
