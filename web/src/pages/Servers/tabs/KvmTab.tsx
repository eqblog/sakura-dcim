import React, { useRef, useState, useCallback, useEffect } from 'react';
import { Button, Space, Alert, Spin, Tooltip, Input } from 'antd';
import {
  DesktopOutlined,
  FullscreenOutlined,
  FullscreenExitOutlined,
  PoweroffOutlined,
  LoadingOutlined,
  SendOutlined,
  CodeOutlined,
} from '@ant-design/icons';
import { serverAPI } from '../../../api';
import type { Server } from '../../../types';
import KvmCredentialsCard from './KvmCredentialsCard';

// @ts-expect-error noVNC has no type declarations
import RFB from '@novnc/novnc/lib/rfb';

const KVM_MODE_KEY = 'sakura_kvm_mode';
type KvmMode = 'webkvm' | 'vconsole';
const getKvmMode = (): KvmMode => (localStorage.getItem(KVM_MODE_KEY) as KvmMode) || 'webkvm';

const sendTextToVnc = (rfb: any, text: string) => {
  for (const char of text) rfb.sendKey(char.charCodeAt(0));
};

interface KvmTabProps {
  server: Server;
}

type KvmStatus = 'idle' | 'starting' | 'connecting' | 'connected' | 'error';

const KvmTab: React.FC<KvmTabProps> = ({ server }) => {
  const serverId = server.id;
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

  const cleanup = useCallback(() => {
    if (rfbRef.current) { rfbRef.current.disconnect(); rfbRef.current = null; }
    if (canvasRef.current) canvasRef.current.innerHTML = '';
  }, []);

  const startKvm = useCallback(async () => {
    setError(''); setStatus('starting');
    setTempUser(''); setTempPass(''); setShowPass(false);
    const directConsole = getKvmMode() === 'vconsole';
    try {
      const { data: resp } = await serverAPI.kvmStart(serverId, directConsole);
      if (!resp.success || !resp.data) throw new Error(resp.error || 'Failed to start KVM session');
      const data = resp.data as { session_id: string; temp_user?: string; temp_pass?: string };
      setSessionId(data.session_id);
      if (data.temp_user) { setTempUser(data.temp_user); setTempPass(data.temp_pass || ''); }
      setStatus('connecting');
      const wsProtocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
      const wsUrl = `${wsProtocol}://${window.location.host}/api/v1/kvm/ws?session=${data.session_id}`;
      await new Promise(r => setTimeout(r, 2000));
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

  const stopKvm = useCallback(async () => {
    cleanup();
    if (sessionId) { try { await serverAPI.kvmStop(serverId, sessionId); } catch { /* ignore */ } }
    setSessionId(''); setTempUser(''); setTempPass(''); setStatus('idle');
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

  const kvmMode = getKvmMode();

  return (
    <div ref={containerRef}>
      <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Space wrap>
          {status === 'idle' || status === 'error' ? (
            <Button type="primary" icon={<DesktopOutlined />} onClick={startKvm}>
              {kvmMode === 'vconsole' ? 'Open vConsole' : 'Open KVM Console'}
            </Button>
          ) : status === 'starting' || status === 'connecting' ? (
            <Button disabled icon={<LoadingOutlined />}>
              {status === 'starting' ? 'Starting...' : 'Connecting...'}
            </Button>
          ) : (
            <Button danger icon={<PoweroffOutlined />} onClick={stopKvm}>Disconnect</Button>
          )}
        </Space>
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
      </div>

      {status === 'connected' && (
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

      {tempUser && (status === 'connecting' || status === 'connected') && (
        <KvmCredentialsCard
          tempUser={tempUser}
          tempPass={tempPass}
          showPass={showPass}
          onShowPassChange={setShowPass}
          rfb={rfbRef.current}
          connected={status === 'connected'}
        />
      )}

      {(status === 'starting' || status === 'connecting') && (
        <div style={{ textAlign: 'center', padding: 60 }}>
          <Spin size="large" />
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
