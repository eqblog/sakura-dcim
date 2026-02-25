import React, { useEffect, useState, useMemo } from 'react';
import { Table, Button, Modal, Select, Space, Tag, Typography, Divider, Radio, Collapse, message } from 'antd';
import { PlusOutlined, DeleteOutlined, ThunderboltOutlined, CodeOutlined, CheckCircleOutlined, ExclamationCircleOutlined } from '@ant-design/icons';
import { ipPoolAPI } from '../../../api';
import type { IPPool, IPAddress, VLANActionMode, VLANProvisionStep } from '../../../types';

const { Text } = Typography;

interface IPTabProps {
  serverId: string;
}

const IPTab: React.FC<IPTabProps> = ({ serverId }) => {
  const [addresses, setAddresses] = useState<IPAddress[]>([]);
  const [pools, setPools] = useState<IPPool[]>([]);
  const [allPools, setAllPools] = useState<IPPool[]>([]);
  const [loading, setLoading] = useState(false);
  const [assignModalOpen, setAssignModalOpen] = useState(false);
  const [selectedPool, setSelectedPool] = useState<string>('');
  const [selectedVRF, setSelectedVRF] = useState<string>('');
  const [vlanAction, setVlanAction] = useState<VLANActionMode>('execute');
  const [assigning, setAssigning] = useState(false);
  // Preview state
  const [previewSteps, setPreviewSteps] = useState<VLANProvisionStep[] | null>(null);
  const [previewAddress, setPreviewAddress] = useState<IPAddress | null>(null);

  useEffect(() => {
    fetchAssignedIPs();
    fetchPools();
  }, [serverId]);

  const fetchAssignedIPs = async () => {
    setLoading(true);
    try {
      const { data: resp } = await ipPoolAPI.listAddressesByServer(serverId);
      if (resp.success && resp.data) {
        setAddresses(resp.data);
      }
    } catch { /* ignore */ }
    setLoading(false);
  };

  const fetchPools = async () => {
    try {
      const { data: resp } = await ipPoolAPI.listAssignable();
      if (resp.success && resp.data) {
        setPools(resp.data);
        setAllPools(prev => {
          const map = new Map(prev.map(p => [p.id, p]));
          resp.data!.forEach(p => map.set(p.id, p));
          return Array.from(map.values());
        });
      }
      const { data: allResp } = await ipPoolAPI.list();
      if (allResp.success && allResp.data) {
        setAllPools(prev => {
          const map = new Map(prev.map(p => [p.id, p]));
          allResp.data!.forEach(p => map.set(p.id, p));
          return Array.from(map.values());
        });
      }
    } catch { /* ignore */ }
  };

  const vrfOptions = useMemo(() => {
    const vrfs = [...new Set(pools.map(p => p.vrf).filter(Boolean))];
    return vrfs.map(v => ({ label: v, value: v }));
  }, [pools]);

  const filteredPools = useMemo(() => {
    if (!selectedVRF) return pools;
    return pools.filter(p => p.vrf === selectedVRF);
  }, [pools, selectedVRF]);

  const resetModal = () => {
    setAssignModalOpen(false);
    setSelectedPool('');
    setSelectedVRF('');
    setVlanAction('execute');
    setPreviewSteps(null);
    setPreviewAddress(null);
  };

  const handleAssign = async () => {
    if (!selectedPool) return;
    setAssigning(true);
    try {
      const { data: resp } = await ipPoolAPI.assignNext(selectedPool, serverId, vlanAction);
      if (resp.success && resp.data) {
        const result = resp.data;
        if (vlanAction === 'preview' && result.vlan_steps?.length) {
          setPreviewSteps(result.vlan_steps);
          setPreviewAddress(result.address);
          message.info('Preview generated — review the commands below');
        } else {
          message.success(`IP ${result.address?.address} assigned`);
          if (result.vlan_steps?.length) {
            showVLANStepsResult(result.vlan_steps);
          }
          resetModal();
          fetchAssignedIPs();
          fetchPools();
        }
      } else {
        message.error(resp.error || 'Failed to assign IP');
      }
    } catch {
      message.error('Failed to assign IP');
    }
    setAssigning(false);
  };

  const handleAutoAssign = async () => {
    setAssigning(true);
    try {
      const { data: resp } = await ipPoolAPI.autoAssign(serverId, undefined, selectedVRF || undefined, vlanAction);
      if (resp.success && resp.data) {
        const result = resp.data;
        if (vlanAction === 'preview' && result.vlan_steps?.length) {
          setPreviewSteps(result.vlan_steps);
          setPreviewAddress(result.address);
          message.info('Preview generated — review the commands below');
        } else {
          message.success(`IP ${result.address?.address} auto-assigned`);
          if (result.vlan_steps?.length) {
            showVLANStepsResult(result.vlan_steps);
          }
          resetModal();
          fetchAssignedIPs();
          fetchPools();
        }
      } else {
        message.error(resp.error || 'No available IP pool found');
      }
    } catch {
      message.error('Failed to auto-assign IP');
    }
    setAssigning(false);
  };

  const handleConfirmPreview = async () => {
    // Re-run the assignment with execute mode
    setAssigning(true);
    try {
      let resp;
      if (selectedPool) {
        resp = (await ipPoolAPI.assignNext(selectedPool, serverId, 'execute')).data;
      } else {
        resp = (await ipPoolAPI.autoAssign(serverId, undefined, selectedVRF || undefined, 'execute')).data;
      }
      if (resp.success && resp.data) {
        message.success(`IP ${resp.data.address?.address} assigned`);
        if (resp.data.vlan_steps?.length) {
          showVLANStepsResult(resp.data.vlan_steps);
        }
        resetModal();
        fetchAssignedIPs();
        fetchPools();
      } else {
        message.error(resp.error || 'Failed to assign IP');
      }
    } catch {
      message.error('Failed to assign IP');
    }
    setAssigning(false);
  };

  const showVLANStepsResult = (steps: VLANProvisionStep[]) => {
    const okCount = steps.filter(s => s.status === 'ok').length;
    const errCount = steps.filter(s => s.status === 'error').length;
    if (errCount > 0) {
      message.warning(`VLAN provisioning: ${okCount} ok, ${errCount} errors`);
    } else if (okCount > 0) {
      message.success(`VLAN infrastructure configured (${okCount} steps)`);
    }
  };

  const handleUnassign = async (addr: IPAddress) => {
    try {
      const { data: resp } = await ipPoolAPI.updateAddress(addr.pool_id, addr.id, {
        server_id: null,
        status: 'available',
      });
      if (resp.success) {
        message.success('IP unassigned');
        fetchAssignedIPs();
        fetchPools();
      }
    } catch {
      message.error('Failed to unassign IP');
    }
  };

  const stepActionLabel = (action: string) => {
    switch (action) {
      case 'create_vlan': return 'Create VLAN';
      case 'create_svi': return 'Create SVI / VLANIF';
      case 'bind_vrf': return 'Bind VRF';
      case 'save_config': return 'Save Configuration';
      default: return action;
    }
  };

  const stepStatusIcon = (status: string) => {
    switch (status) {
      case 'ok': return <CheckCircleOutlined style={{ color: '#52c41a' }} />;
      case 'error': return <ExclamationCircleOutlined style={{ color: '#ff4d4f' }} />;
      default: return <CodeOutlined style={{ color: '#1677ff' }} />;
    }
  };

  const columns = [
    {
      title: 'Address',
      dataIndex: 'address',
      key: 'address',
      render: (ip: string) => <Text code>{ip}</Text>,
    },
    {
      title: 'Pool',
      dataIndex: 'pool_id',
      key: 'pool_id',
      render: (poolId: string) => {
        const pool = allPools.find(p => p.id === poolId) || pools.find(p => p.id === poolId);
        return pool ? <Tag>{pool.network}</Tag> : poolId.slice(0, 8);
      },
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (s: string) => (
        <Tag color={s === 'assigned' ? 'blue' : s === 'reserved' ? 'orange' : 'default'}>{s}</Tag>
      ),
    },
    {
      title: 'Note',
      dataIndex: 'note',
      key: 'note',
    },
    {
      title: '',
      key: 'actions',
      width: 80,
      render: (_: any, record: IPAddress) => (
        <Button
          type="text"
          danger
          size="small"
          icon={<DeleteOutlined />}
          onClick={() => handleUnassign(record)}
        >
          Unassign
        </Button>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Text strong>Assigned IP Addresses</Text>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setAssignModalOpen(true)}>
          Assign IP
        </Button>
      </div>

      <Table
        dataSource={addresses}
        columns={columns}
        rowKey="id"
        loading={loading}
        pagination={false}
        size="small"
      />

      <Modal
        title="Assign IP Address"
        open={assignModalOpen}
        onCancel={resetModal}
        width={600}
        footer={previewSteps ? [
          <Button key="back" onClick={() => setPreviewSteps(null)}>
            Back
          </Button>,
          <Button
            key="confirm"
            type="primary"
            loading={assigning}
            onClick={handleConfirmPreview}
          >
            Confirm & Execute
          </Button>,
        ] : [
          <Button key="cancel" onClick={resetModal}>
            Cancel
          </Button>,
          <Button
            key="auto"
            icon={<ThunderboltOutlined />}
            loading={assigning}
            onClick={handleAutoAssign}
          >
            Auto-assign
          </Button>,
          <Button
            key="assign"
            type="primary"
            disabled={!selectedPool}
            loading={assigning}
            onClick={handleAssign}
          >
            {vlanAction === 'preview' ? 'Preview VLAN Commands' : 'Assign from Selected Pool'}
          </Button>,
        ]}
      >
        {previewSteps ? (
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            {previewAddress && (
              <div>
                <Text strong>Assigned IP: </Text>
                <Text code>{previewAddress.address}</Text>
              </div>
            )}
            <Text strong>VLAN Provisioning Commands (Dry Run):</Text>
            <Collapse
              defaultActiveKey={previewSteps.map((_, i) => String(i))}
              items={previewSteps.map((step, i) => ({
                key: String(i),
                label: (
                  <Space>
                    {stepStatusIcon(step.status)}
                    <span>{stepActionLabel(step.action)}</span>
                  </Space>
                ),
                children: (
                  <pre style={{ margin: 0, fontSize: 12, background: '#f5f5f5', padding: 8, borderRadius: 4 }}>
                    {step.commands.join('\n')}
                  </pre>
                ),
              }))}
            />
            <Text type="secondary" style={{ fontSize: 12 }}>
              Click "Confirm & Execute" to run these commands on the switch.
            </Text>
          </Space>
        ) : (
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            <div>
              <Text type="secondary">Filter by VRF (optional):</Text>
              <Select
                placeholder="All VRFs"
                value={selectedVRF || undefined}
                onChange={(v) => { setSelectedVRF(v || ''); setSelectedPool(''); }}
                allowClear
                style={{ width: '100%', marginTop: 4 }}
                options={vrfOptions}
              />
            </div>

            <Divider style={{ margin: '8px 0' }} />

            <div>
              <Text>Select a specific IP pool:</Text>
              <Select
                placeholder="Select IP Pool"
                value={selectedPool || undefined}
                onChange={setSelectedPool}
                style={{ width: '100%', marginTop: 4 }}
                options={filteredPools.map(p => ({
                  label: `${p.network}${p.vrf ? ` [${p.vrf}]` : ''} (${p.total_ips - p.used_ips} available)`,
                  value: p.id,
                  disabled: p.total_ips - p.used_ips <= 0,
                }))}
              />
            </div>

            <Divider style={{ margin: '8px 0' }} />

            <div>
              <Text>VLAN Action:</Text>
              <Radio.Group
                value={vlanAction}
                onChange={e => setVlanAction(e.target.value)}
                style={{ display: 'flex', flexDirection: 'column', gap: 4, marginTop: 4 }}
              >
                <Radio value="execute">Execute (auto-configure VLAN/SVI/VRF on switch)</Radio>
                <Radio value="preview">Preview (dry run — show commands only)</Radio>
                <Radio value="skip">Don't execute (skip VLAN automation)</Radio>
              </Radio.Group>
            </div>

            <Text type="secondary" style={{ fontSize: 12 }}>
              Or click "Auto-assign" to automatically pick from the best available pool{selectedVRF ? ` in VRF "${selectedVRF}"` : ''}.
            </Text>
          </Space>
        )}
      </Modal>
    </div>
  );
};

export default IPTab;
