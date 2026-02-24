import React, { useEffect, useState, useCallback, useRef } from 'react';
import { Card, Table, Button, Space, Tag, Typography, Spin } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { serverAPI } from '../../../api';

const { Text } = Typography;

interface SensorsTabProps {
  serverId: string;
}

interface SensorReading {
  name: string;
  value: string;
  status: string;
}

const statusColor = (status: string): string => {
  const s = status.toLowerCase().trim();
  if (s === 'ok' || s === 'nominal') return 'green';
  if (s === 'warning' || s === 'nc' || s === 'nr') return 'orange';
  if (s === 'critical' || s === 'cr') return 'red';
  return 'default';
};

const columns: ColumnsType<SensorReading> = [
  {
    title: 'Sensor',
    dataIndex: 'name',
    key: 'name',
    sorter: (a, b) => a.name.localeCompare(b.name),
  },
  {
    title: 'Value',
    dataIndex: 'value',
    key: 'value',
  },
  {
    title: 'Status',
    dataIndex: 'status',
    key: 'status',
    render: (status: string) => (
      <Tag color={statusColor(status)}>{status}</Tag>
    ),
    filters: [
      { text: 'OK', value: 'ok' },
      { text: 'Warning', value: 'warning' },
      { text: 'Critical', value: 'critical' },
    ],
    onFilter: (value, record) =>
      record.status.toLowerCase().includes(value as string),
  },
];

const SensorsTab: React.FC<SensorsTabProps> = ({ serverId }) => {
  const [sensors, setSensors] = useState<SensorReading[]>([]);
  const [loading, setLoading] = useState(false);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const fetchSensors = useCallback(async () => {
    setLoading(true);
    try {
      const { data: resp } = await serverAPI.sensors(serverId);
      if (resp.success && resp.data) {
        const data = resp.data as { sensors: SensorReading[] };
        setSensors(data.sensors || []);
        setLastUpdated(new Date());
      }
    } catch {
      // silently fail on polling
    }
    setLoading(false);
  }, [serverId]);

  useEffect(() => {
    fetchSensors();

    // Poll every 30 seconds
    timerRef.current = setInterval(fetchSensors, 30000);
    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
    };
  }, [fetchSensors]);

  return (
    <div>
      <Card
        title="IPMI Sensors"
        size="small"
        extra={
          <Space>
            {lastUpdated && (
              <Text type="secondary" style={{ fontSize: 12 }}>
                Updated: {lastUpdated.toLocaleTimeString()}
              </Text>
            )}
            <Button
              size="small"
              icon={<ReloadOutlined />}
              onClick={fetchSensors}
              loading={loading}
            >
              Refresh
            </Button>
          </Space>
        }
      >
        {loading && sensors.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 40 }}>
            <Spin />
            <div style={{ marginTop: 8 }}>
              <Text type="secondary">Querying IPMI sensors...</Text>
            </div>
          </div>
        ) : (
          <Table
            columns={columns}
            dataSource={sensors}
            rowKey="name"
            size="small"
            pagination={false}
            locale={{ emptyText: 'No sensor data available. The server may not have IPMI configured.' }}
          />
        )}
      </Card>
    </div>
  );
};

export default SensorsTab;
