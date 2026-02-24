import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Card,
  Descriptions,
  Tag,
  Button,
  Space,
  Typography,
  Spin,
  message,
} from 'antd';
import {
  ArrowLeftOutlined,
  SyncOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { agentAPI } from '../../api';
import type { Agent } from '../../types';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';

dayjs.extend(relativeTime);

const { Title } = Typography;

const statusConfig: Record<string, { color: string; icon?: React.ReactNode }> = {
  online: { color: 'green', icon: <SyncOutlined spin /> },
  offline: { color: 'default' },
  error: { color: 'red' },
};

const AgentDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [agent, setAgent] = useState<Agent | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchAgent = async () => {
    if (!id) return;
    setLoading(true);
    try {
      const { data: resp } = await agentAPI.get(id);
      if (resp.success && resp.data) {
        setAgent(resp.data);
      }
    } catch {
      message.error('Failed to load agent');
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchAgent();
  }, [id]);

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }

  if (!agent) {
    return (
      <Card>
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/agents')}>
          Back to Agents
        </Button>
        <div style={{ textAlign: 'center', padding: 40 }}>Agent not found</div>
      </Card>
    );
  }

  const cfg = statusConfig[agent.status] || statusConfig.offline;

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/agents')}>
            Back
          </Button>
          <Title level={4} style={{ margin: 0 }}>{agent.name}</Title>
          <Tag icon={cfg.icon} color={cfg.color}>{agent.status.toUpperCase()}</Tag>
        </Space>
        <Button icon={<ReloadOutlined />} onClick={fetchAgent}>Refresh</Button>
      </div>

      <Card title="Agent Information">
        <Descriptions column={2} bordered size="small">
          <Descriptions.Item label="ID">{agent.id}</Descriptions.Item>
          <Descriptions.Item label="Name">{agent.name}</Descriptions.Item>
          <Descriptions.Item label="Location">{agent.location || '-'}</Descriptions.Item>
          <Descriptions.Item label="Version">{agent.version || '-'}</Descriptions.Item>
          <Descriptions.Item label="Status">
            <Tag icon={cfg.icon} color={cfg.color}>{agent.status.toUpperCase()}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="Last Seen">
            {agent.last_seen ? `${dayjs(agent.last_seen).format('YYYY-MM-DD HH:mm:ss')} (${dayjs(agent.last_seen).fromNow()})` : 'Never'}
          </Descriptions.Item>
          <Descriptions.Item label="Capabilities" span={2}>
            <Space size={4} wrap>
              {(agent.capabilities || []).length > 0
                ? agent.capabilities.map((c) => <Tag key={c} color="blue">{c}</Tag>)
                : '-'}
            </Space>
          </Descriptions.Item>
          <Descriptions.Item label="Created At">
            {dayjs(agent.created_at).format('YYYY-MM-DD HH:mm:ss')}
          </Descriptions.Item>
        </Descriptions>
      </Card>
    </div>
  );
};

export default AgentDetailPage;
