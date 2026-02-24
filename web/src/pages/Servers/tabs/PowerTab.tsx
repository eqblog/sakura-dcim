import React from 'react';
import { Card, Button, Space, Typography, Alert } from 'antd';
import {
  PoweroffOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  SyncOutlined,
} from '@ant-design/icons';

const { Text } = Typography;

const PowerTab: React.FC = () => {
  return (
    <div>
      <Alert
        message="Power controls will be fully wired in Phase 3."
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
      />
      <Card title="Power Actions" size="small">
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <Text type="secondary">
            Use IPMI to control the physical power state of this server.
          </Text>
          <Space wrap>
            <Button
              icon={<PlayCircleOutlined />}
              type="primary"
              disabled
            >
              Power On
            </Button>
            <Button
              icon={<PoweroffOutlined />}
              danger
              disabled
            >
              Power Off
            </Button>
            <Button
              icon={<ReloadOutlined />}
              disabled
            >
              Reset
            </Button>
            <Button
              icon={<SyncOutlined />}
              disabled
            >
              Power Cycle
            </Button>
          </Space>
        </Space>
      </Card>
    </div>
  );
};

export default PowerTab;
