import React from 'react';
import { Button, Card, Space, Tooltip, Typography, Input } from 'antd';
import {
  KeyOutlined,
  EyeOutlined,
  EyeInvisibleOutlined,
  SendOutlined,
  LoginOutlined,
} from '@ant-design/icons';

const { Text } = Typography;

const XK_Tab = 0xFF09;
const XK_Return = 0xFF0D;

interface KvmCredentialsCardProps {
  tempUser: string;
  tempPass: string;
  showPass: boolean;
  onShowPassChange: (visible: boolean) => void;
  rfb: any;
  connected: boolean;
}

const sendTextToVnc = (rfb: any, text: string) => {
  for (const char of text) rfb.sendKey(char.charCodeAt(0));
};

const KvmCredentialsCard: React.FC<KvmCredentialsCardProps> = ({
  tempUser, tempPass, showPass, onShowPassChange, rfb, connected,
}) => {
  const handleSendUsername = () => {
    if (rfb && tempUser) sendTextToVnc(rfb, tempUser);
  };

  const handleSendPassword = () => {
    if (rfb && tempPass) sendTextToVnc(rfb, tempPass);
  };

  const handleAutoLogin = () => {
    if (!rfb || !tempUser || !tempPass) return;
    sendTextToVnc(rfb, tempUser);
    setTimeout(() => {
      rfb.sendKey(XK_Tab);
      setTimeout(() => {
        sendTextToVnc(rfb, tempPass);
        setTimeout(() => rfb.sendKey(XK_Return), 100);
      }, 100);
    }, 100);
  };

  return (
    <Card
      size="small"
      style={{ marginBottom: 12 }}
      title={<><KeyOutlined style={{ marginRight: 8 }} />BMC Login Credentials</>}
    >
      <Space direction="vertical" style={{ width: '100%' }} size={4}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <Text type="secondary" style={{ width: 80, flexShrink: 0 }}>Username:</Text>
          <Text copyable strong>{tempUser}</Text>
          {connected && (
            <Tooltip title="Type username into VNC">
              <Button size="small" icon={<SendOutlined />} onClick={handleSendUsername}>Send</Button>
            </Tooltip>
          )}
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <Text type="secondary" style={{ width: 80, flexShrink: 0 }}>Password:</Text>
          <Input.Password
            value={tempPass}
            readOnly
            size="small"
            style={{ width: 220 }}
            visibilityToggle={{ visible: showPass, onVisibleChange: onShowPassChange }}
            iconRender={(visible) => visible ? <EyeOutlined /> : <EyeInvisibleOutlined />}
          />
          <Text copyable={{ text: tempPass }} />
          {connected && (
            <Tooltip title="Type password into VNC">
              <Button size="small" icon={<SendOutlined />} onClick={handleSendPassword}>Send</Button>
            </Tooltip>
          )}
        </div>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Text type="warning" style={{ fontSize: 12 }}>
            Temporary account — automatically revoked on disconnect
          </Text>
          {connected && (
            <Tooltip title="Send username + Tab + password + Enter into VNC">
              <Button size="small" type="primary" icon={<LoginOutlined />} onClick={handleAutoLogin}>
                Auto Login
              </Button>
            </Tooltip>
          )}
        </div>
      </Space>
    </Card>
  );
};

export default KvmCredentialsCard;
