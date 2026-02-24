import React from 'react';
import { Card, Typography, Empty } from 'antd';

const { Title } = Typography;

const UsersPage: React.FC = () => (
  <div>
    <Title level={4}>Users & Roles</Title>
    <Card>
      <Empty description="User management - Coming in Phase 2" />
    </Card>
  </div>
);

export default UsersPage;
