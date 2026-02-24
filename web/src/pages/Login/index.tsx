import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Card, Form, Input, Button, Typography, message, Space } from 'antd';
import { CloudServerOutlined, LockOutlined, MailOutlined } from '@ant-design/icons';
import { useAuthStore } from '../../store/auth';
import { useBrandingStore } from '../../store/branding';

const { Title, Text } = Typography;

const LoginPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const { login } = useAuthStore();
  const { branding } = useBrandingStore();
  const primaryColor = branding.primary_color || '#667eea';

  const onFinish = async (values: { email: string; password: string }) => {
    setLoading(true);
    try {
      await login(values.email, values.password);
      message.success('Login successful');
      navigate('/dashboard');
    } catch (error: any) {
      message.error(typeof error === 'string' ? error : 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: `linear-gradient(135deg, ${primaryColor} 0%, #764ba2 100%)`,
      }}
    >
      <Card
        style={{
          width: 400,
          boxShadow: '0 8px 32px rgba(0, 0, 0, 0.2)',
          borderRadius: 12,
        }}
      >
        <Space direction="vertical" align="center" style={{ width: '100%', marginBottom: 32 }}>
          {branding.logo_url ? (
            <img src={branding.logo_url} alt={branding.name} style={{ height: 48, objectFit: 'contain' }} />
          ) : (
            <CloudServerOutlined style={{ fontSize: 48, color: primaryColor }} />
          )}
          <Title level={3} style={{ margin: 0 }}>
            {branding.name || 'Sakura DCIM'}
          </Title>
          <Text type="secondary">Data Center Infrastructure Management</Text>
        </Space>

        <Form
          name="login"
          onFinish={onFinish}
          size="large"
          layout="vertical"
        >
          <Form.Item
            name="email"
            rules={[
              { required: true, message: 'Please enter your email' },
              { type: 'email', message: 'Please enter a valid email' },
            ]}
          >
            <Input
              prefix={<MailOutlined />}
              placeholder="Email"
              autoComplete="email"
            />
          </Form.Item>

          <Form.Item
            name="password"
            rules={[{ required: true, message: 'Please enter your password' }]}
          >
            <Input.Password
              prefix={<LockOutlined />}
              placeholder="Password"
              autoComplete="current-password"
            />
          </Form.Item>

          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block>
              Sign In
            </Button>
          </Form.Item>
        </Form>

        <div style={{ textAlign: 'center' }}>
          <Text type="secondary" style={{ fontSize: 12 }}>
            Default: admin@sakura-dcim.local / admin123
          </Text>
        </div>
      </Card>
    </div>
  );
};

export default LoginPage;
