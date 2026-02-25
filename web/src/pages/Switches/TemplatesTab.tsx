import React, { useState } from 'react';
import { Card, Typography, Button, Space, Tag, Collapse } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { switchAPI } from '../../api';

const { Text } = Typography;

interface CommandTemplate {
  operation: string;
  description: string;
  template: string;
}

interface VendorTemplates {
  vendor: string;
  label: string;
  templates: CommandTemplate[];
}

const TemplatesTab: React.FC = () => {
  const [templates, setTemplates] = useState<VendorTemplates[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchTemplates = async () => {
    setLoading(true);
    try {
      const { data: resp } = await switchAPI.getCommandTemplates();
      if (resp.success) setTemplates(resp.data as VendorTemplates[] || []);
    } catch { /* */ }
    setLoading(false);
  };

  if (templates.length === 0 && !loading) {
    return (
      <Card>
        <Space direction="vertical" align="center" style={{ width: '100%', padding: 24 }}>
          <Text type="secondary">Click to load command templates for all supported switch vendors.</Text>
          <Button type="primary" icon={<ReloadOutlined />} onClick={fetchTemplates}>Load Templates</Button>
        </Space>
      </Card>
    );
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 12 }}>
        <Button icon={<ReloadOutlined />} onClick={fetchTemplates} loading={loading}>Refresh</Button>
      </div>
      <Collapse
        items={templates.map((vt) => ({
          key: vt.vendor,
          label: <Text strong>{vt.label}</Text>,
          extra: <Tag>{vt.templates.length} templates</Tag>,
          children: (
            <Collapse size="small"
              items={vt.templates.map((t) => ({
                key: t.operation,
                label: <Space><Tag color="blue">{t.operation}</Tag><Text type="secondary">{t.description}</Text></Space>,
                children: (
                  <pre style={{ background: '#f5f5f5', padding: 12, borderRadius: 4, margin: 0, fontSize: 13, lineHeight: 1.5, overflow: 'auto' }}>
                    {t.template}
                  </pre>
                ),
              }))}
            />
          ),
        }))}
      />
    </div>
  );
};

export default TemplatesTab;
