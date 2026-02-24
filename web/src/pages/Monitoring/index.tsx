import React from 'react';
import { Card, Typography, Empty } from 'antd';

const { Title } = Typography;

const MonitoringPage: React.FC = () => (
  <div>
    <Title level={4}>Monitoring</Title>
    <Card>
      <Empty description="IPMI sensor monitoring - Coming in Phase 3" />
    </Card>
  </div>
);

export default MonitoringPage;
