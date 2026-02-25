import React, { useEffect, useState } from 'react';
import { Table, Button, Space, Tag, Modal, Select, message, Popconfirm, Empty } from 'antd';
import { PlusOutlined, DisconnectOutlined, ApiOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { switchAPI } from '../../../api';
import type { Switch, SwitchPort } from '../../../types';

interface Props {
  serverId: string;
}

interface SwitchPortWithSwitch extends SwitchPort {
  switch_name: string;
  switch_ip: string;
}

const NetworkTab: React.FC<Props> = ({ serverId }) => {
  const [ports, setPorts] = useState<SwitchPortWithSwitch[]>([]);
  const [loading, setLoading] = useState(false);
  const [linkModalOpen, setLinkModalOpen] = useState(false);

  // Link modal state
  const [switches, setSwitches] = useState<Switch[]>([]);
  const [switchesLoading, setSwitchesLoading] = useState(false);
  const [selectedSwitchId, setSelectedSwitchId] = useState<string | null>(null);
  const [availablePorts, setAvailablePorts] = useState<SwitchPort[]>([]);
  const [portsLoading, setPortsLoading] = useState(false);
  const [selectedPortId, setSelectedPortId] = useState<string | null>(null);

  const fetchLinkedPorts = async () => {
    setLoading(true);
    try {
      const { data: resp } = await switchAPI.listPortsByServer(serverId);
      if (resp.success) setPorts((resp.data as SwitchPortWithSwitch[]) || []);
    } catch { /* */ }
    setLoading(false);
  };

  useEffect(() => { fetchLinkedPorts(); }, [serverId]);

  const handleUnlink = async (portId: string) => {
    try {
      const { data: resp } = await switchAPI.unlinkPort(portId);
      if (resp.success) {
        message.success('Port unlinked');
        fetchLinkedPorts();
      } else {
        message.error(resp.error);
      }
    } catch {
      message.error('Failed to unlink port');
    }
  };

  const openLinkModal = async () => {
    setLinkModalOpen(true);
    setSelectedSwitchId(null);
    setSelectedPortId(null);
    setAvailablePorts([]);
    setSwitchesLoading(true);
    try {
      const { data: resp } = await switchAPI.list();
      if (resp.success) setSwitches(resp.data || []);
    } catch { /* */ }
    setSwitchesLoading(false);
  };

  const handleSwitchChange = async (switchId: string) => {
    setSelectedSwitchId(switchId);
    setSelectedPortId(null);
    setPortsLoading(true);
    try {
      const { data: resp } = await switchAPI.listPorts(switchId);
      if (resp.success) {
        // Only show ports that are not already linked to a server
        const unlinked = (resp.data || []).filter((p: SwitchPort) => !p.server_id);
        setAvailablePorts(unlinked);
      }
    } catch { /* */ }
    setPortsLoading(false);
  };

  const handleLink = async () => {
    if (!selectedPortId) return;
    try {
      const { data: resp } = await switchAPI.linkPort(selectedPortId, serverId);
      if (resp.success) {
        message.success('Port linked to server');
        setLinkModalOpen(false);
        fetchLinkedPorts();
      } else {
        message.error(resp.error);
      }
    } catch {
      message.error('Failed to link port');
    }
  };

  const columns: ColumnsType<SwitchPortWithSwitch> = [
    {
      title: 'Switch', key: 'switch',
      render: (_, r) => (
        <span>{r.switch_name} <Tag>{r.switch_ip}</Tag></span>
      ),
    },
    { title: 'Port', dataIndex: 'port_name', key: 'port_name' },
    {
      title: 'Speed', dataIndex: 'speed_mbps', key: 'speed',
      render: (v: number) => v >= 1000 ? `${v / 1000}G` : `${v}M`,
    },
    {
      title: 'VLAN', dataIndex: 'vlan_id', key: 'vlan',
      render: (v: number) => v > 0 ? <Tag color="blue">{v}</Tag> : '-',
    },
    {
      title: 'Admin', dataIndex: 'admin_status', key: 'admin',
      render: (v: string) => v === 'up' ? <Tag color="green">Up</Tag> : <Tag color="red">Down</Tag>,
    },
    {
      title: 'Oper', dataIndex: 'oper_status', key: 'oper',
      render: (v: string) => v === 'up' ? <Tag color="green">Up</Tag> : v === 'down' ? <Tag color="red">Down</Tag> : <Tag>Unknown</Tag>,
    },
    { title: 'Description', dataIndex: 'description', key: 'desc', ellipsis: true },
    {
      title: 'Actions', key: 'actions', width: 100,
      render: (_, record) => (
        <Popconfirm title="Unlink this port from server?" onConfirm={() => handleUnlink(record.id)}>
          <Button size="small" danger icon={<DisconnectOutlined />} title="Unlink" />
        </Popconfirm>
      ),
    },
  ];

  if (!loading && ports.length === 0) {
    return (
      <div>
        <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 16 }}>
          <Button type="primary" icon={<PlusOutlined />} onClick={openLinkModal}>Link Port</Button>
        </div>
        <Empty
          image={<ApiOutlined style={{ fontSize: 48, color: '#ccc' }} />}
          description="No switch ports linked to this server. Click 'Link Port' to associate a switch port."
        />
        {renderLinkModal()}
      </div>
    );
  }

  function renderLinkModal() {
    return (
      <Modal
        title="Link Switch Port"
        open={linkModalOpen}
        onOk={handleLink}
        onCancel={() => setLinkModalOpen(false)}
        okText="Link"
        okButtonProps={{ disabled: !selectedPortId }}
        width={500}
      >
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          <div>
            <div style={{ marginBottom: 4, fontWeight: 500 }}>Switch</div>
            <Select
              style={{ width: '100%' }}
              placeholder="Select a switch"
              loading={switchesLoading}
              value={selectedSwitchId}
              onChange={handleSwitchChange}
              options={switches.map(s => ({ label: `${s.name} (${s.ip})`, value: s.id }))}
              showSearch
              filterOption={(input, option) =>
                (option?.label as string || '').toLowerCase().includes(input.toLowerCase())
              }
            />
          </div>
          <div>
            <div style={{ marginBottom: 4, fontWeight: 500 }}>Port</div>
            <Select
              style={{ width: '100%' }}
              placeholder={selectedSwitchId ? 'Select an available port' : 'Select a switch first'}
              loading={portsLoading}
              disabled={!selectedSwitchId}
              value={selectedPortId}
              onChange={setSelectedPortId}
              options={availablePorts.map(p => ({
                label: `${p.port_name} (${p.speed_mbps >= 1000 ? `${p.speed_mbps / 1000}G` : `${p.speed_mbps}M`}${p.vlan_id > 0 ? ` VLAN ${p.vlan_id}` : ''})`,
                value: p.id,
              }))}
              showSearch
              filterOption={(input, option) =>
                (option?.label as string || '').toLowerCase().includes(input.toLowerCase())
              }
              notFoundContent={selectedSwitchId && !portsLoading ? 'No available (unlinked) ports' : undefined}
            />
          </div>
        </Space>
      </Modal>
    );
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={openLinkModal}>Link Port</Button>
      </div>
      <Table columns={columns} dataSource={ports} rowKey="id" loading={loading} size="small" />
      {renderLinkModal()}
    </div>
  );
};

export default NetworkTab;
