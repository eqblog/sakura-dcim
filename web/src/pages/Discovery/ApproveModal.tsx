import React, { useEffect } from 'react';
import { Modal, Form, Input, Select, message } from 'antd';
import { discoveryAPI, agentAPI } from '../../api';
import type { DiscoveredServer, Agent } from '../../types';

interface Props {
  open: boolean;
  server: DiscoveredServer | null;
  agents: Agent[];
  onClose: () => void;
  onSuccess: () => void;
}

const ApproveModal: React.FC<Props> = ({ open, server, agents, onClose, onSuccess }) => {
  const [form] = Form.useForm();

  useEffect(() => {
    if (server && open) {
      form.setFieldsValue({
        hostname: '',
        label: `${server.system_vendor} ${server.system_product}`.trim(),
        agent_id: server.agent_id,
        ipmi_ip: server.bmc_ip || '',
        ipmi_user: '',
        ipmi_pass: '',
        tags: [],
        notes: `Auto-discovered: ${server.cpu_model}, ${server.cpu_cores} cores, ${Math.round(server.ram_mb / 1024)}GB RAM, ${server.disk_count} disks (${server.disk_total_gb}GB)`,
      });
    }
  }, [server, open, form]);

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      const { data: resp } = await discoveryAPI.approve(server!.id, values);
      if (resp.success) {
        message.success('Server approved and created');
        onSuccess();
      } else {
        message.error(resp.error || 'Failed to approve');
      }
    } catch {
      // validation error
    }
  };

  return (
    <Modal
      title="Approve Discovered Server"
      open={open}
      onOk={handleOk}
      onCancel={onClose}
      width={600}
      destroyOnClose
    >
      {server && (
        <div style={{ marginBottom: 16, padding: 12, background: '#f5f5f5', borderRadius: 6, fontSize: 13 }}>
          <strong>Hardware:</strong> {server.system_vendor} {server.system_product} |
          CPU: {server.cpu_model} ({server.cpu_cores} cores, {server.cpu_sockets} sockets) |
          RAM: {Math.round(server.ram_mb / 1024)} GB |
          Disks: {server.disk_count} ({server.disk_total_gb} GB) |
          NICs: {server.nic_count} | MAC: {server.mac_address}
        </div>
      )}
      <Form form={form} layout="vertical">
        <Form.Item name="hostname" label="Hostname" rules={[{ required: true, message: 'Hostname is required' }]}>
          <Input placeholder="e.g. svr-rack01-u12" />
        </Form.Item>
        <Form.Item name="label" label="Label">
          <Input />
        </Form.Item>
        <Form.Item name="agent_id" label="Agent">
          <Select allowClear>
            {agents.map(a => (
              <Select.Option key={a.id} value={a.id}>{a.name} ({a.location})</Select.Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="ipmi_ip" label="IPMI IP">
          <Input placeholder="e.g. 10.99.1.101" />
        </Form.Item>
        <Form.Item name="ipmi_user" label="IPMI User">
          <Input />
        </Form.Item>
        <Form.Item name="ipmi_pass" label="IPMI Password">
          <Input.Password />
        </Form.Item>
        <Form.Item name="tags" label="Tags">
          <Select mode="tags" placeholder="Add tags" />
        </Form.Item>
        <Form.Item name="notes" label="Notes">
          <Input.TextArea rows={3} />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default ApproveModal;
