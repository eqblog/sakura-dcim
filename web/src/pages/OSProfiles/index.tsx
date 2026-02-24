import React, { useEffect, useState } from 'react';
import { Card, Typography, Table, Button, Space, Modal, Form, Input, Select, Switch, Tag, message, Popconfirm } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { osProfileAPI } from '../../api';
import type { OSProfile } from '../../types';

const { Title } = Typography;
const { TextArea } = Input;

const templateTypes = [
  { label: 'Kickstart (RHEL/CentOS)', value: 'kickstart' },
  { label: 'Preseed (Debian/Ubuntu)', value: 'preseed' },
  { label: 'Autoinstall (Ubuntu 20.04+)', value: 'autoinstall' },
  { label: 'Cloud-Init', value: 'cloud-init' },
];

const osFamilies = [
  'ubuntu', 'debian', 'centos', 'rocky', 'alma', 'rhel', 'fedora', 'windows', 'proxmox', 'esxi',
];

const OSProfilesPage: React.FC = () => {
  const [profiles, setProfiles] = useState<OSProfile[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<OSProfile | null>(null);
  const [form] = Form.useForm();

  const fetchProfiles = async () => {
    setLoading(true);
    try {
      const { data: resp } = await osProfileAPI.list();
      if (resp.success) setProfiles(resp.data || []);
    } catch { /* */ }
    setLoading(false);
  };

  useEffect(() => { fetchProfiles(); }, []);

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({ arch: 'amd64', template_type: 'kickstart', is_active: true });
    setModalOpen(true);
  };

  const openEdit = (record: OSProfile) => {
    setEditing(record);
    form.setFieldsValue(record);
    setModalOpen(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (editing) {
        const { data: resp } = await osProfileAPI.update(editing.id, values);
        if (resp.success) { message.success('OS profile updated'); }
        else { message.error(resp.error); return; }
      } else {
        const { data: resp } = await osProfileAPI.create(values);
        if (resp.success) { message.success('OS profile created'); }
        else { message.error(resp.error); return; }
      }
      setModalOpen(false);
      fetchProfiles();
    } catch { /* validation */ }
  };

  const handleDelete = async (id: string) => {
    const { data: resp } = await osProfileAPI.delete(id);
    if (resp.success) { message.success('Deleted'); fetchProfiles(); }
    else { message.error(resp.error); }
  };

  const columns: ColumnsType<OSProfile> = [
    { title: 'Name', dataIndex: 'name', key: 'name', sorter: (a, b) => a.name.localeCompare(b.name) },
    { title: 'OS Family', dataIndex: 'os_family', key: 'os_family', render: (v: string) => <Tag>{v}</Tag> },
    { title: 'Version', dataIndex: 'version', key: 'version' },
    { title: 'Arch', dataIndex: 'arch', key: 'arch' },
    { title: 'Template', dataIndex: 'template_type', key: 'template_type' },
    { title: 'Active', dataIndex: 'is_active', key: 'is_active', render: (v: boolean) => v ? <Tag color="green">Active</Tag> : <Tag>Inactive</Tag> },
    {
      title: 'Actions', key: 'actions', width: 120,
      render: (_, record) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => openEdit(record)} />
          <Popconfirm title="Delete this profile?" onConfirm={() => handleDelete(record.id)}>
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4}>OS Profiles</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>Add Profile</Button>
      </div>

      <Card>
        <Table columns={columns} dataSource={profiles} rowKey="id" loading={loading} size="small" />
      </Card>

      <Modal
        title={editing ? 'Edit OS Profile' : 'Create OS Profile'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        width={800}
        okText={editing ? 'Update' : 'Create'}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true }]}>
            <Input placeholder="Ubuntu 22.04 LTS" />
          </Form.Item>
          <Space style={{ width: '100%' }} size="middle">
            <Form.Item name="os_family" label="OS Family" rules={[{ required: true }]} style={{ flex: 1 }}>
              <Select options={osFamilies.map(f => ({ label: f, value: f }))} placeholder="Select OS family" />
            </Form.Item>
            <Form.Item name="version" label="Version" style={{ flex: 1 }}>
              <Input placeholder="22.04" />
            </Form.Item>
            <Form.Item name="arch" label="Architecture" style={{ flex: 1 }}>
              <Select options={[{ label: 'amd64', value: 'amd64' }, { label: 'arm64', value: 'arm64' }]} />
            </Form.Item>
          </Space>
          <Form.Item name="kernel_url" label="Kernel URL">
            <Input placeholder="http://archive.ubuntu.com/.../vmlinuz" />
          </Form.Item>
          <Form.Item name="initrd_url" label="Initrd URL">
            <Input placeholder="http://archive.ubuntu.com/.../initrd.gz" />
          </Form.Item>
          <Form.Item name="boot_args" label="Boot Arguments">
            <Input placeholder="auto=true priority=critical" />
          </Form.Item>
          <Form.Item name="template_type" label="Template Type" rules={[{ required: true }]}>
            <Select options={templateTypes} />
          </Form.Item>
          <Form.Item name="template" label="Template Content">
            <TextArea rows={12} placeholder="Paste your Kickstart / Preseed / cloud-init template here..." style={{ fontFamily: 'monospace', fontSize: 12 }} />
          </Form.Item>
          <Form.Item name="is_active" label="Active" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default OSProfilesPage;
