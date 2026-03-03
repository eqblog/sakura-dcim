import React, { useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider, App as AntdApp, theme } from 'antd';
import { useAuthStore } from './store/auth';
import { useBrandingStore } from './store/branding';
import AppLayout from './components/Layout/AppLayout';
import LoginPage from './pages/Login';
import DashboardPage from './pages/Dashboard';
import ServerListPage from './pages/Servers';
import ServerDetailPage from './pages/Servers/ServerDetail';
import AgentListPage from './pages/Agents';
import AgentDetailPage from './pages/Agents/AgentDetail';
import OSProfilesPage from './pages/OSProfiles';
import DiskLayoutsPage from './pages/DiskLayouts';
import ScriptsPage from './pages/Scripts';
import SwitchesPage from './pages/Switches';
import BandwidthPage from './pages/Bandwidth';
import MonitoringPage from './pages/Monitoring';
import UsersPage from './pages/Users';
import TenantsPage from './pages/Tenants';
import AuditLogPage from './pages/AuditLog';
import SettingsPage from './pages/Settings';
import IPManagementPage from './pages/IPManagement';
import RolesPage from './pages/Roles';
import ResellerDashboard from './pages/ResellerDashboard';
import APIDocsPage from './pages/APIDocs';
import DiscoveryPage from './pages/Discovery';
import KvmWindowPage from './pages/KvmWindow';

const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated } = useAuthStore();
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  return <>{children}</>;
};

const App: React.FC = () => {
  const { isAuthenticated, fetchUser } = useAuthStore();
  const { branding, fetchBranding } = useBrandingStore();

  useEffect(() => {
    fetchBranding();
  }, [fetchBranding]);

  useEffect(() => {
    if (isAuthenticated) {
      fetchUser();
    }
  }, [isAuthenticated, fetchUser]);

  return (
    <ConfigProvider
      theme={{
        algorithm: theme.defaultAlgorithm,
        token: {
          colorPrimary: branding.primary_color || '#1677ff',
          colorBgLayout: '#f0f2f5',
          borderRadius: 8,
          boxShadow: '0 2px 8px rgba(0, 21, 41, 0.08)',
          boxShadowSecondary: '0 4px 16px rgba(0, 21, 41, 0.12)',
          colorLink: branding.primary_color || '#1677ff',
          colorLinkHover: '#4096ff',
        },
        components: {
          Card: {
            boxShadowTertiary: '0 1px 4px rgba(0, 21, 41, 0.08)',
          },
          Table: {
            rowHoverBg: '#e6f4ff',
            headerBg: '#fafafa',
          },
          Menu: {
            darkItemBg: '#001529',
            darkSubMenuItemBg: '#000c17',
            darkItemSelectedBg: branding.primary_color || '#1677ff',
            darkItemHoverBg: 'rgba(255, 255, 255, 0.08)',
            darkItemColor: 'rgba(255, 255, 255, 0.65)',
            darkItemSelectedColor: '#ffffff',
            darkGroupTitleColor: 'rgba(255, 255, 255, 0.45)',
            itemHeight: 44,
          },
          Layout: {
            siderBg: '#001529',
            headerBg: '#ffffff',
            bodyBg: '#f0f2f5',
          },
          Button: {
            primaryShadow: '0 2px 4px rgba(22, 119, 255, 0.3)',
          },
          Input: {
            activeShadow: '0 0 0 2px rgba(22, 119, 255, 0.1)',
          },
          Tabs: {
            inkBarColor: branding.primary_color || '#1677ff',
            itemActiveColor: branding.primary_color || '#1677ff',
            itemSelectedColor: branding.primary_color || '#1677ff',
          },
        },
      }}
    >
      <AntdApp>
        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            {/* Standalone KVM popup window — no AppLayout chrome */}
            <Route
              path="/kvm-window/:serverId"
              element={
                <ProtectedRoute>
                  <KvmWindowPage />
                </ProtectedRoute>
              }
            />
            <Route
              path="/"
              element={
                <ProtectedRoute>
                  <AppLayout />
                </ProtectedRoute>
              }
            >
              <Route index element={<Navigate to="/dashboard" replace />} />
              <Route path="dashboard" element={<DashboardPage />} />
              <Route path="servers" element={<ServerListPage />} />
              <Route path="servers/:id" element={<ServerDetailPage />} />
              <Route path="agents" element={<AgentListPage />} />
              <Route path="agents/:id" element={<AgentDetailPage />} />
              <Route path="discovery" element={<DiscoveryPage />} />
              <Route path="os-profiles" element={<OSProfilesPage />} />
              <Route path="disk-layouts" element={<DiskLayoutsPage />} />
              <Route path="scripts" element={<ScriptsPage />} />
              <Route path="switches" element={<SwitchesPage />} />
              <Route path="ip-pools" element={<IPManagementPage />} />
              <Route path="bandwidth" element={<BandwidthPage />} />
              <Route path="monitoring" element={<MonitoringPage />} />
              <Route path="users" element={<UsersPage />} />
              <Route path="roles" element={<RolesPage />} />
              <Route path="tenants" element={<TenantsPage />} />
              <Route path="audit-log" element={<AuditLogPage />} />
              <Route path="reseller" element={<ResellerDashboard />} />
              <Route path="api-docs" element={<APIDocsPage />} />
              <Route path="settings" element={<SettingsPage />} />
            </Route>
            <Route path="*" element={<Navigate to="/dashboard" replace />} />
          </Routes>
        </BrowserRouter>
      </AntdApp>
    </ConfigProvider>
  );
};

export default App;
