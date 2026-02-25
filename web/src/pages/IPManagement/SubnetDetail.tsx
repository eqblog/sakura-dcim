import React from 'react';
import { Button, Tabs, Typography, Space, Tag } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import type { IPPool, IPAddress, Tenant } from '../../types';
import IPOverviewTab from './IPOverviewTab';
import SubnetConfigTab from './SubnetConfigTab';

const { Title } = Typography;

interface Props {
  pool: IPPool;
  addresses: IPAddress[];
  addrLoading: boolean;
  tenants: Tenant[];
  onBack: () => void;
  onPoolSaved: () => void;
  onRefreshAddresses: () => void;
}

const SubnetDetail: React.FC<Props> = ({ pool, addresses, addrLoading, tenants, onBack, onPoolSaved, onRefreshAddresses }) => {
  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={onBack}>Back to Pools</Button>
        <Title level={4} style={{ margin: 0 }}>{pool.network}</Title>
        <Tag color="blue">{pool.gateway}</Tag>
        {pool.vrf && <Tag color="purple">VRF: {pool.vrf}</Tag>}
      </Space>
      <Tabs defaultActiveKey="overview" items={[
        {
          key: 'overview',
          label: 'IP Overview',
          children: <IPOverviewTab pool={pool} addresses={addresses} loading={addrLoading} onRefresh={onRefreshAddresses} onPoolChanged={onPoolSaved} />,
        },
        {
          key: 'config',
          label: 'Subnet Configuration',
          children: <SubnetConfigTab pool={pool} tenants={tenants} onSaved={onPoolSaved} />,
        },
      ]} />
    </div>
  );
};

export default SubnetDetail;
