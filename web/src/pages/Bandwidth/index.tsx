import React, { useEffect, useState } from 'react';
import { Card, Typography, Table, Tag, Select, Space, Empty } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { switchAPI } from '../../api';
import type { Switch } from '../../types';

const { Title } = Typography;

const BandwidthPage: React.FC = () => {
  const [switches, setSwitches] = useState<Switch[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const fetch = async () => {
      setLoading(true);
      try {
        const { data: resp } = await switchAPI.list();
        if (resp.success) setSwitches(resp.data || []);
      } catch { /* */ }
      setLoading(false);
    };
    fetch();
  }, []);

  const columns: ColumnsType<Switch> = [
    { title: 'Switch', dataIndex: 'name', key: 'name', sorter: (a, b) => a.name.localeCompare(b.name) },
    { title: 'IP', dataIndex: 'ip', key: 'ip' },
    { title: 'Vendor', dataIndex: 'vendor', key: 'vendor', render: (v: string) => v ? <Tag>{v}</Tag> : '-' },
    { title: 'SNMP', dataIndex: 'snmp_version', key: 'snmp', render: (v: string) => <Tag>{v}</Tag> },
    { title: 'Model', dataIndex: 'model', key: 'model', ellipsis: true },
  ];

  return (
    <div>
      <Title level={4}>Bandwidth Overview</Title>
      <Card>
        {switches.length === 0 && !loading ? (
          <Empty description="No switches configured. Add switches and configure SNMP polling to see bandwidth data." />
        ) : (
          <Table columns={columns} dataSource={switches} rowKey="id" loading={loading} size="small" />
        )}
      </Card>
    </div>
  );
};

export default BandwidthPage;
