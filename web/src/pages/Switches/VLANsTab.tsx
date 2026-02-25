import React from 'react';
import { Table, Tag, Card, Empty } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import type { VLANSummary } from '../../types';

interface Props {
  vlans: VLANSummary[];
}

const VLANsTab: React.FC<Props> = ({ vlans }) => {
  if (vlans.length === 0) {
    return <Card><Empty description="No VLANs configured on this switch" /></Card>;
  }

  const columns: ColumnsType<VLANSummary> = [
    {
      title: 'VLAN ID', dataIndex: 'vlan_id', key: 'vlan_id', width: 120,
      sorter: (a, b) => a.vlan_id - b.vlan_id,
      render: (v: number) => <Tag color="blue">{v}</Tag>,
    },
    {
      title: 'Port Count', dataIndex: 'port_count', key: 'port_count', width: 120,
      sorter: (a, b) => a.port_count - b.port_count,
    },
    {
      title: 'Member Ports', dataIndex: 'ports', key: 'ports',
      render: (ports: string[]) => (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
          {ports.map((p, i) => (
            <Tag key={i} color={p.includes('native') ? 'volcano' : undefined}>{p}</Tag>
          ))}
        </div>
      ),
    },
  ];

  return (
    <Card>
      <Table columns={columns} dataSource={vlans} rowKey="vlan_id" size="small" pagination={false} />
    </Card>
  );
};

export default VLANsTab;
