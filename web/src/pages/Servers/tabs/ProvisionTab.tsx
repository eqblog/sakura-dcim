import React, { useEffect, useState, useRef, useMemo } from 'react';
import { Steps, Card, Form, Select, Input, Button, Space, Alert, Progress, Tag, Typography, Descriptions, message, Popconfirm } from 'antd';
import {
  RocketOutlined, CheckCircleOutlined, CloseCircleOutlined, LoadingOutlined,
  WarningOutlined, CheckOutlined, CloseOutlined,
} from '@ant-design/icons';
import { provisionAPI, reinstallAPI, osProfileAPI, diskLayoutAPI, ipPoolAPI } from '../../../api';
import type { Server, OSProfile, DiskLayout, InstallTask, InstallTaskStatus, PreflightResult, IPPool } from '../../../types';

const { Title, Text } = Typography;
const { TextArea } = Input;

const raidOptions = [
  { label: 'Auto (based on disk count)', value: 'auto' },
  { label: 'None (single disk / JBOD)', value: 'none' },
  { label: 'RAID 1 (mirror)', value: '1' },
  { label: 'RAID 5 (striped parity)', value: '5' },
  { label: 'RAID 10 (striped mirror)', value: '10' },
];

const statusStepMap: Record<InstallTaskStatus, number> = {
  pending: 0, pxe_booting: 1, installing: 2, post_scripts: 3, completed: 4, failed: -1,
};

const statusLabels: Record<InstallTaskStatus, string> = {
  pending: 'Pending', pxe_booting: 'PXE Booting', installing: 'Installing OS',
  post_scripts: 'Running Scripts', completed: 'Completed', failed: 'Failed',
};

interface Props {
  serverId: string;
  server: Server;
}

const ProvisionTab: React.FC<Props> = ({ serverId, server }) => {
  const [step, setStep] = useState(0);
  const [preflight, setPreflight] = useState<PreflightResult | null>(null);
  const [profiles, setProfiles] = useState<OSProfile[]>([]);
  const [layouts, setLayouts] = useState<DiskLayout[]>([]);
  const [pools, setPools] = useState<IPPool[]>([]);
  const [form] = Form.useForm();
  const [submitting, setSubmitting] = useState(false);
  const [activeTask, setActiveTask] = useState<InstallTask | null>(null);
  const [loading, setLoading] = useState(true);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const selectedVRF = Form.useWatch('vrf', form);

  useEffect(() => {
    checkActiveTask();
    return () => { if (pollRef.current) clearInterval(pollRef.current); };
  }, [serverId]);

  const checkActiveTask = async () => {
    setLoading(true);
    try {
      const { data: resp } = await reinstallAPI.status(serverId);
      if (resp.success && resp.data) {
        setActiveTask(resp.data);
        if (resp.data.status !== 'completed' && resp.data.status !== 'failed') {
          startPolling();
        }
      }
    } catch { /* no active task */ }
    setLoading(false);
  };

  const startPolling = () => {
    if (pollRef.current) clearInterval(pollRef.current);
    pollRef.current = setInterval(async () => {
      try {
        const { data: resp } = await reinstallAPI.status(serverId);
        if (resp.success && resp.data) {
          setActiveTask(resp.data);
          if (resp.data.status === 'completed' || resp.data.status === 'failed') {
            if (pollRef.current) clearInterval(pollRef.current);
          }
        }
      } catch { /* */ }
    }, 5000);
  };

  useEffect(() => {
    if (!activeTask || activeTask.status === 'completed' || activeTask.status === 'failed') {
      loadFormData();
    }
  }, [activeTask]);

  const loadFormData = async () => {
    try {
      const [preflightResp, profilesResp, layoutsResp, poolsResp] = await Promise.all([
        provisionAPI.preflight(serverId),
        osProfileAPI.list({ active_only: 'true' }),
        diskLayoutAPI.list(),
        ipPoolAPI.listAssignable(),
      ]);
      if (preflightResp.data.success) setPreflight(preflightResp.data.data as PreflightResult);
      if (profilesResp.data.success) setProfiles(profilesResp.data.data || []);
      if (layoutsResp.data.success) setLayouts(layoutsResp.data.data || []);
      if (poolsResp.data.success) setPools(poolsResp.data.data || []);
    } catch { /* */ }
  };

  const vrfOptions = useMemo(() => {
    const vrfs = [...new Set(pools.map(p => p.vrf).filter(Boolean))];
    return vrfs.map(v => ({ label: v, value: v }));
  }, [pools]);

  const filteredPools = useMemo(() => {
    if (!selectedVRF) return pools;
    return pools.filter(p => p.vrf === selectedVRF);
  }, [pools, selectedVRF]);

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setSubmitting(true);
      const payload: any = {
        os_profile_id: values.os_profile_id,
        disk_layout_id: values.disk_layout_id || undefined,
        raid_level: values.raid_level || 'auto',
        root_password: values.root_password,
        ssh_keys: values.ssh_keys ? values.ssh_keys.split('\n').filter((k: string) => k.trim()) : [],
      };
      if (!preflight?.has_ip) {
        if (values.pool_id) payload.pool_id = values.pool_id;
        if (values.vrf) payload.vrf = values.vrf;
      }
      const { data: resp } = await provisionAPI.start(serverId, payload);
      if (resp.success && resp.data) {
        message.success('Provisioning started');
        setActiveTask(resp.data);
        setStep(0);
        startPolling();
      } else {
        message.error(resp.error || 'Failed to start provisioning');
      }
    } catch { /* validation */ }
    setSubmitting(false);
  };

  const startNew = () => {
    setActiveTask(null);
    setStep(0);
    form.resetFields();
    form.setFieldsValue({ raid_level: 'auto' });
    loadFormData();
  };

  if (loading) return <Card loading />;

  // Active installation — show progress
  if (activeTask && activeTask.status !== 'completed' && activeTask.status !== 'failed') {
    const currentStep = statusStepMap[activeTask.status];
    return (
      <div>
        <Alert type="info" showIcon icon={<LoadingOutlined />}
          message="Provisioning In Progress"
          description={`Status: ${statusLabels[activeTask.status]}`}
          style={{ marginBottom: 24 }} />
        <Steps current={currentStep} items={[
          { title: 'Queued', description: 'Task created' },
          { title: 'PXE Boot', description: 'Booting via network' },
          { title: 'Installing', description: 'OS installation' },
          { title: 'Post-Install', description: 'Running scripts' },
          { title: 'Done', description: 'Completed' },
        ]} style={{ marginBottom: 24 }} />
        <Progress percent={activeTask.progress} status="active" style={{ marginBottom: 24 }} />
        {activeTask.log && (
          <Card title="Install Log" size="small">
            <pre style={{ maxHeight: 300, overflow: 'auto', fontSize: 12, margin: 0, whiteSpace: 'pre-wrap' }}>
              {activeTask.log}
            </pre>
          </Card>
        )}
      </div>
    );
  }

  // Completed or failed
  if (activeTask && (activeTask.status === 'completed' || activeTask.status === 'failed')) {
    const isSuccess = activeTask.status === 'completed';
    return (
      <div>
        <Alert type={isSuccess ? 'success' : 'error'} showIcon
          icon={isSuccess ? <CheckCircleOutlined /> : <CloseCircleOutlined />}
          message={isSuccess ? 'Provisioning Completed' : 'Provisioning Failed'}
          description={
            <Space direction="vertical">
              <Text>Started: {activeTask.started_at ? new Date(activeTask.started_at).toLocaleString() : '-'}</Text>
              <Text>Finished: {activeTask.completed_at ? new Date(activeTask.completed_at).toLocaleString() : '-'}</Text>
            </Space>
          }
          style={{ marginBottom: 24 }} />
        {activeTask.log && (
          <Card title="Install Log" size="small" style={{ marginBottom: 24 }}>
            <pre style={{ maxHeight: 300, overflow: 'auto', fontSize: 12, margin: 0, whiteSpace: 'pre-wrap' }}>
              {activeTask.log}
            </pre>
          </Card>
        )}
        <Button type="primary" icon={<RocketOutlined />} onClick={startNew}>
          Start New Provisioning
        </Button>
      </div>
    );
  }

  // Wizard steps
  const PreflightCheck = ({ ok, label }: { ok: boolean; label: string }) => (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
      {ok ? <CheckOutlined style={{ color: '#52c41a' }} /> : <CloseOutlined style={{ color: '#ff4d4f' }} />}
      <Text type={ok ? undefined : 'danger'}>{label}</Text>
    </div>
  );

  const canProceed = preflight?.has_agent && preflight?.agent_online;

  const wizardSteps = [
    {
      title: 'Preflight',
      content: (
        <>
          <Title level={5}>Server Readiness Check</Title>
          {preflight ? (
            <>
              <Card size="small" style={{ marginBottom: 16 }}>
                <PreflightCheck ok={preflight.has_mac} label={`MAC Address: ${server.mac_address || 'Not set'}`} />
                <PreflightCheck ok={preflight.has_agent} label={`Agent: ${server.agent_id ? 'Assigned' : 'Not assigned'}`} />
                <PreflightCheck ok={preflight.agent_online} label={`Agent Status: ${preflight.agent_online ? 'Online' : 'Offline'}`} />
                <PreflightCheck ok={preflight.has_ip} label={`IP Address: ${preflight.has_ip ? server.primary_ip || 'Assigned' : 'Not assigned (will auto-assign)'}`} />
                <PreflightCheck ok={preflight.has_switch_port} label={`Switch Port: ${preflight.has_switch_port ? 'Linked' : 'Not linked (VLAN automation skipped)'}`} />
              </Card>
              {preflight.warnings.length > 0 && (
                <Alert type="warning" showIcon icon={<WarningOutlined />}
                  message="Warnings"
                  description={
                    <ul style={{ margin: 0, paddingLeft: 20 }}>
                      {preflight.warnings.map((w, i) => <li key={i}>{w}</li>)}
                    </ul>
                  }
                  style={{ marginBottom: 16 }} />
              )}
              {!canProceed && (
                <Alert type="error" message="Cannot proceed: Agent must be assigned and online." style={{ marginBottom: 16 }} />
              )}
            </>
          ) : (
            <Alert type="info" message="Loading preflight checks..." />
          )}
        </>
      ),
    },
    {
      title: 'Network',
      content: (
        <>
          {preflight?.has_ip ? (
            <Alert type="success" showIcon
              message="IP Already Assigned"
              description={`Server has IP ${server.primary_ip}. Network config will be resolved from the assigned pool.`}
              style={{ marginBottom: 16 }} />
          ) : (
            <>
              <Alert type="info" showIcon
                message="Auto-assign IP"
                description="An IP address will be automatically assigned during provisioning. Optionally select a specific pool or filter by VRF."
                style={{ marginBottom: 16 }} />
              <Form.Item name="vrf" label="Filter by VRF (optional)">
                <Select allowClear placeholder="All VRFs" options={vrfOptions} />
              </Form.Item>
              <Form.Item name="pool_id" label="IP Pool (optional — leave empty for auto)">
                <Select allowClear placeholder="Auto-select best available pool"
                  options={filteredPools.map(p => ({
                    label: `${p.network}${p.vrf ? ` [${p.vrf}]` : ''} (${p.total_ips - p.used_ips} available)`,
                    value: p.id,
                    disabled: p.total_ips - p.used_ips <= 0,
                  }))} />
              </Form.Item>
            </>
          )}
        </>
      ),
    },
    {
      title: 'OS Profile',
      content: (
        <>
          <Form.Item name="os_profile_id" label="Operating System" rules={[{ required: true, message: 'Select an OS profile' }]}>
            <Select placeholder="Select operating system" showSearch optionFilterProp="label"
              options={profiles.map(p => ({
                label: `${p.name} (${p.os_family} ${p.version} / ${p.arch})`,
                value: p.id,
              }))} />
          </Form.Item>
          {(() => {
            const selected = profiles.find(p => p.id === form.getFieldValue('os_profile_id'));
            return selected ? (
              <Descriptions size="small" column={2} bordered style={{ marginTop: 8 }}>
                <Descriptions.Item label="OS Family"><Tag>{selected.os_family}</Tag></Descriptions.Item>
                <Descriptions.Item label="Architecture">{selected.arch}</Descriptions.Item>
                <Descriptions.Item label="Template Type">{selected.template_type}</Descriptions.Item>
                <Descriptions.Item label="Version">{selected.version}</Descriptions.Item>
              </Descriptions>
            ) : null;
          })()}
        </>
      ),
    },
    {
      title: 'Disk & RAID',
      content: (
        <>
          <Form.Item name="raid_level" label="RAID Configuration">
            <Select options={raidOptions} />
          </Form.Item>
          <Form.Item name="disk_layout_id" label="Disk Layout (Optional)">
            <Select allowClear placeholder="Default layout"
              options={layouts.map(l => ({
                label: `${l.name}${l.description ? ' - ' + l.description : ''}`,
                value: l.id,
              }))} />
          </Form.Item>
        </>
      ),
    },
    {
      title: 'Credentials',
      content: (
        <>
          <Form.Item name="root_password" label="Root Password"
            rules={[{ required: true, message: 'Enter root password' }, { min: 8, message: 'Minimum 8 characters' }]}>
            <Input.Password placeholder="Enter root password" />
          </Form.Item>
          <Form.Item name="ssh_keys" label="SSH Public Keys (one per line)">
            <TextArea rows={4} placeholder="ssh-rsa AAAA... user@host" style={{ fontFamily: 'monospace', fontSize: 12 }} />
          </Form.Item>
        </>
      ),
    },
    {
      title: 'Confirm',
      content: (
        <>
          <Alert type="warning" showIcon
            message="This will provision the server"
            description="The server will be assigned an IP (if needed), switch ports configured, then rebooted and installed with the selected OS. All existing data will be lost."
            style={{ marginBottom: 16 }} />
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="Server">{server.hostname} ({server.primary_ip || 'IP will be auto-assigned'})</Descriptions.Item>
            <Descriptions.Item label="MAC Address">{server.mac_address || <Text type="warning">Not set</Text>}</Descriptions.Item>
            <Descriptions.Item label="OS Profile">
              {profiles.find(p => p.id === form.getFieldValue('os_profile_id'))?.name || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="RAID Level">
              {raidOptions.find(r => r.value === form.getFieldValue('raid_level'))?.label || 'Auto'}
            </Descriptions.Item>
            <Descriptions.Item label="Disk Layout">
              {layouts.find(l => l.id === form.getFieldValue('disk_layout_id'))?.name || 'Default'}
            </Descriptions.Item>
            {!preflight?.has_ip && (
              <Descriptions.Item label="IP Pool">
                {form.getFieldValue('pool_id')
                  ? pools.find(p => p.id === form.getFieldValue('pool_id'))?.network || 'Selected'
                  : 'Auto-select'}
              </Descriptions.Item>
            )}
            <Descriptions.Item label="SSH Keys">
              {form.getFieldValue('ssh_keys') ? form.getFieldValue('ssh_keys').split('\n').filter((k: string) => k.trim()).length + ' key(s)' : 'None'}
            </Descriptions.Item>
          </Descriptions>
        </>
      ),
    },
  ];

  return (
    <div>
      <Title level={5} style={{ marginBottom: 16 }}>
        <RocketOutlined style={{ marginRight: 8 }} />
        Server Provisioning
      </Title>

      <Steps current={step} size="small" style={{ marginBottom: 24 }}
        items={wizardSteps.map(s => ({ title: s.title }))} />

      <Form form={form} layout="vertical" initialValues={{ raid_level: 'auto' }}>
        <Card>{wizardSteps[step].content}</Card>
      </Form>

      <div style={{ marginTop: 16, display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
        {step > 0 && <Button onClick={() => setStep(step - 1)}>Previous</Button>}
        {step < wizardSteps.length - 1 && (
          <Button type="primary" disabled={step === 0 && !canProceed}
            onClick={async () => {
              try {
                if (step === 2) await form.validateFields(['os_profile_id']);
                if (step === 4) await form.validateFields(['root_password']);
                setStep(step + 1);
              } catch { /* validation error */ }
            }}>
            Next
          </Button>
        )}
        {step === wizardSteps.length - 1 && (
          <Popconfirm title="Start Server Provisioning?"
            description="This will assign IP, configure switch, and install the OS. Continue?"
            onConfirm={handleSubmit} okText="Yes, Provision" okButtonProps={{ danger: true }}>
            <Button type="primary" danger loading={submitting} icon={<RocketOutlined />}>
              Start Provisioning
            </Button>
          </Popconfirm>
        )}
      </div>
    </div>
  );
};

export default ProvisionTab;
