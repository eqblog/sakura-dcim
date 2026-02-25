import React from 'react';
import { Drawer, Descriptions, Tag, Typography } from 'antd';
import type { DiscoveredServer } from '../../types';

interface Props {
  open: boolean;
  server: DiscoveredServer | null;
  onClose: () => void;
}

const DetailDrawer: React.FC<Props> = ({ open, server, onClose }) => {
  if (!server) return null;

  const statusColor = {
    pending: 'blue',
    approved: 'green',
    rejected: 'red',
  }[server.status] || 'default';

  return (
    <Drawer
      title="Discovered Server Details"
      open={open}
      onClose={onClose}
      width={640}
    >
      <Descriptions column={2} bordered size="small">
        <Descriptions.Item label="Status" span={2}>
          <Tag color={statusColor}>{server.status.toUpperCase()}</Tag>
        </Descriptions.Item>
        <Descriptions.Item label="MAC Address">{server.mac_address}</Descriptions.Item>
        <Descriptions.Item label="IP Address">{server.ip_address}</Descriptions.Item>
        <Descriptions.Item label="System Vendor">{server.system_vendor}</Descriptions.Item>
        <Descriptions.Item label="System Product">{server.system_product}</Descriptions.Item>
        <Descriptions.Item label="Serial Number" span={2}>{server.system_serial}</Descriptions.Item>
        <Descriptions.Item label="CPU Model" span={2}>{server.cpu_model}</Descriptions.Item>
        <Descriptions.Item label="CPU Cores">{server.cpu_cores}</Descriptions.Item>
        <Descriptions.Item label="CPU Sockets">{server.cpu_sockets}</Descriptions.Item>
        <Descriptions.Item label="RAM">{Math.round(server.ram_mb / 1024)} GB ({server.ram_mb} MB)</Descriptions.Item>
        <Descriptions.Item label="NICs">{server.nic_count}</Descriptions.Item>
        <Descriptions.Item label="Disks">{server.disk_count} ({server.disk_total_gb} GB total)</Descriptions.Item>
        <Descriptions.Item label="BMC IP">{server.bmc_ip || '-'}</Descriptions.Item>
        <Descriptions.Item label="Discovered At" span={2}>
          {new Date(server.discovered_at).toLocaleString()}
        </Descriptions.Item>
      </Descriptions>

      <Typography.Title level={5} style={{ marginTop: 24 }}>Raw Inventory</Typography.Title>
      <pre style={{
        background: '#f5f5f5',
        padding: 12,
        borderRadius: 6,
        maxHeight: 400,
        overflow: 'auto',
        fontSize: 12,
      }}>
        {JSON.stringify(server.raw_inventory, null, 2)}
      </pre>
    </Drawer>
  );
};

export default DetailDrawer;
