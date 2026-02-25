import React, { useEffect } from 'react';
import { Modal, Form, Input, Select, InputNumber, Space, message } from 'antd';
import { switchAPI } from '../../api';
import type { SwitchPort } from '../../types';

interface Props {
  open: boolean;
  port: SwitchPort | null;
  switchId: string;
  onClose: () => void;
  onSuccess: () => void;
}

const PortEditModal: React.FC<Props> = ({ open, port, switchId, onClose, onSuccess }) => {
  const [form] = Form.useForm();
  const portMode = Form.useWatch('port_mode', form);

  useEffect(() => {
    if (open) {
      if (port) {
        form.setFieldsValue(port);
      } else {
        form.resetFields();
        form.setFieldsValue({ admin_status: 'up', speed_mbps: 1000, vlan_id: 0, port_mode: 'access', native_vlan_id: 0, trunk_vlans: '' });
      }
    }
  }, [open, port, form]);

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (port) {
        const { data: resp } = await switchAPI.updatePort(switchId, port.id, values);
        if (resp.success) { message.success('Port updated'); onSuccess(); }
        else message.error(resp.error);
      } else {
        const { data: resp } = await switchAPI.createPort(switchId, values);
        if (resp.success) { message.success('Port created'); onSuccess(); }
        else message.error(resp.error);
      }
    } catch { /* validation */ }
  };

  return (
    <Modal title={port ? 'Edit Port' : 'Create Port'} open={open} onOk={handleSubmit} onCancel={onClose} width={600} okText={port ? 'Update' : 'Create'}>
      <Form form={form} layout="vertical">
        <Space style={{ width: '100%' }} size="middle">
          <Form.Item name="port_index" label="Port Index" rules={[{ required: true }]} style={{ flex: 0, minWidth: 120 }}>
            <InputNumber min={1} />
          </Form.Item>
          <Form.Item name="port_name" label="Port Name" rules={[{ required: true }]} style={{ flex: 1 }}>
            <Input placeholder="Ethernet1/1" />
          </Form.Item>
        </Space>

        <Space style={{ width: '100%' }} size="middle">
          <Form.Item name="speed_mbps" label="Speed" style={{ flex: 1 }}>
            <Select options={[
              { label: '100M', value: 100 }, { label: '1G', value: 1000 },
              { label: '10G', value: 10000 }, { label: '25G', value: 25000 },
              { label: '40G', value: 40000 }, { label: '100G', value: 100000 },
            ]} />
          </Form.Item>
          <Form.Item name="port_mode" label="Port Mode" style={{ flex: 1 }}>
            <Select options={[
              { label: 'Access', value: 'access' },
              { label: 'Trunk', value: 'trunk' },
              { label: 'Trunk + Native', value: 'trunk_native' },
            ]} />
          </Form.Item>
          <Form.Item name="admin_status" label="Admin Status" style={{ flex: 1 }}>
            <Select options={[{ label: 'Up', value: 'up' }, { label: 'Down', value: 'down' }]} />
          </Form.Item>
        </Space>

        {(!portMode || portMode === 'access') && (
          <Form.Item name="vlan_id" label="Access VLAN ID">
            <InputNumber min={0} max={4094} style={{ width: '100%' }} />
          </Form.Item>
        )}

        {(portMode === 'trunk' || portMode === 'trunk_native') && (
          <Space style={{ width: '100%' }} size="middle">
            {portMode === 'trunk_native' && (
              <Form.Item name="native_vlan_id" label="Native VLAN ID" style={{ flex: 1 }}>
                <InputNumber min={0} max={4094} style={{ width: '100%' }} />
              </Form.Item>
            )}
            <Form.Item name="trunk_vlans" label="Trunk VLANs (comma-separated)" style={{ flex: 2 }}>
              <Input placeholder="100, 200, 300" />
            </Form.Item>
          </Space>
        )}

        <Form.Item name="description" label="Description">
          <Input placeholder="Server-01 uplink" />
        </Form.Item>
        <Form.Item name="server_id" label="Server ID (Optional)">
          <Input placeholder="Server UUID to link" allowClear />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default PortEditModal;
