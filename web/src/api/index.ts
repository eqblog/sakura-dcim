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
  DiskLayout,
  Script,
  InstallTask,
  ReinstallRequest,
  Switch,
  SwitchPort,
  BandwidthSummary,
  AuditLog,
  IPPool,
  IPAddress,
  InventoryResult,
  DiscoverySession,
  DiscoveredServer,
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
    client.get<APIResponse<InventoryResult>>(`/servers/${id}/inventory`),
  inventoryScan: (id: string) =>
    client.post<APIResponse<InventoryResult>>(`/servers/${id}/inventory/scan`),
  reinstall: (id: string, data: any) =>
    client.post<APIResponse>(`/servers/${id}/reinstall`, data),
  reinstallStatus: (id: string) =>
    client.get<APIResponse>(`/servers/${id}/reinstall/status`),
  kvmStart: (id: string) =>
    client.post<APIResponse>(`/servers/${id}/kvm`),
  kvmStop: (id: string, sessionId: string) =>
    client.delete<APIResponse>(`/servers/${id}/kvm`, { params: { session: sessionId } }),
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
  children: (id: string) =>
    client.get<APIResponse<Tenant[]>>(`/tenants/${id}/children`),
  tree: (id: string) =>
    client.get<APIResponse>(`/tenants/${id}/tree`),
};

// OS Profiles
export const osProfileAPI = {
  list: (params?: Record<string, any>) =>
    client.get<APIResponse<OSProfile[]>>('/os-profiles', { params }),
  get: (id: string) =>
    client.get<APIResponse<OSProfile>>(`/os-profiles/${id}`),
  create: (data: any) =>
    client.post<APIResponse<OSProfile>>('/os-profiles', data),
  update: (id: string, data: any) =>
    client.put<APIResponse<OSProfile>>(`/os-profiles/${id}`, data),
  delete: (id: string) =>
    client.delete<APIResponse>(`/os-profiles/${id}`),
};

// Disk Layouts
export const diskLayoutAPI = {
  list: () =>
    client.get<APIResponse<DiskLayout[]>>('/disk-layouts'),
  get: (id: string) =>
    client.get<APIResponse<DiskLayout>>(`/disk-layouts/${id}`),
  create: (data: any) =>
    client.post<APIResponse<DiskLayout>>('/disk-layouts', data),
  update: (id: string, data: any) =>
    client.put<APIResponse<DiskLayout>>(`/disk-layouts/${id}`, data),
  delete: (id: string) =>
    client.delete<APIResponse>(`/disk-layouts/${id}`),
};

// Scripts
export const scriptAPI = {
  list: () =>
    client.get<APIResponse<Script[]>>('/scripts'),
  get: (id: string) =>
    client.get<APIResponse<Script>>(`/scripts/${id}`),
  create: (data: any) =>
    client.post<APIResponse<Script>>('/scripts', data),
  update: (id: string, data: any) =>
    client.put<APIResponse<Script>>(`/scripts/${id}`, data),
  delete: (id: string) =>
    client.delete<APIResponse>(`/scripts/${id}`),
};

// Reinstall
export const reinstallAPI = {
  start: (serverId: string, data: ReinstallRequest) =>
    client.post<APIResponse<InstallTask>>(`/servers/${serverId}/reinstall`, data),
  status: (serverId: string) =>
    client.get<APIResponse<InstallTask>>(`/servers/${serverId}/reinstall/status`),
};

// Switches
export const switchAPI = {
  list: () =>
    client.get<APIResponse<Switch[]>>('/switches'),
  get: (id: string) =>
    client.get<APIResponse<Switch>>(`/switches/${id}`),
  create: (data: any) =>
    client.post<APIResponse<Switch>>('/switches', data),
  update: (id: string, data: any) =>
    client.put<APIResponse<Switch>>(`/switches/${id}`, data),
  delete: (id: string) =>
    client.delete<APIResponse>(`/switches/${id}`),
  // Ports
  listPorts: (switchId: string) =>
    client.get<APIResponse<SwitchPort[]>>(`/switches/${switchId}/ports`),
  createPort: (switchId: string, data: any) =>
    client.post<APIResponse<SwitchPort>>(`/switches/${switchId}/ports`, data),
  updatePort: (switchId: string, portId: string, data: any) =>
    client.put<APIResponse<SwitchPort>>(`/switches/${switchId}/ports/${portId}`, data),
  deletePort: (switchId: string, portId: string) =>
    client.delete<APIResponse>(`/switches/${switchId}/ports/${portId}`),
  provisionPort: (switchId: string, portId: string) =>
    client.post<APIResponse>(`/switches/${switchId}/ports/${portId}/provision`),
  getPortStatus: (switchId: string, portId: string) =>
    client.get<APIResponse>(`/switches/${switchId}/ports/${portId}/status`),
};

// Bandwidth
export const bandwidthAPI = {
  getServerBandwidth: (serverId: string, period?: string) =>
    client.get<APIResponse<BandwidthSummary[]>>(`/servers/${serverId}/bandwidth`, { params: { period } }),
};

// IP Pools & Addresses
export const ipPoolAPI = {
  list: () =>
    client.get<APIResponse<IPPool[]>>('/ip-pools'),
  get: (id: string) =>
    client.get<APIResponse<IPPool>>(`/ip-pools/${id}`),
  create: (data: any) =>
    client.post<APIResponse<IPPool>>('/ip-pools', data),
  update: (id: string, data: any) =>
    client.put<APIResponse<IPPool>>(`/ip-pools/${id}`, data),
  delete: (id: string) =>
    client.delete<APIResponse>(`/ip-pools/${id}`),
  // Addresses
  listAddresses: (poolId: string) =>
    client.get<APIResponse<IPAddress[]>>(`/ip-pools/${poolId}/addresses`),
  createAddress: (poolId: string, data: any) =>
    client.post<APIResponse<IPAddress>>(`/ip-pools/${poolId}/addresses`, data),
  updateAddress: (poolId: string, addrId: string, data: any) =>
    client.put<APIResponse<IPAddress>>(`/ip-pools/${poolId}/addresses/${addrId}`, data),
  deleteAddress: (poolId: string, addrId: string) =>
    client.delete<APIResponse>(`/ip-pools/${poolId}/addresses/${addrId}`),
  assignNext: (poolId: string, serverId: string) =>
    client.post<APIResponse<IPAddress>>(`/ip-pools/${poolId}/assign`, { server_id: serverId }),
};

// Audit Logs
export const auditAPI = {
  list: (params?: Record<string, any>) =>
    client.get<APIResponse<PaginatedResult<AuditLog>>>('/audit-logs', { params }),
};

// Discovery
export const discoveryAPI = {
  start: (agentId: string, data: { dhcp_range_start: string; dhcp_range_end: string; gateway: string; netmask: string }) =>
    client.post<APIResponse<DiscoverySession>>(`/agents/${agentId}/discovery/start`, data),
  stop: (agentId: string) =>
    client.post<APIResponse>(`/agents/${agentId}/discovery/stop`),
  status: (agentId: string) =>
    client.get<APIResponse>(`/agents/${agentId}/discovery/status`),
  listServers: (params?: Record<string, any>) =>
    client.get<APIResponse<PaginatedResult<DiscoveredServer>>>('/discovery/servers', { params }),
  getServer: (id: string) =>
    client.get<APIResponse<DiscoveredServer>>(`/discovery/servers/${id}`),
  approve: (id: string, data: any) =>
    client.post<APIResponse>(`/discovery/servers/${id}/approve`, data),
  reject: (id: string) =>
    client.post<APIResponse>(`/discovery/servers/${id}/reject`),
  deleteServer: (id: string) =>
    client.delete<APIResponse>(`/discovery/servers/${id}`),
};

// Settings
export const settingsAPI = {
  get: () =>
    client.get<APIResponse>('/settings'),
  update: (data: any) =>
    client.put<APIResponse>('/settings', data),
};
