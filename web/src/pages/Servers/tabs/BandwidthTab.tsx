import React, { useEffect, useState } from 'react';
import { Card, Table, Tag, Select, Space, Empty, Statistic, Row, Col } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { bandwidthAPI } from '../../../api';
import type { BandwidthSummary } from '../../../types';

interface Props {
  serverId: string;
}

const formatBps = (bps: number): string => {
  if (bps >= 1e9) return `${(bps / 1e9).toFixed(2)} Gbps`;
  if (bps >= 1e6) return `${(bps / 1e6).toFixed(2)} Mbps`;
  if (bps >= 1e3) return `${(bps / 1e3).toFixed(2)} Kbps`;
  return `${bps.toFixed(0)} bps`;
};

const BandwidthTab: React.FC<Props> = ({ serverId }) => {
  const [summaries, setSummaries] = useState<BandwidthSummary[]>([]);
  const [loading, setLoading] = useState(false);
  const [period, setPeriod] = useState('hourly');

  const fetchBandwidth = async () => {
    setLoading(true);
    try {
      const { data: resp } = await bandwidthAPI.getServerBandwidth(serverId, period);
      if (resp.success) setSummaries(resp.data || []);
    } catch { /* */ }
    setLoading(false);
  };

  useEffect(() => { fetchBandwidth(); }, [serverId, period]);

  if (!loading && summaries.length === 0) {
    return <Empty description="No bandwidth data available. Assign switch ports to this server and configure SNMP polling." />;
  }

  const columns: ColumnsType<BandwidthSummary> = [
    { title: 'Port', dataIndex: 'port_name', key: 'port_name' },
    { title: 'Speed', dataIndex: 'speed_mbps', key: 'speed', render: (v: number) => v >= 1000 ? `${v / 1000}G` : `${v}M` },
    { title: 'In 95th', key: 'in_95th', render: (_, r) => <Tag color="blue">{formatBps(r.in_95th_bps)}</Tag> },
    { title: 'Out 95th', key: 'out_95th', render: (_, r) => <Tag color="green">{formatBps(r.out_95th_bps)}</Tag> },
    { title: 'In Avg', key: 'in_avg', render: (_, r) => formatBps(r.in_avg_bps) },
    { title: 'Out Avg', key: 'out_avg', render: (_, r) => formatBps(r.out_avg_bps) },
    { title: 'In Max', key: 'in_max', render: (_, r) => formatBps(r.in_max_bps) },
    { title: 'Out Max', key: 'out_max', render: (_, r) => formatBps(r.out_max_bps) },
  ];

  // Aggregate stats
  const totalIn95th = summaries.reduce((sum, s) => sum + s.in_95th_bps, 0);
  const totalOut95th = summaries.reduce((sum, s) => sum + s.out_95th_bps, 0);

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <span>Period:</span>
        <Select value={period} onChange={setPeriod} style={{ width: 120 }} options={[
          { label: 'Hourly', value: 'hourly' },
          { label: 'Daily', value: 'daily' },
          { label: 'Monthly', value: 'monthly' },
        ]} />
      </Space>

      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}><Card size="small"><Statistic title="In 95th (Total)" value={formatBps(totalIn95th)} /></Card></Col>
        <Col span={6}><Card size="small"><Statistic title="Out 95th (Total)" value={formatBps(totalOut95th)} /></Card></Col>
        <Col span={6}><Card size="small"><Statistic title="Ports" value={summaries.length} /></Card></Col>
      </Row>

      <Table columns={columns} dataSource={summaries} rowKey="port_id" loading={loading} size="small" pagination={false} />
    </div>
  );
};

export default BandwidthTab;
