import React from 'react';
import { Result } from 'antd';
import { CloudDownloadOutlined } from '@ant-design/icons';

const ReinstallTab: React.FC = () => {
  return (
    <Result
      icon={<CloudDownloadOutlined />}
      title="OS Reinstall"
      subTitle="OS Reinstall will be available in Phase 5."
    />
  );
};

export default ReinstallTab;
