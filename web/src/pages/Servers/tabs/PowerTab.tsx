import React, { useEffect, useState, useCallback, useRef } from 'react';
import { Card, Button, Space, Typography, Badge, Popconfirm, message } from 'antd';
import {
  PoweroffOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { serverAPI } from '../../../api';

const { Text } = Typography;

interface PowerTabProps {
  serverId: string;
}

type PowerState = 'on' | 'off' | 'unknown';

const PowerTab: React.FC<PowerTabProps> = ({ serverId }) => {
  const [powerStatus, setPowerStatus] = useState<PowerState>('unknown');
  const [loading, setLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const fetchStatus = useCallback(async () => {
    try {
      const { data: resp } = await serverAPI.powerStatus(serverId);
      if (resp.success && resp.data) {
        setPowerStatus((resp.data as { status: PowerState }).status);
      }
    } catch {
      // silently fail on polling
    }
  }, [serverId]);

  useEffect(() => {
    setLoading(true);
    fetchStatus().finally(() => setLoading(false));

    // Poll every 15 seconds
    timerRef.current = setInterval(fetchStatus, 15000);
    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
    };
  }, [fetchStatus]);

  const handlePowerAction = async (action: string) => {
    setActionLoading(action);
    try {
      const { data: resp } = await serverAPI.power(serverId, action);
      if (resp.success) {
        message.success(`Power ${action} command sent`);
        // Refresh status after a short delay
        setTimeout(fetchStatus, 3000);
      } else {
        message.error(resp.error || `Power ${action} failed`);
      }
    } catch {
      message.error(`Failed to send power ${action} command`);
    }
    setActionLoading(null);
  };

  const statusBadge = () => {
    if (loading) return <Badge status="processing" text="Checking..." />;
    switch (powerStatus) {
      case 'on':
        return <Badge status="success" text="Powered On" />;
      case 'off':
        return <Badge status="default" text="Powered Off" />;
      default:
        return <Badge status="warning" text="Unknown" />;
    }
  };

  return (
    <div>
      <Card title="Power Status" size="small" style={{ marginBottom: 16 }}>
        <Space>
          {statusBadge()}
          <Button size="small" onClick={() => fetchStatus()} loading={loading}>
            Refresh
          </Button>
        </Space>
      </Card>

      <Card title="Power Actions" size="small">
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <Text type="secondary">
            Use IPMI to control the physical power state of this server.
          </Text>
          <Space wrap>
            <Popconfirm
              title="Power On"
              description="Are you sure you want to power on this server?"
              onConfirm={() => handlePowerAction('on')}
              okText="Yes"
              cancelText="No"
            >
              <Button
                icon={<PlayCircleOutlined />}
                type="primary"
                loading={actionLoading === 'on'}
              >
                Power On
              </Button>
            </Popconfirm>

            <Popconfirm
              title="Power Off"
              description="This will immediately cut power. Are you sure?"
              onConfirm={() => handlePowerAction('off')}
              okText="Yes"
              cancelText="No"
            >
              <Button
                icon={<PoweroffOutlined />}
                danger
                loading={actionLoading === 'off'}
              >
                Power Off
              </Button>
            </Popconfirm>

            <Popconfirm
              title="Reset"
              description="This will hard-reset the server. Are you sure?"
              onConfirm={() => handlePowerAction('reset')}
              okText="Yes"
              cancelText="No"
            >
              <Button
                icon={<ReloadOutlined />}
                loading={actionLoading === 'reset'}
              >
                Reset
              </Button>
            </Popconfirm>

            <Popconfirm
              title="Power Cycle"
              description="This will power-cycle the server. Are you sure?"
              onConfirm={() => handlePowerAction('cycle')}
              okText="Yes"
              cancelText="No"
            >
              <Button
                icon={<SyncOutlined />}
                loading={actionLoading === 'cycle'}
              >
                Power Cycle
              </Button>
            </Popconfirm>
          </Space>
        </Space>
      </Card>
    </div>
  );
};

export default PowerTab;
