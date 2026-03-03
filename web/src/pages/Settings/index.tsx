import React, { useEffect, useState } from 'react';
import { Card, Typography, Form, Input, Button, ColorPicker, message, Space, Divider, Descriptions, Tag, Radio } from 'antd';
import { SaveOutlined, ReloadOutlined, DesktopOutlined } from '@ant-design/icons';
import { useAuthStore } from '../../store/auth';
import { useBrandingStore } from '../../store/branding';
import { tenantAPI } from '../../api';

const { Title, Text } = Typography;

const SettingsPage: React.FC = () => {
  const { user, fetchUser } = useAuthStore();
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
        kvm_mode: tenant.kvm_mode || 'webkvm',
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

  return (
    <div>
      <Title level={4}>Settings</Title>

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

          <Divider titlePlacement="start">KVM Console</Divider>

          <Form.Item name="kvm_mode" label={<><DesktopOutlined style={{ marginRight: 4 }} />KVM Console Mode</>}>
            <Radio.Group>
              <Space direction="vertical">
                <Radio value="webkvm">
                  <Text strong>Web KVM</Text>
                  <Text type="secondary" style={{ display: 'block', marginLeft: 24, fontSize: 12 }}>
                    Opens BMC web UI in an embedded VNC viewer (Docker + Chromium). Works even when BMC is not directly accessible from browser.
                  </Text>
                </Radio>
                <Radio value="vconsole">
                  <Text strong>Direct Console</Text>
                  <Text type="secondary" style={{ display: 'block', marginLeft: 24, fontSize: 12 }}>
                    Opens BMC web console directly in a new browser tab. Requires browser to have network access to the BMC IP.
                  </Text>
                </Radio>
              </Space>
            </Radio.Group>
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
                    kvm_mode: tenant.kvm_mode || 'webkvm',
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
