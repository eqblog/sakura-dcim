import React from 'react';
import { Card, Typography, Empty } from 'antd';

const { Title } = Typography;

const BandwidthPage: React.FC = () => (
  <div>
    <Title level={4}>Bandwidth</Title>
    <Card>
      <Empty description="Bandwidth monitoring - Coming in Phase 6" />
    </Card>
  </div>
);

export default BandwidthPage;
