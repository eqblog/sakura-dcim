import React, { useState } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import {
  Layout,
  Menu,
  Avatar,
  Dropdown,
  Typography,
  theme,
} from 'antd';
import {
  DashboardOutlined,
  CloudServerOutlined,
  ApiOutlined,
  DesktopOutlined,
  BarChartOutlined,
  LineChartOutlined,
  TeamOutlined,
  BankOutlined,
  AuditOutlined,
  SettingOutlined,
  GlobalOutlined,
  LogoutOutlined,
  UserOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import { useAuthStore } from '../../store/auth';
import { useBrandingStore } from '../../store/branding';

const { Header, Sider, Content } = Layout;
const { Text } = Typography;

const menuItems: MenuProps['items'] = [
  {
    key: '/dashboard',
    icon: <DashboardOutlined />,
    label: 'Dashboard',
  },
  {
    key: '/servers',
    icon: <CloudServerOutlined />,
    label: 'Servers',
  },
  {
    key: '/agents',
    icon: <ApiOutlined />,
    label: 'Agents',
  },
  {
    key: 'os-mgmt',
    icon: <DesktopOutlined />,
    label: 'OS Management',
    children: [
      { key: '/os-profiles', label: 'OS Profiles' },
      { key: '/disk-layouts', label: 'Disk Layouts' },
      { key: '/scripts', label: 'Scripts' },
    ],
  },
  {
    key: 'network',
    icon: <GlobalOutlined />,
    label: 'Network',
    children: [
      { key: '/switches', label: 'Switches' },
      { key: '/ip-pools', label: 'IP Pools' },
    ],
  },
  {
    key: '/bandwidth',
    icon: <BarChartOutlined />,
    label: 'Bandwidth',
  },
  {
    key: '/monitoring',
    icon: <LineChartOutlined />,
    label: 'Monitoring',
  },
  {
    key: 'users-group',
    icon: <TeamOutlined />,
    label: 'Users',
    children: [
      { key: '/users', label: 'Users' },
      { key: '/roles', label: 'Roles' },
    ],
  },
  {
    key: '/tenants',
    icon: <BankOutlined />,
    label: 'Tenants',
  },
  {
    key: '/audit-log',
    icon: <AuditOutlined />,
    label: 'Audit Log',
  },
  {
    key: '/settings',
    icon: <SettingOutlined />,
    label: 'Settings',
  },
];

const AppLayout: React.FC = () => {
  const [collapsed, setCollapsed] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuthStore();
  const { branding } = useBrandingStore();
  const { token } = theme.useToken();

  const handleMenuClick: MenuProps['onClick'] = ({ key }) => {
    navigate(key);
  };

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  const userMenuItems: MenuProps['items'] = [
    {
      key: 'profile',
      icon: <UserOutlined />,
      label: 'Profile',
    },
    {
      type: 'divider',
    },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: 'Logout',
      danger: true,
      onClick: handleLogout,
    },
  ];

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        trigger={null}
        collapsible
        collapsed={collapsed}
        width={240}
        style={{
          overflow: 'auto',
          height: '100vh',
          position: 'fixed',
          left: 0,
          top: 0,
          bottom: 0,
          background: token.colorBgContainer,
          borderRight: `1px solid ${token.colorBorderSecondary}`,
        }}
      >
        <div
          style={{
            height: 64,
            display: 'flex',
            alignItems: 'center',
            justifyContent: collapsed ? 'center' : 'flex-start',
            padding: collapsed ? '0' : '0 24px',
            borderBottom: `1px solid ${token.colorBorderSecondary}`,
          }}
        >
          {branding.logo_url ? (
            <img src={branding.logo_url} alt={branding.name} style={{ height: 28, objectFit: 'contain' }} />
          ) : (
            <CloudServerOutlined style={{ fontSize: 24, color: token.colorPrimary }} />
          )}
          {!collapsed && (
            <Text strong style={{ marginLeft: 12, fontSize: 18 }}>
              {branding.name || 'Sakura DCIM'}
            </Text>
          )}
        </div>
        <Menu
          mode="inline"
          selectedKeys={[location.pathname]}
          defaultOpenKeys={['os-mgmt', 'network', 'users-group']}
          items={menuItems}
          onClick={handleMenuClick}
          style={{ borderRight: 0 }}
        />
      </Sider>

      <Layout style={{ marginLeft: collapsed ? 80 : 240, transition: 'margin-left 0.2s' }}>
        <Header
          style={{
            padding: '0 24px',
            background: token.colorBgContainer,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            borderBottom: `1px solid ${token.colorBorderSecondary}`,
            position: 'sticky',
            top: 0,
            zIndex: 1,
          }}
        >
          <div
            onClick={() => setCollapsed(!collapsed)}
            style={{ fontSize: 18, cursor: 'pointer' }}
          >
            {collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
          </div>
          <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
            <div style={{ cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 8 }}>
              <Avatar icon={<UserOutlined />} size="small" />
              <Text>{user?.name || user?.email || 'User'}</Text>
            </div>
          </Dropdown>
        </Header>

        <Content style={{ margin: 24, minHeight: 280 }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
};

export default AppLayout;
