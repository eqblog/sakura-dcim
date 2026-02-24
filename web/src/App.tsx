import React, { useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider, App as AntdApp, theme } from 'antd';
import { useAuthStore } from './store/auth';
import AppLayout from './components/Layout/AppLayout';
import LoginPage from './pages/Login';
import DashboardPage from './pages/Dashboard';
import ServerListPage from './pages/Servers';
import AgentListPage from './pages/Agents';
import OSProfilesPage from './pages/OSProfiles';
import BandwidthPage from './pages/Bandwidth';
import MonitoringPage from './pages/Monitoring';
import UsersPage from './pages/Users';
import TenantsPage from './pages/Tenants';
import AuditLogPage from './pages/AuditLog';
import SettingsPage from './pages/Settings';
import IPManagementPage from './pages/IPManagement';

const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated } = useAuthStore();
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  return <>{children}</>;
};

const App: React.FC = () => {
  const { isAuthenticated, fetchUser } = useAuthStore();

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
          colorPrimary: '#667eea',
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
              <Route path="agents" element={<AgentListPage />} />
              <Route path="os-profiles" element={<OSProfilesPage />} />
              <Route path="disk-layouts" element={<OSProfilesPage />} />
              <Route path="scripts" element={<OSProfilesPage />} />
              <Route path="switches" element={<BandwidthPage />} />
              <Route path="ip-pools" element={<IPManagementPage />} />
              <Route path="bandwidth" element={<BandwidthPage />} />
              <Route path="monitoring" element={<MonitoringPage />} />
              <Route path="users" element={<UsersPage />} />
              <Route path="tenants" element={<TenantsPage />} />
              <Route path="audit-log" element={<AuditLogPage />} />
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
