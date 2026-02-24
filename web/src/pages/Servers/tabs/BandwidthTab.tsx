import React from 'react';
import { Result } from 'antd';
import { BarChartOutlined } from '@ant-design/icons';

const BandwidthTab: React.FC = () => {
  return (
    <Result
      icon={<BarChartOutlined />}
      title="Bandwidth"
      subTitle="Bandwidth graphs will be available in Phase 6."
    />
  );
};

export default BandwidthTab;
