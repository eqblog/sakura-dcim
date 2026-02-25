import React, { useState } from 'react';
import { Typography, Button, Space, Tag, Tabs } from 'antd';
import { ArrowLeftOutlined, ReloadOutlined, CodeOutlined } from '@ant-design/icons';
import type { Switch, SwitchPort, VLANSummary, SwitchBandwidthMap } from '../../types';
import InterfacesTab from './InterfacesTab';
import VLANsTab from './VLANsTab';
import TemplatesTab from './TemplatesTab';

const { Title, Text } = Typography;

const vendorLabels: Record<string, string> = {
  cisco_ios: 'Cisco IOS', cisco_nxos: 'Cisco NX-OS', junos: 'JunOS',
  arista_eos: 'Arista EOS', sonic: 'SONiC', cumulus: 'Cumulus Linux',
};

interface Props {
  sw: Switch;
  ports: SwitchPort[];
  portsLoading: boolean;
  vlans: VLANSummary[];
  bandwidth: SwitchBandwidthMap;
  onBack: () => void;
  onRefresh: () => void;
}

const SwitchDetail: React.FC<Props> = ({ sw, ports, portsLoading, vlans, bandwidth, onBack, onRefresh }) => {
  const [activeTab, setActiveTab] = useState('interfaces');

  return (
    <div>
      <Space style={{ marginBottom: 16 }} align="center">
        <Button icon={<ArrowLeftOutlined />} onClick={onBack}>Back</Button>
        <Title level={4} style={{ margin: 0 }}>{sw.name}</Title>
        <Text type="secondary">{sw.ip}</Text>
        {sw.vendor && <Tag>{vendorLabels[sw.vendor] || sw.vendor}</Tag>}
        {sw.model && <Tag color="blue">{sw.model}</Tag>}
      </Space>

      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        tabBarExtraContent={
          <Button icon={<ReloadOutlined />} loading={portsLoading} onClick={onRefresh}>Refresh</Button>
        }
        items={[
          {
            key: 'interfaces',
            label: `Interfaces (${ports.length})`,
            children: <InterfacesTab sw={sw} ports={ports} loading={portsLoading} bandwidth={bandwidth} onRefresh={onRefresh} />,
          },
          {
            key: 'vlans',
            label: `VLANs (${vlans.length})`,
            children: <VLANsTab vlans={vlans} />,
          },
          {
            key: 'templates',
            label: <Space><CodeOutlined />Templates</Space>,
            children: <TemplatesTab />,
          },
        ]}
      />
    </div>
  );
};

export default SwitchDetail;
