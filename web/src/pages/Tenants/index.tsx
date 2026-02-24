import React from 'react';
import { Card, Typography, Empty } from 'antd';

const { Title } = Typography;

const TenantsPage: React.FC = () => (
  <div>
    <Title level={4}>Tenants</Title>
    <Card>
      <Empty description="Tenant management - Coming in Phase 8" />
    </Card>
  </div>
);

export default TenantsPage;
