import React, { useEffect, useState } from 'react';
import { Card, Typography, Table, Button, Space, Modal, Form, Input, Select, InputNumber, Tag, message, Popconfirm, Tabs, Collapse } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, ThunderboltOutlined, ReloadOutlined, CodeOutlined, ApiOutlined, RadarChartOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { switchAPI, agentAPI } from '../../api';
import type { Switch, SwitchPort, Agent } from '../../types';

const { Title, Text } = Typography;

const vendorOptions = [
  { label: 'Cisco IOS', value: 'cisco_ios' },
  { label: 'Cisco NX-OS (Nexus)', value: 'cisco_nxos' },
  { label: 'Juniper JunOS', value: 'junos' },
  { label: 'Arista EOS', value: 'arista_eos' },
  { label: 'SONiC', value: 'sonic' },
  { label: 'Cumulus Linux', value: 'cumulus' },
];

interface CommandTemplate {
  operation: string;
  description: string;
  template: string;
}

interface VendorTemplates {
  vendor: string;
  label: string;
  templates: CommandTemplate[];
}

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

  // Agents
  const [agents, setAgents] = useState<Agent[]>([]);

  // Test / SNMP state
  const [testResult, setTestResult] = useState<any>(null);
  const [testModalOpen, setTestModalOpen] = useState(false);
  const [testLoading, setTestLoading] = useState<string | null>(null);

  // Command templates state
  const [templates, setTemplates] = useState<VendorTemplates[]>([]);
  const [templatesLoading, setTemplatesLoading] = useState(false);

  const fetchSwitches = async () => {
    setLoading(true);
    try {
      const { data: resp } = await switchAPI.list();
      if (resp.success) setSwitches(resp.data || []);
    } catch { /* */ }
    setLoading(false);
  };

  const handleTestConnection = async (sw: Switch) => {
    setTestLoading(sw.id);
    try {
      const { data: resp } = await switchAPI.testConnection(sw.id);
      if (resp.success) {
        setTestResult({ type: 'test', switch_name: sw.name, ...resp.data });
        setTestModalOpen(true);
      } else {
        message.error(resp.error || 'Test failed');
      }
    } catch { message.error('Test connection failed'); }
    setTestLoading(null);
  };

  const handleSNMPPoll = async (sw: Switch) => {
    setTestLoading(`snmp-${sw.id}`);
    try {
      const { data: resp } = await switchAPI.pollSNMP(sw.id);
      if (resp.success) {
        setTestResult({ type: 'snmp', switch_name: sw.name, ...resp.data });
        setTestModalOpen(true);
        // SNMP poll also saves ports to DB — refresh port table if viewing this switch
        if (selectedSwitch?.id === sw.id || !selectedSwitch) {
          setSelectedSwitch(sw);
          switchAPI.listPorts(sw.id).then(({ data: portResp }) => {
            if (portResp.success) setPorts(portResp.data || []);
          });
        }
      } else {
        message.error(resp.error || 'SNMP poll failed');
      }
    } catch { message.error('SNMP poll failed'); }
    setTestLoading(null);
  };

  const fetchTemplates = async () => {
    setTemplatesLoading(true);
    try {
      const { data: resp } = await switchAPI.getCommandTemplates();
      if (resp.success) setTemplates(resp.data as VendorTemplates[] || []);
    } catch { /* */ }
    setTemplatesLoading(false);
  };

  useEffect(() => {
    fetchSwitches();
    agentAPI.list({ page: 1, page_size: 200 }).then(({ data: resp }) => {
      if (resp.success) setAgents(resp.data?.items || []);
    });
  }, []);

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
          // Auto-sync ports from SNMP for the new switch
          const newSwitch = resp.data as Switch;
          if (newSwitch?.id) {
            switchAPI.syncPorts(newSwitch.id).catch(() => { /* best effort */ });
          }
        } else { message.error(resp.error); return; }
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
      if (resp.success) {
        const portData = resp.data || [];
        setPorts(portData);
        // Auto-sync from SNMP if no ports in DB yet
        if (portData.length === 0) {
          syncPorts(sw);
          return;
        }
      }
    } catch { /* */ }
    setPortsLoading(false);
  };

  const syncPorts = async (sw: Switch) => {
    setSelectedSwitch(sw);
    setPortsLoading(true);
    try {
      const { data: resp } = await switchAPI.syncPorts(sw.id);
      if (resp.success) {
        setPorts(resp.data || []);
        message.success('Ports synced from SNMP');
      } else {
        message.error(resp.error || 'SNMP sync failed');
        // Fall back to loading from DB
        const { data: fallback } = await switchAPI.listPorts(sw.id);
        if (fallback.success) setPorts(fallback.data || []);
      }
    } catch {
      message.error('SNMP sync failed');
    }
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
    {
      title: 'Vendor', dataIndex: 'vendor', key: 'vendor',
      render: (v: string) => {
        const opt = vendorOptions.find(o => o.value === v);
        return v ? <Tag>{opt?.label || v}</Tag> : '-';
      },
    },
    { title: 'Model', dataIndex: 'model', key: 'model', ellipsis: true },
    { title: 'SNMP', dataIndex: 'snmp_version', key: 'snmp', render: (v: string) => <Tag>{v}</Tag> },
    {
      title: 'Actions', key: 'actions', width: 320,
      render: (_, record) => (
        <Space wrap>
          <Button size="small" onClick={() => fetchPorts(record)}>Ports</Button>
          <Button size="small" icon={<ApiOutlined />} loading={testLoading === record.id} onClick={() => handleTestConnection(record)}>Test</Button>
          <Button size="small" icon={<RadarChartOutlined />} loading={testLoading === `snmp-${record.id}`} onClick={() => handleSNMPPoll(record)}>SNMP</Button>
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

  const renderSwitchManagement = () => (
    <>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>Switches</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>Add Switch</Button>
      </div>

      <Card>
        <Table columns={switchColumns} dataSource={switches} rowKey="id" loading={loading} size="small" />
      </Card>

      {selectedSwitch && (
        <Card title={`Ports — ${selectedSwitch.name} (${selectedSwitch.ip})`} style={{ marginTop: 16 }}
          extra={
            <Space>
              <Button icon={<ReloadOutlined />} loading={portsLoading} onClick={() => syncPorts(selectedSwitch)}>Refresh</Button>
              <Button type="primary" icon={<PlusOutlined />} onClick={openCreatePort}>Add Port</Button>
            </Space>
          }>
          <Table columns={portColumns} dataSource={ports} rowKey="id" loading={portsLoading} size="small" />
        </Card>
      )}
    </>
  );

  const renderCommandTemplates = () => (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>Command Templates</Title>
        <Button icon={<ReloadOutlined />} onClick={fetchTemplates} loading={templatesLoading}>Refresh</Button>
      </div>
      {templates.length === 0 && !templatesLoading && (
        <Card>
          <Text type="secondary">Click Refresh to load command templates for all supported switch vendors.</Text>
        </Card>
      )}
      <Collapse
        items={templates.map((vt) => ({
          key: vt.vendor,
          label: <Text strong>{vt.label}</Text>,
          extra: <Tag>{vt.templates.length} templates</Tag>,
          children: (
            <Collapse
              size="small"
              items={vt.templates.map((t) => ({
                key: t.operation,
                label: (
                  <Space>
                    <Tag color="blue">{t.operation}</Tag>
                    <Text type="secondary">{t.description}</Text>
                  </Space>
                ),
                children: (
                  <pre style={{
                    background: '#f5f5f5',
                    padding: 12,
                    borderRadius: 4,
                    margin: 0,
                    fontSize: 13,
                    lineHeight: 1.5,
                    overflow: 'auto',
                  }}>
                    {t.template}
                  </pre>
                ),
              }))}
            />
          ),
        }))}
      />
    </div>
  );

  return (
    <div>
      <Tabs
        items={[
          {
            key: 'switches',
            label: 'Switches',
            children: renderSwitchManagement(),
          },
          {
            key: 'templates',
            label: (
              <Space>
                <CodeOutlined />
                Command Templates
              </Space>
            ),
            children: renderCommandTemplates(),
          },
        ]}
        onChange={(key) => {
          if (key === 'templates' && templates.length === 0) fetchTemplates();
        }}
      />

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
              <Input placeholder="Nexus 9336C-FX2" />
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
          <Form.Item name="agent_id" label="Agent" rules={[{ required: true }]}>
            <Select
              placeholder="Select agent"
              showSearch
              optionFilterProp="label"
              options={agents.map(a => ({
                label: `${a.name} (${a.location})`,
                value: a.id,
              }))}
            />
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
              <Input placeholder="Ethernet1/1" />
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

      {/* Test / SNMP Result Modal */}
      <Modal
        title={testResult?.type === 'snmp' ? `SNMP Poll — ${testResult?.switch_name}` : `Connection Test — ${testResult?.switch_name}`}
        open={testModalOpen}
        onCancel={() => setTestModalOpen(false)}
        footer={<Button onClick={() => setTestModalOpen(false)}>Close</Button>}
        width={700}
      >
        {testResult?.type === 'test' && (
          <div>
            <Space direction="vertical" style={{ width: '100%' }} size="middle">
              <Card size="small" title="SSH">
                <Tag color={testResult.ssh_ok ? 'green' : 'red'}>{testResult.ssh_ok ? 'Connected' : 'Failed'}</Tag>
                <Text type="secondary" style={{ marginLeft: 8 }}>{testResult.ssh_message}</Text>
                {testResult.ssh_output && (
                  <pre style={{ background: '#f5f5f5', padding: 8, borderRadius: 4, marginTop: 8, fontSize: 12, maxHeight: 150, overflow: 'auto' }}>
                    {testResult.ssh_output}
                  </pre>
                )}
              </Card>
              <Card size="small" title="SNMP">
                <Tag color={testResult.snmp_ok ? 'green' : 'red'}>{testResult.snmp_ok ? 'Connected' : 'Failed'}</Tag>
                <Text type="secondary" style={{ marginLeft: 8 }}>{testResult.snmp_message}</Text>
                {testResult.snmp_sysdescr && (
                  <pre style={{ background: '#f5f5f5', padding: 8, borderRadius: 4, marginTop: 8, fontSize: 12, maxHeight: 150, overflow: 'auto' }}>
                    {testResult.snmp_sysdescr}
                  </pre>
                )}
              </Card>
            </Space>
          </div>
        )}
        {testResult?.type === 'snmp' && testResult?.ports && (
          <Table
            size="small"
            rowKey="port_index"
            dataSource={testResult.ports}
            pagination={{ pageSize: 20 }}
            columns={[
              { title: 'Index', dataIndex: 'port_index', key: 'idx', width: 70 },
              { title: 'Port Name', dataIndex: 'port_name', key: 'name' },
              { title: 'Speed', dataIndex: 'speed', key: 'speed', render: (v: number) => {
                if (!v) return '-';
                if (v >= 1000000000) return `${v / 1000000000}G`;
                if (v >= 1000000) return `${v / 1000000}M`;
                return `${v}`;
              }},
              { title: 'Status', dataIndex: 'oper_status', key: 'status', render: (v: string) => v === 'up' ? <Tag color="green">Up</Tag> : v === 'down' ? <Tag color="red">Down</Tag> : <Tag>{v}</Tag> },
              { title: 'In (bytes)', dataIndex: 'in_octets', key: 'in', render: (v: number) => v?.toLocaleString() || '0' },
              { title: 'Out (bytes)', dataIndex: 'out_octets', key: 'out', render: (v: number) => v?.toLocaleString() || '0' },
            ]}
          />
        )}
      </Modal>
    </div>
  );
};

export default SwitchesPage;
