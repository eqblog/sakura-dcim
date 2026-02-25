import React, { useState } from 'react';
import { Card, Typography, Table, Button, Space, Modal, Form, Input, Select, InputNumber, Tag, message, Popconfirm } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, ApiOutlined, RadarChartOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { switchAPI } from '../../api';
import type { Switch, Agent } from '../../types';

const { Title, Text } = Typography;

const vendorOptions = [
  { label: 'Cisco IOS', value: 'cisco_ios' },
  { label: 'Cisco NX-OS (Nexus)', value: 'cisco_nxos' },
  { label: 'Juniper JunOS', value: 'junos' },
  { label: 'Arista EOS', value: 'arista_eos' },
  { label: 'SONiC', value: 'sonic' },
  { label: 'Cumulus Linux', value: 'cumulus' },
];

interface Props {
  switches: Switch[];
  loading: boolean;
  agents: Agent[];
  onSelect: (sw: Switch) => void;
  onRefresh: () => void;
}

const SwitchList: React.FC<Props> = ({ switches, loading, agents, onSelect, onRefresh }) => {
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Switch | null>(null);
  const [form] = Form.useForm();
  const [testLoading, setTestLoading] = useState<string | null>(null);
  const [testResult, setTestResult] = useState<any>(null);
  const [testModalOpen, setTestModalOpen] = useState(false);

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({ ssh_port: 22, snmp_community: 'public', snmp_version: 'v2c' });
    setModalOpen(true);
  };

  const openEdit = (record: Switch) => {
    setEditing(record);
    form.setFieldsValue(record);
    setModalOpen(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (editing) {
        const { data: resp } = await switchAPI.update(editing.id, values);
        if (resp.success) message.success('Switch updated');
        else { message.error(resp.error); return; }
      } else {
        const { data: resp } = await switchAPI.create(values);
        if (resp.success) {
          message.success('Switch created');
          const newSwitch = resp.data as Switch;
          if (newSwitch?.id) switchAPI.syncPorts(newSwitch.id).catch(() => {});
        } else { message.error(resp.error); return; }
      }
      setModalOpen(false);
      onRefresh();
    } catch { /* validation */ }
  };

  const handleDelete = async (id: string) => {
    const { data: resp } = await switchAPI.delete(id);
    if (resp.success) { message.success('Deleted'); onRefresh(); }
    else message.error(resp.error);
  };

  const handleTest = async (sw: Switch) => {
    setTestLoading(sw.id);
    try {
      const { data: resp } = await switchAPI.testConnection(sw.id);
      if (resp.success) { setTestResult({ ...resp.data, switch_name: sw.name }); setTestModalOpen(true); }
      else message.error(resp.error || 'Test failed');
    } catch { message.error('Test connection failed'); }
    setTestLoading(null);
  };

  const handleSNMP = async (sw: Switch) => {
    setTestLoading(`snmp-${sw.id}`);
    try {
      const { data: resp } = await switchAPI.pollSNMP(sw.id);
      if (resp.success) message.success('SNMP poll completed');
      else message.error(resp.error || 'SNMP poll failed');
    } catch { message.error('SNMP poll failed'); }
    setTestLoading(null);
  };

  const columns: ColumnsType<Switch> = [
    { title: 'Name', dataIndex: 'name', key: 'name', sorter: (a, b) => a.name.localeCompare(b.name) },
    { title: 'IP', dataIndex: 'ip', key: 'ip' },
    { title: 'Vendor', dataIndex: 'vendor', key: 'vendor', render: (v: string) => {
      const opt = vendorOptions.find(o => o.value === v);
      return v ? <Tag>{opt?.label || v}</Tag> : '-';
    }},
    { title: 'Model', dataIndex: 'model', key: 'model', ellipsis: true },
    { title: 'SNMP', dataIndex: 'snmp_version', key: 'snmp', render: (v: string) => <Tag>{v}</Tag> },
    { title: 'Actions', key: 'actions', width: 320, render: (_, record) => (
      <Space wrap>
        <Button size="small" type="link" onClick={() => onSelect(record)}>Manage</Button>
        <Button size="small" icon={<ApiOutlined />} loading={testLoading === record.id} onClick={() => handleTest(record)}>Test</Button>
        <Button size="small" icon={<RadarChartOutlined />} loading={testLoading === `snmp-${record.id}`} onClick={() => handleSNMP(record)}>SNMP</Button>
        <Button size="small" icon={<EditOutlined />} onClick={(e) => { e.stopPropagation(); openEdit(record); }} />
        <Popconfirm title="Delete this switch?" onConfirm={() => handleDelete(record.id)}>
          <Button size="small" danger icon={<DeleteOutlined />} />
        </Popconfirm>
      </Space>
    )},
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>Switches</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>Add Switch</Button>
      </div>
      <Card>
        <Table columns={columns} dataSource={switches} rowKey="id" loading={loading} size="small"
          onRow={(record) => ({ onClick: () => onSelect(record), style: { cursor: 'pointer' } })} />
      </Card>

      <Modal title={editing ? 'Edit Switch' : 'Create Switch'} open={modalOpen} onOk={handleSubmit} onCancel={() => setModalOpen(false)} width={700} okText={editing ? 'Update' : 'Create'}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true }]}><Input placeholder="Core Switch DC-1" /></Form.Item>
          <Space style={{ width: '100%' }} size="middle">
            <Form.Item name="ip" label="IP Address" rules={[{ required: true }]} style={{ flex: 1 }}><Input placeholder="10.0.0.1" /></Form.Item>
            <Form.Item name="vendor" label="Vendor" style={{ flex: 1 }}><Select options={vendorOptions} placeholder="Select vendor" allowClear /></Form.Item>
            <Form.Item name="model" label="Model" style={{ flex: 1 }}><Input placeholder="Nexus 9336C-FX2" /></Form.Item>
          </Space>
          <Space style={{ width: '100%' }} size="middle">
            <Form.Item name="snmp_community" label="SNMP Community" style={{ flex: 1 }}><Input placeholder="public" /></Form.Item>
            <Form.Item name="snmp_version" label="SNMP Version" style={{ flex: 1 }}><Select options={[{ label: 'v2c', value: 'v2c' }, { label: 'v3', value: 'v3' }]} /></Form.Item>
          </Space>
          <Space style={{ width: '100%' }} size="middle">
            <Form.Item name="ssh_user" label="SSH User" style={{ flex: 1 }}><Input placeholder="admin" /></Form.Item>
            <Form.Item name="ssh_pass" label="SSH Password" style={{ flex: 1 }}><Input.Password placeholder="SSH password" /></Form.Item>
            <Form.Item name="ssh_port" label="SSH Port" style={{ flex: 0, minWidth: 100 }}><InputNumber min={1} max={65535} /></Form.Item>
          </Space>
          <Form.Item name="agent_id" label="Agent" rules={[{ required: true }]}>
            <Select placeholder="Select agent" showSearch optionFilterProp="label" options={agents.map(a => ({ label: `${a.name} (${a.location})`, value: a.id }))} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title={`Connection Test — ${testResult?.switch_name}`} open={testModalOpen} onCancel={() => setTestModalOpen(false)} footer={<Button onClick={() => setTestModalOpen(false)}>Close</Button>} width={700}>
        {testResult && (
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            <Card size="small" title="SSH">
              <Tag color={testResult.ssh_ok ? 'green' : 'red'}>{testResult.ssh_ok ? 'Connected' : 'Failed'}</Tag>
              <Text type="secondary" style={{ marginLeft: 8 }}>{testResult.ssh_message}</Text>
            </Card>
            <Card size="small" title="SNMP">
              <Tag color={testResult.snmp_ok ? 'green' : 'red'}>{testResult.snmp_ok ? 'Connected' : 'Failed'}</Tag>
              <Text type="secondary" style={{ marginLeft: 8 }}>{testResult.snmp_message}</Text>
            </Card>
          </Space>
        )}
      </Modal>
    </div>
  );
};

export default SwitchList;
