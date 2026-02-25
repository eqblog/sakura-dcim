import React, { useEffect, useState, useCallback } from 'react';
import { Card, Typography, Tree, Table, Button, Space, Tag, Spin, Descriptions } from 'antd';
import { ApartmentOutlined, ReloadOutlined } from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import { tenantAPI, serverAPI } from '../../api';
import type { Tenant, Server, PaginatedResult } from '../../types';
import { useAuthStore } from '../../store/auth';

const { Title, Text } = Typography;

interface TenantTree extends Tenant {
  children?: TenantTree[];
}

const ResellerDashboard: React.FC = () => {
  const { user } = useAuthStore();
  const [tree, setTree] = useState<TenantTree | null>(null);
  const [loading, setLoading] = useState(false);
  const [selectedTenant, setSelectedTenant] = useState<Tenant | null>(null);
  const [servers, setServers] = useState<PaginatedResult<Server> | null>(null);
  const [serverLoading, setServerLoading] = useState(false);

  const fetchTree = useCallback(async () => {
    if (!user?.tenant_id) return;
    setLoading(true);
    try {
      const { data: resp } = await tenantAPI.list({ parent_id: user.tenant_id });
      if (resp.success) {
        setTree(resp.data as any);
      }
    } catch { /* ignore */ }
    setLoading(false);
  }, [user?.tenant_id]);

  useEffect(() => {
    fetchTree();
  }, [fetchTree]);

  const fetchTenantServers = async (tenantId: string) => {
    setServerLoading(true);
    try {
      const { data: resp } = await serverAPI.list({ tenant_id: tenantId, page_size: 50 });
      if (resp.success) {
        setServers(resp.data || null);
      }
    } catch { /* ignore */ }
    setServerLoading(false);
  };

  const convertToTreeData = (node: TenantTree): DataNode => ({
    key: node.id,
    title: (
      <Space size={4}>
        <Text strong>{node.name}</Text>
        <Tag>{node.slug}</Tag>
      </Space>
    ),
    children: node.children?.map(convertToTreeData),
  });

  const handleSelectTenant = async (keys: React.Key[]) => {
    if (keys.length === 0) return;
    const tenantId = keys[0] as string;
    try {
      const { data: resp } = await tenantAPI.list({});
      if (resp.success && resp.data?.items) {
        const t = resp.data.items.find((t: Tenant) => t.id === tenantId);
        if (t) {
          setSelectedTenant(t);
          fetchTenantServers(tenantId);
        }
      }
    } catch { /* ignore */ }
  };

  const serverColumns = [
    { title: 'Hostname', dataIndex: 'hostname', key: 'hostname' },
    { title: 'Primary IP', dataIndex: 'primary_ip', key: 'primary_ip', render: (ip: string) => <Text code>{ip}</Text> },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (s: string) => {
        const colors: Record<string, string> = { active: 'green', offline: 'default', error: 'red', provisioning: 'blue' };
        return <Tag color={colors[s] || 'default'}>{s}</Tag>;
      },
    },
    { title: 'CPU', dataIndex: 'cpu_model', key: 'cpu', ellipsis: true },
    { title: 'RAM', dataIndex: 'ram_mb', key: 'ram', render: (v: number) => v ? `${(v / 1024).toFixed(0)} GB` : '-' },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>
          <ApartmentOutlined style={{ marginRight: 8 }} />
          Reseller Dashboard
        </Title>
        <Button icon={<ReloadOutlined />} onClick={fetchTree}>Refresh</Button>
      </div>

      <div style={{ display: 'flex', gap: 16 }}>
        <Card title="Tenant Hierarchy" style={{ width: 360, flexShrink: 0 }}>
          {loading ? (
            <Spin />
          ) : tree ? (
            <Tree
              treeData={[convertToTreeData(tree)]}
              defaultExpandAll
              onSelect={handleSelectTenant}
              showLine
            />
          ) : (
            <Text type="secondary">No sub-tenants found</Text>
          )}
        </Card>

        <div style={{ flex: 1 }}>
          {selectedTenant ? (
            <Card>
              <Descriptions title={selectedTenant.name} size="small" column={2} style={{ marginBottom: 16 }}>
                <Descriptions.Item label="Slug">{selectedTenant.slug}</Descriptions.Item>
                <Descriptions.Item label="Domain">{selectedTenant.custom_domain || '-'}</Descriptions.Item>
                <Descriptions.Item label="Color">
                  {selectedTenant.primary_color ? (
                    <Space>
                      <div style={{ width: 16, height: 16, borderRadius: 4, background: selectedTenant.primary_color }} />
                      {selectedTenant.primary_color}
                    </Space>
                  ) : '-'}
                </Descriptions.Item>
                <Descriptions.Item label="Created">{new Date(selectedTenant.created_at).toLocaleDateString()}</Descriptions.Item>
              </Descriptions>

              <Title level={5}>Servers ({servers?.total || 0})</Title>
              <Table
                dataSource={servers?.items || []}
                columns={serverColumns}
                rowKey="id"
                loading={serverLoading}
                pagination={false}
                size="small"
              />
            </Card>
          ) : (
            <Card>
              <div style={{ textAlign: 'center', padding: 40 }}>
                <Text type="secondary">Select a tenant from the tree to view details</Text>
              </div>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
};

export default ResellerDashboard;
