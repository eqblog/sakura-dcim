import React, { useEffect, useState } from 'react';
import { Card, Button, Spin, Empty, Descriptions, Table, Tag, Space, message, Typography, Row, Col, Alert } from 'antd';
import { ReloadOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { serverAPI } from '../../../api';
import type { InventoryResult } from '../../../types';

const { Text } = Typography;

interface Props {
  serverId: string;
}

const CPUInfo: React.FC<{ details: any }> = ({ details }) => {
  if (!details) return <Text type="secondary">No CPU data</Text>;
  return (
    <Descriptions column={2} size="small" bordered>
      <Descriptions.Item label="Model">{details['Model name'] || details['model'] || '-'}</Descriptions.Item>
      <Descriptions.Item label="Architecture">{details['Architecture'] || '-'}</Descriptions.Item>
      <Descriptions.Item label="CPUs">{details['CPU(s)'] || details['cores'] || '-'}</Descriptions.Item>
      <Descriptions.Item label="Threads/Core">{details['Thread(s) per core'] || '-'}</Descriptions.Item>
      <Descriptions.Item label="Cores/Socket">{details['Core(s) per socket'] || '-'}</Descriptions.Item>
      <Descriptions.Item label="Sockets">{details['Socket(s)'] || details['sockets'] || '-'}</Descriptions.Item>
      <Descriptions.Item label="MHz">{details['CPU MHz'] || details['CPU max MHz'] || '-'}</Descriptions.Item>
      <Descriptions.Item label="Vendor">{details['Vendor ID'] || details['vendor'] || '-'}</Descriptions.Item>
    </Descriptions>
  );
};

const DiskInfo: React.FC<{ details: any }> = ({ details }) => {
  const blockdevices = details?.blockdevices || [];
  const disks = blockdevices.filter((d: any) => d.type === 'disk');
  if (disks.length === 0) return <Text type="secondary">No disk data</Text>;

  const columns = [
    { title: 'Name', dataIndex: 'name', key: 'name' },
    { title: 'Model', dataIndex: 'model', key: 'model', render: (v: string) => v || '-' },
    { title: 'Size', dataIndex: 'size', key: 'size' },
    { title: 'Type', key: 'dtype', render: (_: any, r: any) => {
      if (r.rota === '0' || r.rota === false) return <Tag color="blue">SSD</Tag>;
      if (r.tran === 'nvme') return <Tag color="purple">NVMe</Tag>;
      return <Tag>HDD</Tag>;
    }},
    { title: 'Transport', dataIndex: 'tran', key: 'tran', render: (v: string) => v ? <Tag>{v.toUpperCase()}</Tag> : '-' },
    { title: 'Serial', dataIndex: 'serial', key: 'serial', render: (v: string) => v || '-', ellipsis: true },
  ];

  return <Table columns={columns} dataSource={disks} rowKey="name" size="small" pagination={false} />;
};

const NetworkInfo: React.FC<{ details: any }> = ({ details }) => {
  if (!Array.isArray(details) || details.length === 0) return <Text type="secondary">No network data</Text>;

  const columns = [
    { title: 'Interface', dataIndex: 'ifname', key: 'ifname' },
    { title: 'State', dataIndex: 'operstate', key: 'state', render: (v: string) =>
      v === 'UP' ? <Tag color="green">UP</Tag> : <Tag color="red">{v}</Tag>
    },
    { title: 'MAC', dataIndex: 'address', key: 'mac' },
    { title: 'IP Addresses', key: 'ips', render: (_: any, r: any) => {
      const addrs = r.addr_info || [];
      return addrs.map((a: any, i: number) => (
        <Tag key={i} color={a.family === 'inet' ? 'blue' : 'cyan'}>{a.local}/{a.prefixlen}</Tag>
      ));
    }},
    { title: 'MTU', dataIndex: 'mtu', key: 'mtu' },
  ];

  return <Table columns={columns} dataSource={details.filter((d: any) => d.ifname !== 'lo')} rowKey="ifname" size="small" pagination={false} />;
};

const MemoryInfo: React.FC<{ details: any }> = ({ details }) => {
  if (!details) return <Text type="secondary">No memory data</Text>;
  if (typeof details === 'string') {
    return <pre style={{ maxHeight: 300, overflow: 'auto', fontSize: 12 }}>{details}</pre>;
  }
  return <pre style={{ maxHeight: 300, overflow: 'auto', fontSize: 12 }}>{JSON.stringify(details, null, 2)}</pre>;
};

const SystemInfo: React.FC<{ details: any }> = ({ details }) => {
  if (!details) return <Text type="secondary">No system data</Text>;
  if (typeof details === 'string') {
    return <pre style={{ maxHeight: 200, overflow: 'auto', fontSize: 12 }}>{details}</pre>;
  }
  return <pre style={{ maxHeight: 200, overflow: 'auto', fontSize: 12 }}>{JSON.stringify(details, null, 2)}</pre>;
};

const componentRenderers: Record<string, React.FC<{ details: any }>> = {
  cpu: CPUInfo,
  disks: DiskInfo,
  network: NetworkInfo,
  memory_raw: MemoryInfo,
  system_raw: SystemInfo,
};

const componentTitles: Record<string, string> = {
  cpu: 'CPU',
  disks: 'Storage',
  network: 'Network Interfaces',
  memory_raw: 'Memory (dmidecode)',
  system_raw: 'System Info (dmidecode)',
};

const InventoryTab: React.FC<Props> = ({ serverId }) => {
  const [inventory, setInventory] = useState<InventoryResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [scanning, setScanning] = useState(false);
  const [scanInitiated, setScanInitiated] = useState(false);

  const fetchInventory = async () => {
    setLoading(true);
    try {
      const { data: resp } = await serverAPI.inventory(serverId);
      if (resp.success && resp.data) {
        setInventory(resp.data);
        if (scanInitiated && resp.data.components?.length > 0) {
          setScanInitiated(false);
        }
      }
    } catch { /* */ }
    setLoading(false);
  };

  const triggerScan = async () => {
    setScanning(true);
    try {
      const { data: resp } = await serverAPI.inventoryScan(serverId);
      if (resp.success) {
        message.success('PXE inventory scan initiated. The server will reboot into scanning mode and report hardware automatically.');
        setScanInitiated(true);
      } else {
        message.error(resp.error || 'Scan failed');
      }
    } catch {
      message.error('Failed to trigger PXE inventory scan');
    }
    setScanning(false);
  };

  useEffect(() => { fetchInventory(); }, [serverId]);

  // Auto-refresh while scan is in progress
  useEffect(() => {
    if (!scanInitiated) return;
    const timer = setInterval(fetchInventory, 15000);
    return () => clearInterval(timer);
  }, [scanInitiated, serverId]);

  if (loading && !inventory) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>;
  }

  if (!inventory || !inventory.components || inventory.components.length === 0) {
    return (
      <Empty description="No inventory data collected yet.">
        {scanInitiated ? (
          <Alert
            type="info"
            showIcon
            title="PXE scan in progress"
            description="The server is rebooting into inventory scanning mode. Hardware data will appear here automatically when the scan completes."
            style={{ marginBottom: 16, maxWidth: 500, margin: '0 auto 16px' }}
          />
        ) : (
          <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
            Click below to PXE boot this server into a hardware scanning system.
            The server will reboot, scan its hardware, and report back automatically.
          </Text>
        )}
        <Button type="primary" icon={<ThunderboltOutlined />} onClick={triggerScan} loading={scanning} disabled={scanInitiated}>
          {scanInitiated ? 'Scanning...' : 'PXE Inventory Scan'}
        </Button>
      </Empty>
    );
  }

  const componentMap: Record<string, any> = {};
  for (const c of inventory.components) {
    componentMap[c.component] = c.details;
  }

  const renderOrder = ['cpu', 'disks', 'network', 'memory_raw', 'system_raw'];
  const knownComponents = renderOrder.filter(k => componentMap[k]);
  const unknownComponents = Object.keys(componentMap).filter(k => !renderOrder.includes(k));

  return (
    <div>
      {scanInitiated && (
        <Alert
          type="info"
          showIcon
          title="PXE scan in progress — waiting for hardware report from the server..."
          style={{ marginBottom: 16 }}
          closable
          onClose={() => setScanInitiated(false)}
        />
      )}
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ReloadOutlined />} onClick={fetchInventory}>Refresh</Button>
        <Button type="primary" icon={<ThunderboltOutlined />} onClick={triggerScan} loading={scanning} disabled={scanInitiated}>
          {scanInitiated ? 'Scanning...' : 'PXE Re-scan'}
        </Button>
        {inventory.collected_at && (
          <Text type="secondary">Last scan: {new Date(inventory.collected_at).toLocaleString()}</Text>
        )}
      </Space>

      <Row gutter={[16, 16]}>
        {knownComponents.map(key => {
          const Renderer = componentRenderers[key];
          return (
            <Col span={24} key={key}>
              <Card size="small" title={componentTitles[key] || key}>
                <Renderer details={componentMap[key]} />
              </Card>
            </Col>
          );
        })}
        {unknownComponents.map(key => (
          <Col span={24} key={key}>
            <Card size="small" title={key}>
              <pre style={{ maxHeight: 200, overflow: 'auto', fontSize: 12 }}>
                {JSON.stringify(componentMap[key], null, 2)}
              </pre>
            </Card>
          </Col>
        ))}
      </Row>
    </div>
  );
};

export default InventoryTab;
