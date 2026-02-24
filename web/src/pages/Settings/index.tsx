import React from 'react';
import { Card, Typography, Empty } from 'antd';

const { Title } = Typography;

const SettingsPage: React.FC = () => (
  <div>
    <Title level={4}>Settings</Title>
    <Card>
      <Empty description="System settings - Coming in Phase 8" />
    </Card>
  </div>
);

export default SettingsPage;
