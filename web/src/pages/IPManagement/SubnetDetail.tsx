import React from 'react';
import { Button, Tabs, Typography, Space, Tag, Breadcrumb } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import type { IPPool, IPAddress, Tenant } from '../../types';
import IPOverviewTab from './IPOverviewTab';
import SubnetConfigTab from './SubnetConfigTab';
import ChildSubnetsTab from './ChildSubnetsTab';

const { Title } = Typography;

interface Props {
  pool: IPPool;
  poolStack: IPPool[];
  childPools: IPPool[];
  addresses: IPAddress[];
  addrLoading: boolean;
  tenants: Tenant[];
  onBack: () => void;
  onBreadcrumbNav: (index: number) => void;
  onDrillDown: (pool: IPPool) => void;
  onPoolSaved: () => void;
  onRefreshAddresses: () => void;
  onDeleteChild: (id: string) => void;
}

const SubnetDetail: React.FC<Props> = ({
  pool, poolStack, childPools, addresses, addrLoading, tenants,
  onBack, onBreadcrumbNav, onDrillDown, onPoolSaved, onRefreshAddresses, onDeleteChild,
}) => {
  const breadcrumbItems = [
    { title: <a onClick={() => onBreadcrumbNav(-1)}>IP Pools</a>, key: 'root' },
    ...poolStack.map((p, i) => ({
      title: i < poolStack.length - 1
        ? <a onClick={() => onBreadcrumbNav(i)}>{p.network}</a>
        : p.network,
      key: p.id,
    })),
  ];

  const isSubnet = pool.pool_type === 'subnet';

  const tabItems = isSubnet
    ? [
        {
          key: 'subnets',
          label: `Subnets (${childPools.length})`,
          children: (
            <ChildSubnetsTab
              parentPool={pool}
              childPools={childPools}
              loading={addrLoading}
              tenants={tenants}
              onSelect={onDrillDown}
              onRefresh={onRefreshAddresses}
              onDelete={onDeleteChild}
            />
          ),
        },
        {
          key: 'config',
          label: 'Subnet Configuration',
          children: <SubnetConfigTab pool={pool} tenants={tenants} onSaved={onPoolSaved} />,
        },
      ]
    : [
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
      ];

  return (
    <div>
      <Breadcrumb items={breadcrumbItems} style={{ marginBottom: 12 }} />
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={onBack}>Back</Button>
        <Title level={4} style={{ margin: 0 }}>{pool.network}</Title>
        <Tag color="blue">{pool.gateway}</Tag>
        {pool.vrf && <Tag color="purple">VRF: {pool.vrf}</Tag>}
        <Tag color={isSubnet ? 'geekblue' : 'green'}>{isSubnet ? 'Subnet' : 'IP Pool'}</Tag>
      </Space>
      <Tabs defaultActiveKey={isSubnet ? 'subnets' : 'overview'} items={tabItems} />
    </div>
  );
};

export default SubnetDetail;
