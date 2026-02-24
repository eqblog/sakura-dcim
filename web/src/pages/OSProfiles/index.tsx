import React from 'react';
import { Card, Typography, Empty } from 'antd';

const { Title } = Typography;

const OSProfilesPage: React.FC = () => (
  <div>
    <Title level={4}>OS Profiles</Title>
    <Card>
      <Empty description="OS Profile management - Coming in Phase 5" />
    </Card>
  </div>
);

export default OSProfilesPage;
