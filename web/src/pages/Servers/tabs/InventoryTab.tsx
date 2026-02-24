import React from 'react';
import { Result } from 'antd';
import { HddOutlined } from '@ant-design/icons';

const InventoryTab: React.FC = () => {
  return (
    <Result
      icon={<HddOutlined />}
      title="Hardware Inventory"
      subTitle="Hardware inventory will be available in Phase 7."
    />
  );
};

export default InventoryTab;
