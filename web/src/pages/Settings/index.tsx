import React, { useEffect, useState } from 'react';
import { Card, Typography, Form, Input, Button, ColorPicker, message, Space, Divider, Descriptions, Tag, Radio } from 'antd';
import { SaveOutlined, ReloadOutlined, DesktopOutlined } from '@ant-design/icons';
import { useAuthStore } from '../../store/auth';
import { useBrandingStore } from '../../store/branding';
import { tenantAPI } from '../../api';

const { Title, Text } = Typography;

const KVM_MODE_KEY = 'sakura_kvm_mode';
type KvmMode = 'webkvm' | 'vconsole';

const SettingsPage: React.FC = () => {
  const { user, fetchUser } = useAuthStore();
  const [kvmMode, setKvmMode] = useState<KvmMode>(() => (localStorage.getItem(KVM_MODE_KEY) as KvmMode) || 'webkvm');
  const { setBrandingFromTenant } = useBrandingStore();
  const [form] = Form.useForm();
  const [saving, setSaving] = useState(false);

  const tenantId = user?.tenant?.id || user?.tenant_id;
  const tenant = user?.tenant;

  useEffect(() => {
    if (tenant) {
      form.setFieldsValue({
        name: tenant.name,
        slug: tenant.slug,
        custom_domain: tenant.custom_domain || '',
        logo_url: tenant.logo_url || '',
        favicon_url: tenant.favicon_url || '',
        primary_color: tenant.primary_color || '#667eea',
      });
    }
  }, [tenant, form]);

  const handleSave = async (values: any) => {
    if (!tenantId) return;
    setSaving(true);
    try {
      const payload = {
        ...values,
        primary_color:
          typeof values.primary_color === 'string'
            ? values.primary_color
            : values.primary_color?.toHexString?.() || undefined,
      };
      const { data: resp } = await tenantAPI.update(tenantId, payload);
      if (resp.success) {
        message.success('Settings saved');
        await fetchUser();
        if (resp.data) setBrandingFromTenant(resp.data);
      } else {
        message.error(resp.error || 'Failed to save settings');
      }
    } catch {
      message.error('Failed to save settings');
    }
    setSaving(false);
  };

  const handleKvmModeChange = (mode: KvmMode) => {
    setKvmMode(mode);
    localStorage.setItem(KVM_MODE_KEY, mode);
    message.success(`KVM mode set to ${mode === 'webkvm' ? 'Web KVM (Full BMC UI)' : 'Direct vConsole'}`);
  };

  return (
    <div>
      <Title level={4}>Settings</Title>

      <Card
        title={<><DesktopOutlined style={{ marginRight: 8 }} />KVM Preferences</>}
        style={{ marginBottom: 16 }}
      >
        <Space direction="vertical" style={{ width: '100%' }} size={12}>
          <div>
            <Text strong style={{ display: 'block', marginBottom: 8 }}>KVM Console Mode</Text>
            <Radio.Group value={kvmMode} onChange={(e) => handleKvmModeChange(e.target.value)}>
              <Space direction="vertical">
                <Radio value="webkvm">
                  <Text strong>Web KVM (Full BMC)</Text>
                  <Text type="secondary" style={{ display: 'block', marginLeft: 24, fontSize: 12 }}>
                    Docker-isolated Chromium opens the BMC web UI login page. You can navigate to any BMC feature.
                  </Text>
                </Radio>
                <Radio value="vconsole">
                  <Text strong>Direct vConsole</Text>
                  <Text type="secondary" style={{ display: 'block', marginLeft: 24, fontSize: 12 }}>
                    Docker-isolated Chromium opens the BMC virtual console (KVM) page directly, skipping the dashboard.
                    Supported: iDRAC vConsole, iLO Remote Console, Supermicro iKVM, Lenovo XCC, Huawei iBMC.
                  </Text>
                </Radio>
              </Space>
            </Radio.Group>
          </div>
          <Text type="secondary" style={{ fontSize: 12 }}>
            Current mode: <Tag color={kvmMode === 'webkvm' ? 'blue' : 'green'}>{kvmMode === 'webkvm' ? 'Web KVM' : 'Direct vConsole'}</Tag>
            &mdash; Both modes use Docker container isolation.
          </Text>
        </Space>
      </Card>

      <Card title="Tenant Information" style={{ marginBottom: 16 }}>
        <Descriptions column={2} size="small">
          <Descriptions.Item label="Tenant ID">
            <Text copyable={{ text: tenantId }}>{tenantId ? `${tenantId.slice(0, 8)}...` : '-'}</Text>
          </Descriptions.Item>
          <Descriptions.Item label="Role">
            <Tag color="blue">{user?.role?.name || 'N/A'}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="Email">{user?.email || '-'}</Descriptions.Item>
          <Descriptions.Item label="Name">{user?.name || '-'}</Descriptions.Item>
        </Descriptions>
      </Card>

      <Card title="Branding & White-Label">
        <Form form={form} layout="vertical" onFinish={handleSave}>
          <Form.Item
            name="name"
            label="Organization Name"
            rules={[{ required: true, message: 'Please enter a name' }]}
          >
            <Input placeholder="Acme Hosting" />
          </Form.Item>

          <Form.Item
            name="slug"
            label="URL Slug"
            rules={[
              { required: true, message: 'Please enter a slug' },
              { pattern: /^[a-z0-9-]+$/, message: 'Lowercase alphanumeric with hyphens only' },
            ]}
          >
            <Input placeholder="acme-hosting" />
          </Form.Item>

          <Divider titlePlacement="start">Appearance</Divider>

          <Form.Item name="primary_color" label="Primary Color">
            <ColorPicker format="hex" showText />
          </Form.Item>

          <Form.Item name="logo_url" label="Logo URL">
            <Input placeholder="https://example.com/logo.png" />
          </Form.Item>

          {form.getFieldValue('logo_url') && (
            <div style={{ marginBottom: 16 }}>
              <Text type="secondary" style={{ display: 'block', marginBottom: 8 }}>Preview:</Text>
              <img
                src={form.getFieldValue('logo_url')}
                alt="Logo preview"
                style={{ maxHeight: 48, maxWidth: 200, objectFit: 'contain', border: '1px solid #f0f0f0', padding: 4, borderRadius: 4 }}
                onError={(e) => { (e.target as HTMLImageElement).style.display = 'none'; }}
              />
            </div>
          )}

          <Form.Item name="favicon_url" label="Favicon URL">
            <Input placeholder="https://example.com/favicon.ico" />
          </Form.Item>

          <Divider titlePlacement="start">Custom Domain</Divider>

          <Form.Item name="custom_domain" label="Custom Domain">
            <Input placeholder="panel.acme.com" />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" icon={<SaveOutlined />} loading={saving}>
                Save Settings
              </Button>
              <Button icon={<ReloadOutlined />} onClick={() => {
                if (tenant) {
                  form.setFieldsValue({
                    name: tenant.name,
                    slug: tenant.slug,
                    custom_domain: tenant.custom_domain || '',
                    logo_url: tenant.logo_url || '',
                    favicon_url: tenant.favicon_url || '',
                    primary_color: tenant.primary_color || '#667eea',
                  });
                }
              }}>
                Reset
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
};

export default SettingsPage;
