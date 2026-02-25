import React, { useRef, useState, useCallback, useEffect } from 'react';
import { Button, Space, Alert, Spin, Tooltip, Card, Typography, Input } from 'antd';
import {
  DesktopOutlined,
  FullscreenOutlined,
  FullscreenExitOutlined,
  PoweroffOutlined,
  LoadingOutlined,
  KeyOutlined,
  EyeOutlined,
  EyeInvisibleOutlined,
  SendOutlined,
  LoginOutlined,
} from '@ant-design/icons';
import { serverAPI } from '../../../api';

// @ts-expect-error noVNC has no type declarations
import RFB from '@novnc/novnc/lib/rfb';

const { Text } = Typography;

// X11 keysym constants for special keys
const XK_Tab = 0xFF09;
const XK_Return = 0xFF0D;

/** Send a text string to VNC character-by-character via sendKey (press+release). */
const sendTextToVnc = (rfb: any, text: string) => {
  for (const char of text) {
    rfb.sendKey(char.charCodeAt(0));
  }
};

/** Send a single special key to VNC. */
const sendKeyToVnc = (rfb: any, keysym: number) => {
  rfb.sendKey(keysym);
};

interface KvmTabProps {
  serverId: string;
}

type KvmStatus = 'idle' | 'starting' | 'connecting' | 'connected' | 'error';

const KvmTab: React.FC<KvmTabProps> = ({ serverId }) => {
  const canvasRef = useRef<HTMLDivElement>(null);
  const rfbRef = useRef<any>(null);
  const [status, setStatus] = useState<KvmStatus>('idle');
  const [error, setError] = useState<string>('');
  const [sessionId, setSessionId] = useState<string>('');
  const [isFullscreen, setIsFullscreen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const [tempUser, setTempUser] = useState<string>('');
  const [tempPass, setTempPass] = useState<string>('');
  const [showPass, setShowPass] = useState(false);

  const cleanup = useCallback(() => {
    if (rfbRef.current) {
      rfbRef.current.disconnect();
      rfbRef.current = null;
    }
    if (canvasRef.current) {
      canvasRef.current.innerHTML = '';
    }
  }, []);

  const startKvm = useCallback(async () => {
    setError('');
    setStatus('starting');
    setTempUser('');
    setTempPass('');
    setShowPass(false);

    try {
      const { data: resp } = await serverAPI.kvmStart(serverId);
      if (!resp.success || !resp.data) {
        throw new Error(resp.error || 'Failed to start KVM session');
      }

      const data = resp.data as { session_id: string; temp_user?: string; temp_pass?: string };
      setSessionId(data.session_id);
      if (data.temp_user) {
        setTempUser(data.temp_user);
        setTempPass(data.temp_pass || '');
      }
      setStatus('connecting');

      // Build WS URL from current browser location (avoids proxy host mismatch)
      const wsProtocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
      const wsUrl = `${wsProtocol}://${window.location.host}/api/v1/kvm/ws?session=${data.session_id}`;

      // Wait briefly for agent relay to establish
      await new Promise(r => setTimeout(r, 2000));

      if (!canvasRef.current) return;

      const rfb = new RFB(canvasRef.current, wsUrl, {
        scaleViewport: true,
        resizeSession: false,
      });
      rfb.showDotCursor = true;

      rfb.addEventListener('connect', () => {
        setStatus('connected');
      });

      rfb.addEventListener('disconnect', (e: any) => {
        setStatus(e.detail.clean ? 'idle' : 'error');
        if (!e.detail.clean) {
          setError('KVM connection lost');
        }
        rfbRef.current = null;
      });

      rfb.addEventListener('securityfailure', (e: any) => {
        setStatus('error');
        setError('VNC security error: ' + (e.detail.reason || 'unknown'));
      });

      rfbRef.current = rfb;
    } catch (err: any) {
      setStatus('error');
      setError(err.message || 'Failed to start KVM');
      cleanup();
    }
  }, [serverId, cleanup]);

  const stopKvm = useCallback(async () => {
    cleanup();
    if (sessionId) {
      try {
        await serverAPI.kvmStop(serverId, sessionId);
      } catch {
        // Ignore stop errors
      }
    }
    setSessionId('');
    setTempUser('');
    setTempPass('');
    setStatus('idle');
  }, [serverId, sessionId, cleanup]);

  const toggleFullscreen = useCallback(() => {
    if (!containerRef.current) return;
    if (!document.fullscreenElement) {
      containerRef.current.requestFullscreen();
      setIsFullscreen(true);
    } else {
      document.exitFullscreen();
      setIsFullscreen(false);
    }
  }, []);

  const handleSendUsername = useCallback(() => {
    if (rfbRef.current && tempUser) {
      sendTextToVnc(rfbRef.current, tempUser);
    }
  }, [tempUser]);

  const handleSendPassword = useCallback(() => {
    if (rfbRef.current && tempPass) {
      sendTextToVnc(rfbRef.current, tempPass);
    }
  }, [tempPass]);

  const handleAutoLogin = useCallback(() => {
    if (!rfbRef.current || !tempUser || !tempPass) return;
    const rfb = rfbRef.current;
    sendTextToVnc(rfb, tempUser);
    setTimeout(() => {
      sendKeyToVnc(rfb, XK_Tab);
      setTimeout(() => {
        sendTextToVnc(rfb, tempPass);
        setTimeout(() => {
          sendKeyToVnc(rfb, XK_Return);
        }, 100);
      }, 100);
    }, 100);
  }, [tempUser, tempPass]);

  useEffect(() => {
    const handler = () => setIsFullscreen(!!document.fullscreenElement);
    document.addEventListener('fullscreenchange', handler);
    return () => {
      document.removeEventListener('fullscreenchange', handler);
      cleanup();
    };
  }, [cleanup]);

  return (
    <div ref={containerRef}>
      <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Space>
          {status === 'idle' || status === 'error' ? (
            <Button type="primary" icon={<DesktopOutlined />} onClick={startKvm}>
              Open KVM Console
            </Button>
          ) : status === 'starting' || status === 'connecting' ? (
            <Button disabled icon={<LoadingOutlined />}>
              {status === 'starting' ? 'Starting...' : 'Connecting...'}
            </Button>
          ) : (
            <Button danger icon={<PoweroffOutlined />} onClick={stopKvm}>
              Disconnect
            </Button>
          )}
        </Space>
        {status === 'connected' && (
          <Tooltip title={isFullscreen ? 'Exit Fullscreen' : 'Fullscreen'}>
            <Button
              icon={isFullscreen ? <FullscreenExitOutlined /> : <FullscreenOutlined />}
              onClick={toggleFullscreen}
            />
          </Tooltip>
        )}
      </div>

      {error && (
        <Alert type="error" title={error} closable onClose={() => setError('')} style={{ marginBottom: 12 }} />
      )}

      {tempUser && (status === 'connecting' || status === 'connected') && (
        <Card
          size="small"
          style={{ marginBottom: 12 }}
          title={<><KeyOutlined style={{ marginRight: 8 }} />BMC Login Credentials</>}
        >
          <Space direction="vertical" style={{ width: '100%' }} size={4}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <Text type="secondary" style={{ width: 80, flexShrink: 0 }}>Username:</Text>
              <Text copyable strong>{tempUser}</Text>
              {status === 'connected' && (
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
                visibilityToggle={{ visible: showPass, onVisibleChange: setShowPass }}
                iconRender={(visible) => visible ? <EyeOutlined /> : <EyeInvisibleOutlined />}
              />
              <Text copyable={{ text: tempPass }} />
              {status === 'connected' && (
                <Tooltip title="Type password into VNC">
                  <Button size="small" icon={<SendOutlined />} onClick={handleSendPassword}>Send</Button>
                </Tooltip>
              )}
            </div>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <Text type="warning" style={{ fontSize: 12 }}>
                Temporary account — automatically revoked on disconnect
              </Text>
              {status === 'connected' && (
                <Tooltip title="Send username + Tab + password + Enter into VNC">
                  <Button size="small" type="primary" icon={<LoginOutlined />} onClick={handleAutoLogin}>
                    Auto Login
                  </Button>
                </Tooltip>
              )}
            </div>
          </Space>
        </Card>
      )}

      {(status === 'starting' || status === 'connecting') && (
        <div style={{ textAlign: 'center', padding: 60 }}>
          <Spin size="large" description={status === 'starting' ? 'Starting KVM container...' : 'Connecting to VNC...'} />
        </div>
      )}

      <div
        ref={canvasRef}
        style={{
          width: '100%',
          minHeight: status === 'connected' ? 600 : 0,
          background: status === 'connected' ? '#000' : 'transparent',
          borderRadius: 4,
          overflow: 'hidden',
        }}
      />
    </div>
  );
};

export default KvmTab;
