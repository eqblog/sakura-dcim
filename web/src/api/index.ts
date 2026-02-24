import client from './client';
import type {
  APIResponse,
  PaginatedResult,
  LoginRequest,
  LoginResponse,
  Server,
  ServerCreateRequest,
  Agent,
  User,
  Role,
  Tenant,
  OSProfile,
  AuditLog,
} from '../types';

// Auth
export const authAPI = {
  login: (data: LoginRequest) =>
    client.post<APIResponse<LoginResponse>>('/auth/login', data),
  logout: () =>
    client.post<APIResponse>('/auth/logout'),
  refresh: (refreshToken: string) =>
    client.post<APIResponse<LoginResponse>>('/auth/refresh', { refresh_token: refreshToken }),
  me: () =>
    client.get<APIResponse<User>>('/auth/me'),
};

// Servers
export const serverAPI = {
  list: (params?: Record<string, any>) =>
    client.get<APIResponse<PaginatedResult<Server>>>('/servers', { params }),
  get: (id: string) =>
    client.get<APIResponse<Server>>(`/servers/${id}`),
  create: (data: ServerCreateRequest) =>
    client.post<APIResponse<Server>>('/servers', data),
  update: (id: string, data: Partial<ServerCreateRequest>) =>
    client.put<APIResponse<Server>>(`/servers/${id}`, data),
  delete: (id: string) =>
    client.delete<APIResponse>(`/servers/${id}`),
  power: (id: string, action: string) =>
    client.post<APIResponse>(`/servers/${id}/power`, { action }),
  powerStatus: (id: string) =>
    client.get<APIResponse>(`/servers/${id}/power`),
  sensors: (id: string) =>
    client.get<APIResponse>(`/servers/${id}/sensors`),
  inventory: (id: string) =>
    client.get<APIResponse>(`/servers/${id}/inventory`),
  inventoryScan: (id: string) =>
    client.post<APIResponse>(`/servers/${id}/inventory/scan`),
  reinstall: (id: string, data: any) =>
    client.post<APIResponse>(`/servers/${id}/reinstall`, data),
  reinstallStatus: (id: string) =>
    client.get<APIResponse>(`/servers/${id}/reinstall/status`),
  kvm: (id: string) =>
    client.get<APIResponse>(`/servers/${id}/kvm`),
  bandwidth: (id: string, params?: Record<string, any>) =>
    client.get<APIResponse>(`/servers/${id}/bandwidth`, { params }),
  metrics: (id: string, params?: Record<string, any>) =>
    client.get<APIResponse>(`/servers/${id}/metrics`, { params }),
};

// Agents
export const agentAPI = {
  list: (params?: Record<string, any>) =>
    client.get<APIResponse<PaginatedResult<Agent>>>('/agents', { params }),
  get: (id: string) =>
    client.get<APIResponse<Agent>>(`/agents/${id}`),
  create: (data: { name: string; location: string; capabilities?: string[] }) =>
    client.post<APIResponse<{ agent: Agent; token: string }>>('/agents', data),
  update: (id: string, data: Partial<Agent>) =>
    client.put<APIResponse<Agent>>(`/agents/${id}`, data),
  delete: (id: string) =>
    client.delete<APIResponse>(`/agents/${id}`),
};

// Users
export const userAPI = {
  list: (params?: Record<string, any>) =>
    client.get<APIResponse<PaginatedResult<User>>>('/users', { params }),
  create: (data: any) =>
    client.post<APIResponse<User>>('/users', data),
  update: (id: string, data: any) =>
    client.put<APIResponse<User>>(`/users/${id}`, data),
  delete: (id: string) =>
    client.delete<APIResponse>(`/users/${id}`),
};

// Roles
export const roleAPI = {
  list: () =>
    client.get<APIResponse<Role[]>>('/roles'),
  create: (data: any) =>
    client.post<APIResponse<Role>>('/roles', data),
  update: (id: string, data: any) =>
    client.put<APIResponse<Role>>(`/roles/${id}`, data),
  delete: (id: string) =>
    client.delete<APIResponse>(`/roles/${id}`),
  permissions: () =>
    client.get<APIResponse<string[]>>('/roles/permissions'),
};

// Tenants
export const tenantAPI = {
  list: (params?: Record<string, any>) =>
    client.get<APIResponse<PaginatedResult<Tenant>>>('/tenants', { params }),
  create: (data: any) =>
    client.post<APIResponse<Tenant>>('/tenants', data),
  update: (id: string, data: any) =>
    client.put<APIResponse<Tenant>>(`/tenants/${id}`, data),
  delete: (id: string) =>
    client.delete<APIResponse>(`/tenants/${id}`),
  settings: (id: string) =>
    client.get<APIResponse>(`/tenants/${id}/settings`),
};

// OS Profiles
export const osProfileAPI = {
  list: () =>
    client.get<APIResponse<OSProfile[]>>('/os-profiles'),
  create: (data: any) =>
    client.post<APIResponse<OSProfile>>('/os-profiles', data),
  update: (id: string, data: any) =>
    client.put<APIResponse<OSProfile>>(`/os-profiles/${id}`, data),
  delete: (id: string) =>
    client.delete<APIResponse>(`/os-profiles/${id}`),
};

// Audit Logs
export const auditAPI = {
  list: (params?: Record<string, any>) =>
    client.get<APIResponse<PaginatedResult<AuditLog>>>('/audit-logs', { params }),
};

// Settings
export const settingsAPI = {
  get: () =>
    client.get<APIResponse>('/settings'),
  update: (data: any) =>
    client.put<APIResponse>('/settings', data),
};
