import React, { useEffect, useState } from 'react';
import { Card, Typography, Table, Button, Space, Modal, Form, Input, Select, InputNumber, Tag, message, Popconfirm, Tabs, Switch as AntSwitch } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, ThunderboltOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { switchAPI } from '../../api';
import type { Switch, SwitchPort } from '../../types';

const { Title } = Typography;

const vendorOptions = [
  { label: 'Cisco IOS', value: 'cisco_ios' },
  { label: 'Juniper JunOS', value: 'junos' },
  { label: 'Arista EOS', value: 'arista_eos' },
  { label: 'SONiC', value: 'sonic' },
  { label: 'Cumulus', value: 'cumulus' },
];

const SwitchesPage: React.FC = () => {
  const [switches, setSwitches] = useState<Switch[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Switch | null>(null);
  const [form] = Form.useForm();

  // Port state
  const [selectedSwitch, setSelectedSwitch] = useState<Switch | null>(null);
  const [ports, setPorts] = useState<SwitchPort[]>([]);
  const [portsLoading, setPortsLoading] = useState(false);
  const [portModalOpen, setPortModalOpen] = useState(false);
  const [editingPort, setEditingPort] = useState<SwitchPort | null>(null);
  const [portForm] = Form.useForm();

  const fetchSwitches = async () => {
    setLoading(true);
    try {
      const { data: resp } = await switchAPI.list();
      if (resp.success) setSwitches(resp.data || []);
    } catch { /* */ }
    setLoading(false);
  };

  useEffect(() => { fetchSwitches(); }, []);

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
        if (resp.success) message.success('Switch created');
        else { message.error(resp.error); return; }
      }
      setModalOpen(false);
      fetchSwitches();
    } catch { /* validation */ }
  };

  const handleDelete = async (id: string) => {
    const { data: resp } = await switchAPI.delete(id);
    if (resp.success) { message.success('Deleted'); fetchSwitches(); }
    else message.error(resp.error);
  };

  // Port methods
  const fetchPorts = async (sw: Switch) => {
    setSelectedSwitch(sw);
    setPortsLoading(true);
    try {
      const { data: resp } = await switchAPI.listPorts(sw.id);
      if (resp.success) setPorts(resp.data || []);
    } catch { /* */ }
    setPortsLoading(false);
  };

  const openCreatePort = () => {
    setEditingPort(null);
    portForm.resetFields();
    portForm.setFieldsValue({ admin_status: 'up', speed_mbps: 1000, vlan_id: 0 });
    setPortModalOpen(true);
  };

  const openEditPort = (record: SwitchPort) => {
    setEditingPort(record);
    portForm.setFieldsValue(record);
    setPortModalOpen(true);
  };

  const handlePortSubmit = async () => {
    if (!selectedSwitch) return;
    try {
      const values = await portForm.validateFields();
      if (editingPort) {
        const { data: resp } = await switchAPI.updatePort(selectedSwitch.id, editingPort.id, values);
        if (resp.success) message.success('Port updated');
        else { message.error(resp.error); return; }
      } else {
        const { data: resp } = await switchAPI.createPort(selectedSwitch.id, values);
        if (resp.success) message.success('Port created');
        else { message.error(resp.error); return; }
      }
      setPortModalOpen(false);
      fetchPorts(selectedSwitch);
    } catch { /* validation */ }
  };

  const handleDeletePort = async (portId: string) => {
    if (!selectedSwitch) return;
    const { data: resp } = await switchAPI.deletePort(selectedSwitch.id, portId);
    if (resp.success) { message.success('Deleted'); fetchPorts(selectedSwitch); }
    else message.error(resp.error);
  };

  const handleProvision = async (portId: string) => {
    if (!selectedSwitch) return;
    try {
      const { data: resp } = await switchAPI.provisionPort(selectedSwitch.id, portId);
      if (resp.success) message.success('Port provisioned');
      else message.error(resp.error);
    } catch { message.error('Provision failed'); }
  };

  const switchColumns: ColumnsType<Switch> = [
    { title: 'Name', dataIndex: 'name', key: 'name', sorter: (a, b) => a.name.localeCompare(b.name) },
    { title: 'IP', dataIndex: 'ip', key: 'ip' },
    { title: 'Vendor', dataIndex: 'vendor', key: 'vendor', render: (v: string) => v ? <Tag>{v}</Tag> : '-' },
    { title: 'Model', dataIndex: 'model', key: 'model', ellipsis: true },
    { title: 'SNMP', dataIndex: 'snmp_version', key: 'snmp', render: (v: string) => <Tag>{v}</Tag> },
    {
      title: 'Actions', key: 'actions', width: 180,
      render: (_, record) => (
        <Space>
          <Button size="small" onClick={() => fetchPorts(record)}>Ports</Button>
          <Button size="small" icon={<EditOutlined />} onClick={() => openEdit(record)} />
          <Popconfirm title="Delete this switch?" onConfirm={() => handleDelete(record.id)}>
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const portColumns: ColumnsType<SwitchPort> = [
    { title: 'Index', dataIndex: 'port_index', key: 'port_index', width: 70, sorter: (a, b) => a.port_index - b.port_index },
    { title: 'Port Name', dataIndex: 'port_name', key: 'port_name' },
    { title: 'Speed', dataIndex: 'speed_mbps', key: 'speed', render: (v: number) => v >= 1000 ? `${v / 1000}G` : `${v}M` },
    { title: 'VLAN', dataIndex: 'vlan_id', key: 'vlan', render: (v: number) => v > 0 ? <Tag color="blue">{v}</Tag> : '-' },
    { title: 'Admin', dataIndex: 'admin_status', key: 'admin', render: (v: string) => v === 'up' ? <Tag color="green">Up</Tag> : <Tag color="red">Down</Tag> },
    { title: 'Oper', dataIndex: 'oper_status', key: 'oper', render: (v: string) => v === 'up' ? <Tag color="green">Up</Tag> : v === 'down' ? <Tag color="red">Down</Tag> : <Tag>Unknown</Tag> },
    { title: 'Description', dataIndex: 'description', key: 'desc', ellipsis: true },
    {
      title: 'Actions', key: 'actions', width: 180,
      render: (_, record) => (
        <Space>
          <Popconfirm title="Provision this port?" onConfirm={() => handleProvision(record.id)}>
            <Button size="small" icon={<ThunderboltOutlined />} title="Provision" />
          </Popconfirm>
          <Button size="small" icon={<EditOutlined />} onClick={() => openEditPort(record)} />
          <Popconfirm title="Delete this port?" onConfirm={() => handleDeletePort(record.id)}>
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4}>Switches</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>Add Switch</Button>
      </div>

      <Card>
        <Table columns={switchColumns} dataSource={switches} rowKey="id" loading={loading} size="small" />
      </Card>

      {selectedSwitch && (
        <Card title={`Ports — ${selectedSwitch.name} (${selectedSwitch.ip})`} style={{ marginTop: 16 }}
          extra={
            <Space>
              <Button icon={<ReloadOutlined />} onClick={() => fetchPorts(selectedSwitch)}>Refresh</Button>
              <Button type="primary" icon={<PlusOutlined />} onClick={openCreatePort}>Add Port</Button>
            </Space>
          }>
          <Table columns={portColumns} dataSource={ports} rowKey="id" loading={portsLoading} size="small" />
        </Card>
      )}

      {/* Switch Modal */}
      <Modal title={editing ? 'Edit Switch' : 'Create Switch'} open={modalOpen} onOk={handleSubmit} onCancel={() => setModalOpen(false)} width={700} okText={editing ? 'Update' : 'Create'}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true }]}>
            <Input placeholder="Core Switch DC-1" />
          </Form.Item>
          <Space style={{ width: '100%' }} size="middle">
            <Form.Item name="ip" label="IP Address" rules={[{ required: true }]} style={{ flex: 1 }}>
              <Input placeholder="10.0.0.1" />
            </Form.Item>
            <Form.Item name="vendor" label="Vendor" style={{ flex: 1 }}>
              <Select options={vendorOptions} placeholder="Select vendor" allowClear />
            </Form.Item>
            <Form.Item name="model" label="Model" style={{ flex: 1 }}>
              <Input placeholder="Catalyst 9300" />
            </Form.Item>
          </Space>
          <Space style={{ width: '100%' }} size="middle">
            <Form.Item name="snmp_community" label="SNMP Community" style={{ flex: 1 }}>
              <Input placeholder="public" />
            </Form.Item>
            <Form.Item name="snmp_version" label="SNMP Version" style={{ flex: 1 }}>
              <Select options={[{ label: 'v2c', value: 'v2c' }, { label: 'v3', value: 'v3' }]} />
            </Form.Item>
          </Space>
          <Space style={{ width: '100%' }} size="middle">
            <Form.Item name="ssh_user" label="SSH User" style={{ flex: 1 }}>
              <Input placeholder="admin" />
            </Form.Item>
            <Form.Item name="ssh_pass" label="SSH Password" style={{ flex: 1 }}>
              <Input.Password placeholder="SSH password" />
            </Form.Item>
            <Form.Item name="ssh_port" label="SSH Port" style={{ flex: 0, minWidth: 100 }}>
              <InputNumber min={1} max={65535} />
            </Form.Item>
          </Space>
          <Form.Item name="agent_id" label="Agent ID" rules={[{ required: true }]}>
            <Input placeholder="Agent UUID" />
          </Form.Item>
        </Form>
      </Modal>

      {/* Port Modal */}
      <Modal title={editingPort ? 'Edit Port' : 'Create Port'} open={portModalOpen} onOk={handlePortSubmit} onCancel={() => setPortModalOpen(false)} width={600} okText={editingPort ? 'Update' : 'Create'}>
        <Form form={portForm} layout="vertical">
          <Space style={{ width: '100%' }} size="middle">
            <Form.Item name="port_index" label="Port Index" rules={[{ required: true }]} style={{ flex: 0, minWidth: 120 }}>
              <InputNumber min={1} />
            </Form.Item>
            <Form.Item name="port_name" label="Port Name" rules={[{ required: true }]} style={{ flex: 1 }}>
              <Input placeholder="GigabitEthernet0/1" />
            </Form.Item>
          </Space>
          <Space style={{ width: '100%' }} size="middle">
            <Form.Item name="speed_mbps" label="Speed (Mbps)" style={{ flex: 1 }}>
              <Select options={[
                { label: '100M', value: 100 }, { label: '1G', value: 1000 },
                { label: '10G', value: 10000 }, { label: '25G', value: 25000 },
                { label: '40G', value: 40000 }, { label: '100G', value: 100000 },
              ]} />
            </Form.Item>
            <Form.Item name="vlan_id" label="VLAN ID" style={{ flex: 1 }}>
              <InputNumber min={0} max={4094} />
            </Form.Item>
            <Form.Item name="admin_status" label="Admin Status" style={{ flex: 1 }}>
              <Select options={[{ label: 'Up', value: 'up' }, { label: 'Down', value: 'down' }]} />
            </Form.Item>
          </Space>
          <Form.Item name="description" label="Description">
            <Input placeholder="Server-01 uplink" />
          </Form.Item>
          <Form.Item name="server_id" label="Server ID (Optional)">
            <Input placeholder="Server UUID to link" allowClear />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default SwitchesPage;
