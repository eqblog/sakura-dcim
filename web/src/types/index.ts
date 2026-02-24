// API response wrapper
export interface APIResponse<T = any> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
}

export interface PaginatedResult<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

// Auth
export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  user: User;
}

// Tenant
export interface Tenant {
  id: string;
  parent_id?: string;
  name: string;
  slug: string;
  custom_domain?: string;
  logo_url?: string;
  primary_color?: string;
  favicon_url?: string;
  created_at: string;
  updated_at: string;
}

// User
export interface User {
  id: string;
  tenant_id: string;
  email: string;
  name: string;
  role_id?: string;
  is_active: boolean;
  last_login?: string;
  created_at: string;
  role?: Role;
  tenant?: Tenant;
}

// Role
export interface Role {
  id: string;
  tenant_id?: string;
  name: string;
  permissions: string[];
  is_system: boolean;
  created_at: string;
}

// Agent
export interface Agent {
  id: string;
  name: string;
  location: string;
  status: 'online' | 'offline' | 'error';
  last_seen?: string;
  version: string;
  capabilities: string[];
  created_at: string;
}

// Server
export type ServerStatus = 'active' | 'provisioning' | 'reinstalling' | 'offline' | 'error';

export interface Server {
  id: string;
  tenant_id?: string;
  agent_id?: string;
  hostname: string;
  label: string;
  status: ServerStatus;
  primary_ip: string;
  ipmi_ip: string;
  cpu_model: string;
  cpu_cores: number;
  ram_mb: number;
  tags: string[];
  notes: string;
  created_at: string;
  updated_at: string;
  agent?: Agent;
}

export interface ServerCreateRequest {
  agent_id?: string;
  hostname: string;
  label?: string;
  primary_ip?: string;
  ipmi_ip?: string;
  ipmi_user?: string;
  ipmi_pass?: string;
  tags?: string[];
  notes?: string;
}

// OS Profile
export interface OSProfile {
  id: string;
  name: string;
  os_family: string;
  version: string;
  arch: string;
  kernel_url: string;
  initrd_url: string;
  boot_args: string;
  template_type: string;
  template: string;
  is_active: boolean;
  tags: string[];
  created_at: string;
}

// Install Task
export type InstallTaskStatus = 'pending' | 'pxe_booting' | 'installing' | 'post_scripts' | 'completed' | 'failed';

export interface InstallTask {
  id: string;
  server_id: string;
  os_profile_id: string;
  disk_layout_id?: string;
  raid_level: string;
  status: InstallTaskStatus;
  ssh_keys: string[];
  progress: number;
  log: string;
  started_at?: string;
  completed_at?: string;
  created_at: string;
}

// Audit Log
export interface AuditLog {
  id: string;
  tenant_id?: string;
  user_id?: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  details: any;
  ip_address: string;
  user_agent: string;
  created_at: string;
}

// Switch & Bandwidth
export interface Switch {
  id: string;
  agent_id: string;
  name: string;
  ip: string;
  snmp_community: string;
  snmp_version: string;
  created_at: string;
}

export interface SwitchPort {
  id: string;
  switch_id: string;
  server_id?: string;
  port_index: number;
  port_name: string;
  speed_mbps: number;
  description: string;
}

// IP Management
export interface IPPool {
  id: string;
  tenant_id?: string;
  network: string;
  gateway: string;
  description: string;
}

export interface IPAddress {
  id: string;
  pool_id: string;
  address: string;
  server_id?: string;
  status: 'available' | 'assigned' | 'reserved';
  note: string;
}
