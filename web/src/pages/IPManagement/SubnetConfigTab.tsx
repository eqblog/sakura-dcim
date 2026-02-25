import React, { useEffect } from 'react';
import { Card, Form, Input, InputNumber, Select, Switch, Button, Row, Col, message } from 'antd';
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
            <Form.Item name="vlan_id" label="VLAN ID">
              <InputNumber style={{ width: '100%' }} min={0} max={4094} placeholder="0" />
            </Form.Item>
          </Col>
          <Col span={4}>
            <Form.Item name="vlan_range_start" label="VLAN Range Start">
              <InputNumber style={{ width: '100%' }} min={0} max={4094} />
            </Form.Item>
          </Col>
          <Col span={4}>
            <Form.Item name="vlan_range_end" label="VLAN Range End">
              <InputNumber style={{ width: '100%' }} min={0} max={4094} />
            </Form.Item>
          </Col>
        </Row>
        <Row gutter={16}>
          <Col span={8}>
            <Form.Item name="switch_automation" label="Switch Automation" valuePropName="checked">
              <Switch />
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
        <Form.Item name="notes" label="Notes">
          <Input.TextArea rows={4} placeholder="Additional notes for this subnet..." />
        </Form.Item>
        <Button type="primary" icon={<SaveOutlined />} onClick={handleSave}>Save Configuration</Button>
      </Form>
    </Card>
  );
};

export default SubnetConfigTab;
