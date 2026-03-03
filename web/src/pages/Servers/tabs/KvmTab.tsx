import React, { useRef, useState, useCallback, useEffect } from 'react';
import { Button, Space, Alert, Spin, Tooltip, Input, Tag, Typography, Card } from 'antd';
import {
  DesktopOutlined,
  FullscreenOutlined,
  FullscreenExitOutlined,
  PoweroffOutlined,
  LoadingOutlined,
  SendOutlined,
  CodeOutlined,
  LinkOutlined,
} from '@ant-design/icons';
import { serverAPI } from '../../../api';
import type { Server } from '../../../types';
import { useAuthStore } from '../../../store/auth';
import KvmCredentialsCard from './KvmCredentialsCard';

// @ts-expect-error noVNC has no type declarations
import RFB from '@novnc/novnc/lib/rfb';

const { Text } = Typography;

const sendTextToVnc = (rfb: any, text: string) => {
  for (const char of text) rfb.sendKey(char.charCodeAt(0));
};

interface KvmTabProps {
  server: Server;
}

type KvmStatus = 'idle' | 'starting' | 'connecting' | 'connected' | 'error';

const KvmTab: React.FC<KvmTabProps> = ({ server }) => {
  const serverId = server.id;
  const { user } = useAuthStore();
  const kvmMode = user?.tenant?.kvm_mode || 'webkvm';
  const canvasRef = useRef<HTMLDivElement>(null);
  const rfbRef = useRef<any>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [status, setStatus] = useState<KvmStatus>('idle');
  const [error, setError] = useState<string>('');
  const [sessionId, setSessionId] = useState<string>('');
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [tempUser, setTempUser] = useState<string>('');
  const [tempPass, setTempPass] = useState<string>('');
  const [showPass, setShowPass] = useState(false);
  const [commandText, setCommandText] = useState('');
  const [consoleUrl, setConsoleUrl] = useState<string>('');

  const cleanup = useCallback(() => {
    if (rfbRef.current) { rfbRef.current.disconnect(); rfbRef.current = null; }
    if (canvasRef.current) canvasRef.current.innerHTML = '';
  }, []);

  // ── Web KVM mode: start Docker→VNC→noVNC pipeline ──
  const startWebKvm = useCallback(async () => {
    setError(''); setStatus('starting');
    setTempUser(''); setTempPass(''); setShowPass(false);
    try {
      const { data: resp } = await serverAPI.kvmStart(serverId);
      if (!resp.success || !resp.data) throw new Error(resp.error || 'Failed to start KVM session');
      const data = resp.data as { session_id: string; temp_user?: string; temp_pass?: string };
      setSessionId(data.session_id);
      if (data.temp_user) { setTempUser(data.temp_user); setTempPass(data.temp_pass || ''); }
      setStatus('connecting');
      const wsProtocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
      const wsUrl = `${wsProtocol}://${window.location.host}/api/v1/kvm/ws?session=${data.session_id}`;
      await new Promise(r => setTimeout(r, 500));
      if (!canvasRef.current) return;
      const rfb = new RFB(canvasRef.current, wsUrl, { scaleViewport: true, resizeSession: false });
      rfb.showDotCursor = true;
      rfb.addEventListener('connect', () => setStatus('connected'));
      rfb.addEventListener('disconnect', (e: any) => {
        setStatus(e.detail.clean ? 'idle' : 'error');
        if (!e.detail.clean) setError('KVM connection lost');
        rfbRef.current = null;
      });
      rfb.addEventListener('securityfailure', (e: any) => {
        setStatus('error'); setError('VNC security error: ' + (e.detail.reason || 'unknown'));
      });
      rfbRef.current = rfb;
    } catch (err: any) {
      setStatus('error'); setError(err.message || 'Failed to start KVM'); cleanup();
    }
  }, [serverId, cleanup]);

  // ── Direct Console mode: get console URL and open in new tab ──
  const startDirectConsole = useCallback(async () => {
    setError(''); setStatus('starting');
    setTempUser(''); setTempPass(''); setShowPass(false);
    setConsoleUrl('');
    try {
      const { data: resp } = await serverAPI.kvmStart(serverId);
      if (!resp.success || !resp.data) throw new Error(resp.error || 'Failed to start console session');
      const data = resp.data as {
        session_id: string;
        temp_user?: string;
        temp_pass?: string;
        console_url?: string;
        direct_console?: boolean;
      };
      setSessionId(data.session_id);
      if (data.temp_user) { setTempUser(data.temp_user); setTempPass(data.temp_pass || ''); }
      if (data.console_url) {
        setConsoleUrl(data.console_url);
        window.open(data.console_url, '_blank', 'noopener,noreferrer');
      }
      setStatus('connected');
    } catch (err: any) {
      setStatus('error'); setError(err.message || 'Failed to start console'); cleanup();
    }
  }, [serverId, cleanup]);

  const startKvm = kvmMode === 'vconsole' ? startDirectConsole : startWebKvm;

  const stopKvm = useCallback(async () => {
    cleanup();
    if (sessionId) { try { await serverAPI.kvmStop(serverId, sessionId); } catch { /* ignore */ } }
    setSessionId(''); setTempUser(''); setTempPass('');
    setConsoleUrl(''); setStatus('idle');
  }, [serverId, sessionId, cleanup]);

  const toggleFullscreen = useCallback(() => {
    if (!containerRef.current) return;
    if (!document.fullscreenElement) { containerRef.current.requestFullscreen(); setIsFullscreen(true); }
    else { document.exitFullscreen(); setIsFullscreen(false); }
  }, []);

  const handleSendCommand = useCallback(() => {
    if (!rfbRef.current || !commandText) return;
    sendTextToVnc(rfbRef.current, commandText);
    rfbRef.current.sendKey(0xFF0D); // Enter
    setCommandText('');
  }, [commandText]);

  const handleSendCtrlAltDel = useCallback(() => {
    if (rfbRef.current) rfbRef.current.sendCtrlAltDel();
  }, []);

  useEffect(() => {
    const handler = () => setIsFullscreen(!!document.fullscreenElement);
    document.addEventListener('fullscreenchange', handler);
    return () => { document.removeEventListener('fullscreenchange', handler); cleanup(); };
  }, [cleanup]);

  const isIdle = status === 'idle' || status === 'error';
  const isDirectMode = kvmMode === 'vconsole';

  return (
    <div ref={containerRef}>
      {isIdle && (
        <div style={{ marginBottom: 12, display: 'flex', alignItems: 'center', gap: 8 }}>
          <Text type="secondary">Mode:</Text>
          <Tag color={isDirectMode ? 'green' : 'blue'}>
            {isDirectMode ? 'Direct Console' : 'Web KVM'}
          </Tag>
          <Text type="secondary" style={{ fontSize: 12 }}>
            {isDirectMode
              ? 'Opens BMC virtual console directly in a new browser tab'
              : 'Opens BMC web UI via embedded VNC viewer'}
            &nbsp;&mdash; set by admin in Settings
          </Text>
        </div>
      )}

      <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Space wrap>
          {isIdle ? (
            <Button type="primary" icon={isDirectMode ? <LinkOutlined /> : <DesktopOutlined />} onClick={startKvm}>
              {isDirectMode ? 'Open BMC Console' : 'Open KVM Console'}
            </Button>
          ) : status === 'starting' ? (
            <Button disabled icon={<LoadingOutlined />}>Starting...</Button>
          ) : status === 'connecting' ? (
            <Button disabled icon={<LoadingOutlined />}>Connecting...</Button>
          ) : (
            <Button danger icon={<PoweroffOutlined />} onClick={stopKvm}>
              {isDirectMode ? 'End Session' : 'Disconnect'}
            </Button>
          )}
        </Space>
        {/* Web KVM only controls */}
        {!isDirectMode && (
          <Space>
            {status === 'connected' && (
              <Tooltip title="Send Ctrl+Alt+Del"><Button onClick={handleSendCtrlAltDel}>Ctrl+Alt+Del</Button></Tooltip>
            )}
            {status === 'connected' && (
              <Tooltip title={isFullscreen ? 'Exit Fullscreen' : 'Fullscreen'}>
                <Button icon={isFullscreen ? <FullscreenExitOutlined /> : <FullscreenOutlined />} onClick={toggleFullscreen} />
              </Tooltip>
            )}
          </Space>
        )}
      </div>

      {/* Web KVM: command input bar */}
      {!isDirectMode && status === 'connected' && (
        <div style={{ marginBottom: 12, display: 'flex', gap: 8 }}>
          <Input
            placeholder="Type command to send to KVM console..."
            value={commandText}
            onChange={(e) => setCommandText(e.target.value)}
            onPressEnter={handleSendCommand}
            prefix={<CodeOutlined />}
            style={{ flex: 1 }}
          />
          <Button type="primary" icon={<SendOutlined />} onClick={handleSendCommand} disabled={!commandText}>
            Send
          </Button>
        </div>
      )}

      {error && <Alert type="error" message={error} closable onClose={() => setError('')} style={{ marginBottom: 12 }} />}

      {/* Direct Console: info card when session active */}
      {isDirectMode && status === 'connected' && (
        <Card size="small" style={{ marginBottom: 12 }}>
          <Alert
            type="info"
            showIcon
            message="BMC virtual console opened in a new browser tab"
            description={
              <div>
                <p style={{ margin: '8px 0 4px' }}>
                  The BMC virtual console page has been opened. Log in with the credentials below to access KVM.
                </p>
                {consoleUrl && (
                  <p style={{ margin: '4px 0' }}>
                    <Text type="secondary">URL: </Text>
                    <a href={consoleUrl} target="_blank" rel="noopener noreferrer">{consoleUrl}</a>
                  </p>
                )}
                <p style={{ margin: '4px 0 0', fontSize: 12, color: '#888' }}>
                  Click "End Session" when done to clean up the temporary BMC user.
                </p>
              </div>
            }
          />
        </Card>
      )}

      {/* Temp credentials (both modes) */}
      {tempUser && (status === 'connecting' || status === 'connected') && (
        <KvmCredentialsCard
          tempUser={tempUser}
          tempPass={tempPass}
          showPass={showPass}
          onShowPassChange={setShowPass}
          rfb={isDirectMode ? null : rfbRef.current}
          connected={!isDirectMode && status === 'connected'}
        />
      )}

      {/* Web KVM: loading spinner */}
      {!isDirectMode && (status === 'starting' || status === 'connecting') && (
        <div style={{ textAlign: 'center', padding: 60 }}>
          <Spin size="large" />
        </div>
      )}

      {/* Direct Console: starting spinner */}
      {isDirectMode && status === 'starting' && (
        <div style={{ textAlign: 'center', padding: 60 }}>
          <Spin size="large" tip="Creating temporary BMC user..." />
        </div>
      )}

      {/* Web KVM: noVNC canvas */}
      {!isDirectMode && (
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
      )}
    </div>
  );
};

export default KvmTab;
