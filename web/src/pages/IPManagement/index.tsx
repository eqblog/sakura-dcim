import React from 'react';
import { Card, Typography, Empty } from 'antd';

const { Title } = Typography;

const IPManagementPage: React.FC = () => (
  <div>
    <Title level={4}>IP Pools</Title>
    <Card>
      <Empty description="IP management - Coming in Phase 7" />
    </Card>
  </div>
);

export default IPManagementPage;
