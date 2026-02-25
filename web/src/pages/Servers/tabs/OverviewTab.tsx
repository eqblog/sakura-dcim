import React, { useEffect, useState } from 'react';
import { Descriptions, Tag, Space, Typography, Button, Modal, Form, Input, Select, message } from 'antd';
import { EditOutlined } from '@ant-design/icons';
import { serverAPI, agentAPI } from '../../../api';
import type { Server, ServerStatus, Agent } from '../../../types';

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
  onUpdated?: (server: Server) => void;
}

const OverviewTab: React.FC<OverviewTabProps> = ({ server, onUpdated }) => {
  const [editOpen, setEditOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [form] = Form.useForm();
  const [agents, setAgents] = useState<Agent[]>([]);

  useEffect(() => {
    agentAPI.list({ page: 1, page_size: 200 }).then(({ data: resp }) => {
      if (resp.success) setAgents(resp.data?.items || []);
    });
  }, []);

  const openEdit = () => {
    form.setFieldsValue({
      hostname: server.hostname,
      label: server.label,
      agent_id: server.agent_id || undefined,
      primary_ip: server.primary_ip,
      ipmi_ip: server.ipmi_ip,
      bmc_type: server.bmc_type || 'generic',
      tags: server.tags || [],
      notes: server.notes,
    });
    setEditOpen(true);
  };

  const handleSave = async (values: any) => {
    setSaving(true);
    try {
      const { data: resp } = await serverAPI.update(server.id, values);
      if (resp.success) {
        message.success('Server updated');
        setEditOpen(false);
        onUpdated?.(resp.data!);
      } else {
        message.error(resp.error || 'Update failed');
      }
    } catch {
      message.error('Failed to update server');
    }
    setSaving(false);
  };

  return (
    <>
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 12 }}>
        <Button icon={<EditOutlined />} onClick={openEdit}>Edit</Button>
      </div>

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
        <Descriptions.Item label="BMC Type">
          {server.bmc_type ? {
            generic: 'Generic IPMI',
            dell_idrac: 'Dell iDRAC',
            hp_ilo: 'HPE iLO',
            supermicro: 'Supermicro IPMI',
            lenovo_xcc: 'Lenovo XClarity',
            huawei_ibmc: 'Huawei iBMC',
          }[server.bmc_type] || server.bmc_type : '-'}
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

      <Modal
        title="Edit Server"
        open={editOpen}
        onCancel={() => setEditOpen(false)}
        footer={null}
        width={600}
      >
        <Form form={form} layout="vertical" onFinish={handleSave}>
          <Form.Item name="hostname" label="Hostname" rules={[{ required: true, message: 'Hostname is required' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="label" label="Label">
            <Input />
          </Form.Item>
          <Form.Item name="agent_id" label="Agent">
            <Select
              allowClear
              placeholder="Select agent"
              options={agents.map((a) => ({ label: `${a.name} (${a.location || 'No location'})`, value: a.id }))}
            />
          </Form.Item>
          <Form.Item name="primary_ip" label="Primary IP">
            <Input placeholder="192.168.1.100" />
          </Form.Item>
          <Form.Item name="ipmi_ip" label="IPMI IP">
            <Input placeholder="10.0.0.100" />
          </Form.Item>
          <Form.Item name="ipmi_user" label="IPMI User">
            <Input placeholder="Leave empty to keep unchanged" />
          </Form.Item>
          <Form.Item name="ipmi_pass" label="IPMI Password">
            <Input.Password placeholder="Leave empty to keep unchanged" />
          </Form.Item>
          <Form.Item name="bmc_type" label="BMC Type">
            <Select
              options={[
                { value: 'generic', label: 'Generic IPMI' },
                { value: 'dell_idrac', label: 'Dell iDRAC' },
                { value: 'hp_ilo', label: 'HPE iLO' },
                { value: 'supermicro', label: 'Supermicro IPMI' },
                { value: 'lenovo_xcc', label: 'Lenovo XClarity' },
                { value: 'huawei_ibmc', label: 'Huawei iBMC' },
              ]}
            />
          </Form.Item>
          <Form.Item name="tags" label="Tags">
            <Select mode="tags" placeholder="Press Enter to add tag" />
          </Form.Item>
          <Form.Item name="notes" label="Notes">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setEditOpen(false)}>Cancel</Button>
              <Button type="primary" htmlType="submit" loading={saving}>Save</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default OverviewTab;
