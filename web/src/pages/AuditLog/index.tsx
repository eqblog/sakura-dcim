import React, { useEffect, useState } from 'react';
import { Table, Card, Typography, Input, DatePicker, Space, Tag } from 'antd';
import { SearchOutlined } from '@ant-design/icons';
import { auditAPI } from '../../api';
import type { AuditLog, PaginatedResult } from '../../types';
import dayjs from 'dayjs';

const { Title } = Typography;
const { RangePicker } = DatePicker;

const AuditLogPage: React.FC = () => {
  const [data, setData] = useState<PaginatedResult<AuditLog> | null>(null);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [actionFilter, setActionFilter] = useState('');

  useEffect(() => {
    fetchLogs();
  }, [page]);

  const fetchLogs = async () => {
    setLoading(true);
    try {
      const params: Record<string, any> = { page, page_size: 20 };
      if (actionFilter) params.action = actionFilter;
      const { data: resp } = await auditAPI.list(params);
      if (resp.success) setData(resp.data || null);
    } catch { /* ignore */ }
    setLoading(false);
  };

  const columns = [
    {
      title: 'Time',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (t: string) => dayjs(t).format('YYYY-MM-DD HH:mm:ss'),
    },
    { title: 'Action', dataIndex: 'action', key: 'action' },
    { title: 'Resource', dataIndex: 'resource_type', key: 'resource_type' },
    { title: 'IP', dataIndex: 'ip_address', key: 'ip_address' },
    {
      title: 'Details',
      dataIndex: 'details',
      key: 'details',
      render: (d: any) => d ? <Tag>{JSON.stringify(d).slice(0, 60)}</Tag> : '-',
    },
  ];

  return (
    <div>
      <Title level={4}>Audit Log</Title>
      <Card>
        <Space style={{ marginBottom: 16 }}>
          <Input
            placeholder="Filter by action"
            prefix={<SearchOutlined />}
            value={actionFilter}
            onChange={(e) => setActionFilter(e.target.value)}
            onPressEnter={fetchLogs}
            style={{ width: 280 }}
          />
          <RangePicker />
        </Space>
        <Table
          dataSource={data?.items || []}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={{
            current: data?.page || 1,
            pageSize: data?.page_size || 20,
            total: data?.total || 0,
            onChange: setPage,
          }}
          size="small"
        />
      </Card>
    </div>
  );
};

export default AuditLogPage;
