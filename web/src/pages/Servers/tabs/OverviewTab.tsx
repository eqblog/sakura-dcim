import React from 'react';
import { Descriptions, Tag, Space, Typography } from 'antd';
import type { Server, ServerStatus } from '../../../types';

const { Text } = Typography;

const statusColors: Record<ServerStatus, string> = {
  active: 'green',
  provisioning: 'blue',
  reinstalling: 'orange',
  offline: 'default',
  error: 'red',
};

interface OverviewTabProps {
  server: Server;
}

const OverviewTab: React.FC<OverviewTabProps> = ({ server }) => {
  return (
    <Descriptions bordered column={{ xs: 1, sm: 2 }} size="middle">
      <Descriptions.Item label="Hostname">{server.hostname || '-'}</Descriptions.Item>
      <Descriptions.Item label="Label">{server.label || '-'}</Descriptions.Item>
      <Descriptions.Item label="Primary IP">
        <Text copyable={!!server.primary_ip}>{server.primary_ip || '-'}</Text>
      </Descriptions.Item>
      <Descriptions.Item label="IPMI IP">
        <Text copyable={!!server.ipmi_ip}>{server.ipmi_ip || '-'}</Text>
      </Descriptions.Item>
      <Descriptions.Item label="Status">
        <Tag color={statusColors[server.status]}>{server.status.toUpperCase()}</Tag>
      </Descriptions.Item>
      <Descriptions.Item label="CPU">
        {server.cpu_model
          ? `${server.cpu_model} (${server.cpu_cores} cores)`
          : '-'}
      </Descriptions.Item>
      <Descriptions.Item label="RAM">
        {server.ram_mb ? `${Math.round(server.ram_mb / 1024)} GB` : '-'}
      </Descriptions.Item>
      <Descriptions.Item label="Agent">
        {server.agent ? server.agent.name : server.agent_id || '-'}
      </Descriptions.Item>
      <Descriptions.Item label="Tags" span={2}>
        {server.tags?.length ? (
          <Space size={4} wrap>
            {server.tags.map((tag) => (
              <Tag key={tag} color="blue">{tag}</Tag>
            ))}
          </Space>
        ) : (
          '-'
        )}
      </Descriptions.Item>
      <Descriptions.Item label="Notes" span={2}>
        {server.notes || '-'}
      </Descriptions.Item>
      <Descriptions.Item label="Created">
        {server.created_at ? new Date(server.created_at).toLocaleString() : '-'}
      </Descriptions.Item>
      <Descriptions.Item label="Updated">
        {server.updated_at ? new Date(server.updated_at).toLocaleString() : '-'}
      </Descriptions.Item>
    </Descriptions>
  );
};

export default OverviewTab;
