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
          colorPrimary: branding.primary_color || '#667eea',
          borderRadius: 6,
        },
      }}
    >
      <AntdApp>
        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
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
