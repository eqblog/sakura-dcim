import React, { useEffect } from 'react';
import { Card, Form, Input, InputNumber, Select, Switch, Button, Row, Col, Divider, message } from 'antd';
import { SaveOutlined } from '@ant-design/icons';
import { ipPoolAPI } from '../../api';
import type { IPPool, Tenant } from '../../types';

interface Props {
  pool: IPPool;
  tenants: Tenant[];
  onSaved: () => void;
}

const SubnetConfigTab: React.FC<Props> = ({ pool, tenants, onSaved }) => {
  const [form] = Form.useForm();
  const switchAutomation = Form.useWatch('switch_automation', form);
  const vlanMode = Form.useWatch('vlan_mode', form);

  useEffect(() => {
    form.setFieldsValue({
      ...pool,
      nameservers: pool.nameservers?.join(', ') || '',
    });
  }, [pool, form]);

  const handleSave = async () => {
    try {
      const raw = await form.validateFields();
      const values = {
        ...raw,
        nameservers: raw.nameservers ? String(raw.nameservers).split(',').map((s: string) => s.trim()).filter(Boolean) : [],
      };
      const { data: resp } = await ipPoolAPI.update(pool.id, values);
      if (resp.success) {
        message.success('Subnet configuration saved');
        onSaved();
      } else message.error(resp.error);
    } catch { /* validation */ }
  };

  return (
    <Card>
      <Form form={form} layout="vertical">
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="network" label="Network (CIDR)" rules={[{ required: true }]}>
              <Input placeholder="10.0.0.0/24" />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="priority" label="Priority">
              <InputNumber style={{ width: '100%' }} placeholder="0" />
            </Form.Item>
          </Col>
        </Row>
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="gateway" label="Gateway" rules={[{ required: true }]}>
              <Input placeholder="10.0.0.1" />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="netmask" label="Netmask">
              <Input placeholder="255.255.255.0" />
            </Form.Item>
          </Col>
        </Row>
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="rdns_server" label="rDNS Server">
              <Input placeholder="ns1.example.com" />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="nameservers" label="Nameservers">
              <Input placeholder="8.8.8.8, 8.8.4.4 (comma-separated)" />
            </Form.Item>
          </Col>
        </Row>
        <Row gutter={16}>
          <Col span={8}>
            <Form.Item name="vrf" label="VRF">
              <Input placeholder="default" />
            </Form.Item>
          </Col>
          <Col span={8}>
            <Form.Item name="tenant_id" label="Tenant">
              <Select allowClear placeholder="Select tenant" options={tenants.map(t => ({ label: t.name, value: t.id }))} />
            </Form.Item>
          </Col>
          <Col span={8}>
            <Form.Item name="description" label="Description">
              <Input placeholder="Production subnet" />
            </Form.Item>
          </Col>
        </Row>

        <Divider orientation="left">Switch Automation</Divider>

        <Row gutter={16}>
          <Col span={8}>
            <Form.Item name="switch_automation" label="Enable Switch Automation" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Col>
          {switchAutomation && (
            <Col span={8}>
              <Form.Item name="vlan_mode" label="VLAN Mode">
                <Select options={[
                  { label: 'Access', value: 'access' },
                  { label: 'Trunk + Native', value: 'trunk_native' },
                  { label: 'Trunk', value: 'trunk' },
                ]} />
              </Form.Item>
            </Col>
          )}
        </Row>

        {switchAutomation && vlanMode === 'access' && (
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="vlan_id" label="Access VLAN ID">
                <InputNumber style={{ width: '100%' }} min={1} max={4094} />
              </Form.Item>
            </Col>
          </Row>
        )}

        {switchAutomation && vlanMode === 'trunk_native' && (
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="native_vlan_id" label="Native VLAN ID">
                <InputNumber style={{ width: '100%' }} min={1} max={4094} />
              </Form.Item>
            </Col>
            <Col span={16}>
              <Form.Item name="trunk_vlans" label="Trunk VLAN IDs">
                <Input placeholder="100,200,300-400 (comma-separated, ranges supported)" />
              </Form.Item>
            </Col>
          </Row>
        )}

        {switchAutomation && vlanMode === 'trunk' && (
          <Row gutter={16}>
            <Col span={16}>
              <Form.Item name="trunk_vlans" label="Trunk VLAN IDs">
                <Input placeholder="100,200,300-400 (comma-separated, ranges supported)" />
              </Form.Item>
            </Col>
          </Row>
        )}

        {switchAutomation && (
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="vlan_range_start" label="VLAN Range Start">
                <InputNumber style={{ width: '100%' }} min={0} max={4094} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="vlan_range_end" label="VLAN Range End">
                <InputNumber style={{ width: '100%' }} min={0} max={4094} />
              </Form.Item>
            </Col>
          </Row>
        )}

        <Form.Item name="notes" label="Notes">
          <Input.TextArea rows={4} placeholder="Additional notes for this subnet..." />
        </Form.Item>
        <Button type="primary" icon={<SaveOutlined />} onClick={handleSave}>Save Configuration</Button>
      </Form>
    </Card>
  );
};

export default SubnetConfigTab;
