import React from 'react';
import { Result } from 'antd';
import { DesktopOutlined } from '@ant-design/icons';

const KvmTab: React.FC = () => {
  return (
    <Result
      icon={<DesktopOutlined />}
      title="KVM Console"
      subTitle="KVM Console will be available in Phase 4."
    />
  );
};

export default KvmTab;
