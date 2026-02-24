import React, { useEffect, useState, useCallback } from 'react';
import { Table, Card, Typography, Input, DatePicker, Space, Tag, Select, Button, Tooltip, Modal, Descriptions } from 'antd';
import { SearchOutlined, ReloadOutlined, EyeOutlined, ClearOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { auditAPI } from '../../api';
import type { AuditLog, PaginatedResult } from '../../types';
import dayjs from 'dayjs';

const { Title, Text } = Typography;
const { RangePicker } = DatePicker;

const actionColors: Record<string, string> = {
  POST: 'green',
  PUT: 'blue',
  DELETE: 'red',
  PATCH: 'orange',
};

const resourceOptions = [
  { label: 'All Resources', value: '' },
  { label: 'Server', value: 'server' },
  { label: 'Agent', value: 'agent' },
  { label: 'User', value: 'user' },
  { label: 'Role', value: 'role' },
  { label: 'Tenant', value: 'tenant' },
  { label: 'OS Profile', value: 'os_profile' },
  { label: 'Disk Layout', value: 'disk_layout' },
  { label: 'Script', value: 'script' },
  { label: 'Switch', value: 'switch' },
  { label: 'IP Pool', value: 'ip_pool' },
];

const AuditLogPage: React.FC = () => {
  const [data, setData] = useState<PaginatedResult<AuditLog> | null>(null);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [actionFilter, setActionFilter] = useState('');
  const [resourceFilter, setResourceFilter] = useState('');
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs | null, dayjs.Dayjs | null] | null>(null);
  const [detailRecord, setDetailRecord] = useState<AuditLog | null>(null);

  const fetchLogs = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, any> = { page, page_size: 25 };
      if (actionFilter) params.action = actionFilter;
      if (resourceFilter) params.resource_type = resourceFilter;
      if (dateRange?.[0]) params.start_time = dateRange[0].startOf('day').toISOString();
      if (dateRange?.[1]) params.end_time = dateRange[1].endOf('day').toISOString();
      const { data: resp } = await auditAPI.list(params);
      if (resp.success) setData(resp.data || null);
    } catch { /* ignore */ }
    setLoading(false);
  }, [page, actionFilter, resourceFilter, dateRange]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  const clearFilters = () => {
    setActionFilter('');
    setResourceFilter('');
    setDateRange(null);
    setPage(1);
  };

  const columns: ColumnsType<AuditLog> = [
    {
      title: 'Time',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 170,
      render: (t: string) => (
        <Tooltip title={dayjs(t).format('YYYY-MM-DD HH:mm:ss.SSS')}>
          {dayjs(t).format('MM-DD HH:mm:ss')}
        </Tooltip>
      ),
    },
    {
      title: 'Action',
      dataIndex: 'action',
      key: 'action',
      width: 260,
      render: (action: string) => {
        const method = action.split(' ')[0];
        return (
          <Space size={4}>
            <Tag color={actionColors[method] || 'default'}>{method}</Tag>
            <Text style={{ fontSize: 12 }}>{action.replace(method + ' ', '')}</Text>
          </Space>
        );
      },
    },
    {
      title: 'Resource',
      key: 'resource',
      width: 180,
      render: (_, record) => {
        if (!record.resource_type) return <Text type="secondary">-</Text>;
        return (
          <Space size={4}>
            <Tag>{record.resource_type}</Tag>
            {record.resource_id && (
              <Text copyable={{ text: record.resource_id }} style={{ fontSize: 11 }}>
                {record.resource_id.slice(0, 8)}
              </Text>
            )}
          </Space>
        );
      },
    },
    {
      title: 'User',
      dataIndex: 'user_id',
      key: 'user_id',
      width: 100,
      render: (uid: string) => uid ? (
        <Text style={{ fontSize: 11 }}>{uid.slice(0, 8)}...</Text>
      ) : <Text type="secondary">system</Text>,
    },
    {
      title: 'IP',
      dataIndex: 'ip_address',
      key: 'ip_address',
      width: 130,
      render: (ip: string) => <Text code style={{ fontSize: 11 }}>{ip}</Text>,
    },
    {
      title: 'Status',
      key: 'status',
      width: 70,
      render: (_, record) => {
        const status = (record.details as any)?.status;
        if (!status) return '-';
        const color = status >= 400 ? 'red' : status >= 300 ? 'orange' : 'green';
        return <Tag color={color}>{status}</Tag>;
      },
    },
    {
      title: '',
      key: 'actions',
      width: 50,
      render: (_, record) => (
        <Button
          type="text"
          size="small"
          icon={<EyeOutlined />}
          onClick={() => setDetailRecord(record)}
        />
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>Audit Log</Title>
        <Button icon={<ReloadOutlined />} onClick={fetchLogs}>Refresh</Button>
      </div>

      <Card>
        <Space wrap style={{ marginBottom: 16 }}>
          <Input
            placeholder="Filter by action..."
            prefix={<SearchOutlined />}
            value={actionFilter}
            onChange={(e) => { setActionFilter(e.target.value); setPage(1); }}
            onPressEnter={() => fetchLogs()}
            style={{ width: 220 }}
            allowClear
          />
          <Select
            placeholder="Resource type"
            value={resourceFilter}
            onChange={(v) => { setResourceFilter(v); setPage(1); }}
            options={resourceOptions}
            style={{ width: 160 }}
          />
          <RangePicker
            value={dateRange}
            onChange={(dates) => { setDateRange(dates); setPage(1); }}
          />
          <Button icon={<ClearOutlined />} onClick={clearFilters}>Clear</Button>
        </Space>

        <Table
          dataSource={data?.items || []}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={{
            current: data?.page || 1,
            pageSize: data?.page_size || 25,
            total: data?.total || 0,
            showSizeChanger: false,
            showTotal: (total) => `${total} entries`,
            onChange: (p) => setPage(p),
          }}
          size="small"
          scroll={{ x: 960 }}
        />
      </Card>

      <Modal
        title="Audit Log Detail"
        open={!!detailRecord}
        onCancel={() => setDetailRecord(null)}
        footer={null}
        width={640}
      >
        {detailRecord && (
          <Descriptions column={1} size="small" bordered>
            <Descriptions.Item label="ID">{detailRecord.id}</Descriptions.Item>
            <Descriptions.Item label="Time">
              {dayjs(detailRecord.created_at).format('YYYY-MM-DD HH:mm:ss.SSS')}
            </Descriptions.Item>
            <Descriptions.Item label="Action">{detailRecord.action}</Descriptions.Item>
            <Descriptions.Item label="Resource Type">{detailRecord.resource_type || '-'}</Descriptions.Item>
            <Descriptions.Item label="Resource ID">{detailRecord.resource_id || '-'}</Descriptions.Item>
            <Descriptions.Item label="User ID">{detailRecord.user_id || 'system'}</Descriptions.Item>
            <Descriptions.Item label="Tenant ID">{detailRecord.tenant_id || '-'}</Descriptions.Item>
            <Descriptions.Item label="IP Address">{detailRecord.ip_address}</Descriptions.Item>
            <Descriptions.Item label="User Agent">
              <Text style={{ fontSize: 11, wordBreak: 'break-all' }}>{detailRecord.user_agent}</Text>
            </Descriptions.Item>
            <Descriptions.Item label="Details">
              <pre style={{ maxHeight: 300, overflow: 'auto', fontSize: 11, margin: 0 }}>
                {JSON.stringify(detailRecord.details, null, 2)}
              </pre>
            </Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </div>
  );
};

export default AuditLogPage;
