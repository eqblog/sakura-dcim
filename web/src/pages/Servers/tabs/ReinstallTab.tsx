import React, { useEffect, useState, useRef } from 'react';
import { Steps, Card, Form, Select, Input, Button, Space, Alert, Progress, Tag, Typography, Descriptions, message, Popconfirm } from 'antd';
import { CloudDownloadOutlined, CheckCircleOutlined, LoadingOutlined, CloseCircleOutlined } from '@ant-design/icons';
import { osProfileAPI, diskLayoutAPI, reinstallAPI } from '../../../api';
import type { OSProfile, DiskLayout, InstallTask, InstallTaskStatus } from '../../../types';

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
  pending: 0,
  pxe_booting: 1,
  installing: 2,
  post_scripts: 3,
  completed: 4,
  failed: -1,
};

const statusLabels: Record<InstallTaskStatus, string> = {
  pending: 'Pending',
  pxe_booting: 'PXE Booting',
  installing: 'Installing OS',
  post_scripts: 'Running Scripts',
  completed: 'Completed',
  failed: 'Failed',
};

interface Props {
  serverId: string;
}

const ReinstallTab: React.FC<Props> = ({ serverId }) => {
  const [step, setStep] = useState(0); // wizard step
  const [profiles, setProfiles] = useState<OSProfile[]>([]);
  const [layouts, setLayouts] = useState<DiskLayout[]>([]);
  const [form] = Form.useForm();
  const [submitting, setSubmitting] = useState(false);
  const [activeTask, setActiveTask] = useState<InstallTask | null>(null);
  const [loading, setLoading] = useState(true);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Check for active install task on mount
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

  // Load profiles and layouts when wizard is shown
  useEffect(() => {
    if (!activeTask || activeTask.status === 'completed' || activeTask.status === 'failed') {
      loadFormData();
    }
  }, [activeTask]);

  const loadFormData = async () => {
    try {
      const [profilesResp, layoutsResp] = await Promise.all([
        osProfileAPI.list({ active_only: 'true' }),
        diskLayoutAPI.list(),
      ]);
      if (profilesResp.data.success) setProfiles(profilesResp.data.data || []);
      if (layoutsResp.data.success) setLayouts(layoutsResp.data.data || []);
    } catch { /* */ }
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setSubmitting(true);
      const payload = {
        os_profile_id: values.os_profile_id,
        disk_layout_id: values.disk_layout_id || undefined,
        raid_level: values.raid_level || 'auto',
        root_password: values.root_password,
        ssh_keys: values.ssh_keys ? values.ssh_keys.split('\n').filter((k: string) => k.trim()) : [],
      };
      const { data: resp } = await reinstallAPI.start(serverId, payload);
      if (resp.success && resp.data) {
        message.success('Reinstall started');
        setActiveTask(resp.data);
        setStep(0);
        startPolling();
      } else {
        message.error(resp.error || 'Failed to start reinstall');
      }
    } catch { /* validation */ }
    setSubmitting(false);
  };

  const startNewInstall = () => {
    setActiveTask(null);
    setStep(0);
    form.resetFields();
    form.setFieldsValue({ raid_level: 'auto' });
  };

  if (loading) {
    return <Card loading />;
  }

  // Active installation - show progress
  if (activeTask && activeTask.status !== 'completed' && activeTask.status !== 'failed') {
    const currentStep = statusStepMap[activeTask.status];
    return (
      <div>
        <Alert
          type="info"
          showIcon
          icon={<LoadingOutlined />}
          message="OS Installation In Progress"
          description={`Status: ${statusLabels[activeTask.status]}`}
          style={{ marginBottom: 24 }}
        />
        <Steps
          current={currentStep}
          items={[
            { title: 'Queued', description: 'Task created' },
            { title: 'PXE Boot', description: 'Booting via network' },
            { title: 'Installing', description: 'OS installation' },
            { title: 'Post-Install', description: 'Running scripts' },
            { title: 'Done', description: 'Completed' },
          ]}
          style={{ marginBottom: 24 }}
        />
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

  // Completed or failed - show result + option to reinstall again
  if (activeTask && (activeTask.status === 'completed' || activeTask.status === 'failed')) {
    const isSuccess = activeTask.status === 'completed';
    return (
      <div>
        <Alert
          type={isSuccess ? 'success' : 'error'}
          showIcon
          icon={isSuccess ? <CheckCircleOutlined /> : <CloseCircleOutlined />}
          message={isSuccess ? 'Installation Completed' : 'Installation Failed'}
          description={
            <Space direction="vertical">
              <Text>Started: {activeTask.started_at ? new Date(activeTask.started_at).toLocaleString() : '-'}</Text>
              <Text>Finished: {activeTask.completed_at ? new Date(activeTask.completed_at).toLocaleString() : '-'}</Text>
            </Space>
          }
          style={{ marginBottom: 24 }}
        />
        {activeTask.log && (
          <Card title="Install Log" size="small" style={{ marginBottom: 24 }}>
            <pre style={{ maxHeight: 300, overflow: 'auto', fontSize: 12, margin: 0, whiteSpace: 'pre-wrap' }}>
              {activeTask.log}
            </pre>
          </Card>
        )}
        <Button type="primary" icon={<CloudDownloadOutlined />} onClick={startNewInstall}>
          Start New Installation
        </Button>
      </div>
    );
  }

  // Wizard - step-by-step form
  const selectedProfile = profiles.find(p => p.id === form.getFieldValue('os_profile_id'));

  const wizardSteps = [
    {
      title: 'OS Profile',
      content: (
        <>
          <Form.Item name="os_profile_id" label="Operating System" rules={[{ required: true, message: 'Select an OS profile' }]}>
            <Select
              placeholder="Select operating system"
              showSearch
              optionFilterProp="label"
              options={profiles.map(p => ({
                label: `${p.name} (${p.os_family} ${p.version} / ${p.arch})`,
                value: p.id,
              }))}
            />
          </Form.Item>
          {selectedProfile && (
            <Descriptions size="small" column={2} bordered style={{ marginTop: 8 }}>
              <Descriptions.Item label="OS Family"><Tag>{selectedProfile.os_family}</Tag></Descriptions.Item>
              <Descriptions.Item label="Architecture">{selectedProfile.arch}</Descriptions.Item>
              <Descriptions.Item label="Template Type">{selectedProfile.template_type}</Descriptions.Item>
              <Descriptions.Item label="Version">{selectedProfile.version}</Descriptions.Item>
            </Descriptions>
          )}
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
            <Select
              allowClear
              placeholder="Default layout"
              options={layouts.map(l => ({
                label: `${l.name}${l.description ? ' - ' + l.description : ''}`,
                value: l.id,
              }))}
            />
          </Form.Item>
        </>
      ),
    },
    {
      title: 'Credentials',
      content: (
        <>
          <Form.Item
            name="root_password"
            label="Root Password"
            rules={[{ required: true, message: 'Enter root password' }, { min: 8, message: 'Minimum 8 characters' }]}
          >
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
          <Alert
            type="warning"
            showIcon
            message="This will erase all data on the server"
            description="The server will be rebooted and reinstalled with the selected operating system. All existing data will be lost."
            style={{ marginBottom: 16 }}
          />
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="OS Profile">
              {profiles.find(p => p.id === form.getFieldValue('os_profile_id'))?.name || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="RAID Level">
              {raidOptions.find(r => r.value === form.getFieldValue('raid_level'))?.label || 'Auto'}
            </Descriptions.Item>
            <Descriptions.Item label="Disk Layout">
              {layouts.find(l => l.id === form.getFieldValue('disk_layout_id'))?.name || 'Default'}
            </Descriptions.Item>
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
        <CloudDownloadOutlined style={{ marginRight: 8 }} />
        OS Reinstallation
      </Title>

      <Steps current={step} size="small" style={{ marginBottom: 24 }}>
        {wizardSteps.map(s => (
          <Steps.Step key={s.title} title={s.title} />
        ))}
      </Steps>

      <Form form={form} layout="vertical" initialValues={{ raid_level: 'auto' }}>
        <Card>
          {wizardSteps[step].content}
        </Card>
      </Form>

      <div style={{ marginTop: 16, display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
        {step > 0 && (
          <Button onClick={() => setStep(step - 1)}>Previous</Button>
        )}
        {step < wizardSteps.length - 1 && (
          <Button type="primary" onClick={async () => {
            try {
              // Validate current step fields
              if (step === 0) await form.validateFields(['os_profile_id']);
              if (step === 2) await form.validateFields(['root_password']);
              setStep(step + 1);
            } catch { /* validation error */ }
          }}>
            Next
          </Button>
        )}
        {step === wizardSteps.length - 1 && (
          <Popconfirm
            title="Start OS Reinstallation?"
            description="This will erase all data on the server. Are you sure?"
            onConfirm={handleSubmit}
            okText="Yes, Reinstall"
            okButtonProps={{ danger: true }}
          >
            <Button type="primary" danger loading={submitting} icon={<CloudDownloadOutlined />}>
              Start Reinstall
            </Button>
          </Popconfirm>
        )}
      </div>
    </div>
  );
};

export default ReinstallTab;
