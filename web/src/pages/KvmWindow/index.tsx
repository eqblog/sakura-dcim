import React, { useRef, useState, useCallback, useEffect } from 'react';
import { Button, Spin, Alert, Input, Space, Typography } from 'antd';
import { PoweroffOutlined, SendOutlined, CodeOutlined, ReloadOutlined } from '@ant-design/icons';

// @ts-expect-error noVNC has no type declarations
import RFB from '@novnc/novnc/lib/rfb';

import { serverAPI } from '../../api';
import { useParams } from 'react-router-dom';

const { Text } = Typography;

const sendTextToVnc = (rfb: any, text: string) => {
  for (const char of text) rfb.sendKey(char.charCodeAt(0));
};

type KvmStatus = 'starting' | 'connecting' | 'connected' | 'error' | 'idle';

const KvmWindowPage: React.FC = () => {
  const { serverId } = useParams<{ serverId: string }>();
  const canvasRef = useRef<HTMLDivElement>(null);
  const rfbRef = useRef<any>(null);
  const [status, setStatus] = useState<KvmStatus>('starting');
  const [error, setError] = useState('');
  const [sessionId, setSessionId] = useState('');
  const [commandText, setCommandText] = useState('');
  const sessionIdRef = useRef('');

  const cleanup = useCallback(() => {
    if (rfbRef.current) { rfbRef.current.disconnect(); rfbRef.current = null; }
    if (canvasRef.current) canvasRef.current.innerHTML = '';
  }, []);

  const startKvm = useCallback(async () => {
    if (!serverId) return;
    cleanup();
    setError('');
    setStatus('starting');
    try {
      const { data: resp } = await serverAPI.kvmStart(serverId);
      if (!resp.success || !resp.data) throw new Error(resp.error || 'Failed to start KVM');
      const data = resp.data as { session_id: string };
      setSessionId(data.session_id);
      sessionIdRef.current = data.session_id;
      setStatus('connecting');
      const wsProto = window.location.protocol === 'https:' ? 'wss' : 'ws';
      const wsUrl = `${wsProto}://${window.location.host}/api/v1/kvm/ws?session=${data.session_id}`;
      await new Promise(r => setTimeout(r, 600));
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
        setStatus('error'); setError('VNC security error: ' + (e.detail?.reason || 'unknown'));
      });
      rfbRef.current = rfb;
    } catch (err: any) {
      setStatus('error');
      setError(err.message || 'Failed to start KVM');
      cleanup();
    }
  }, [serverId, cleanup]);

  const handleClose = useCallback(async () => {
    cleanup();
    const sid = sessionIdRef.current;
    if (sid && serverId) {
      try { await serverAPI.kvmStop(serverId, sid); } catch { /* ignore */ }
    }
    window.close();
  }, [cleanup, serverId]);

  const handleSendCommand = useCallback(() => {
    if (!rfbRef.current || !commandText) return;
    sendTextToVnc(rfbRef.current, commandText);
    rfbRef.current.sendKey(0xFF0D);
    setCommandText('');
  }, [commandText]);

  useEffect(() => {
    document.title = 'KVM Console';
    startKvm();
    return cleanup;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const handler = () => {
      const sid = sessionIdRef.current;
      if (sid && serverId) {
        serverAPI.kvmStop(serverId, sid).catch(() => {});
      }
    };
    window.addEventListener('beforeunload', handler);
    return () => window.removeEventListener('beforeunload', handler);
  }, [serverId]);

  const isLoading = status === 'starting' || status === 'connecting';

  return (
    <div style={{ width: '100vw', height: '100vh', background: '#000', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      {/* Top control bar */}
      <div style={{ background: '#1a1a1a', padding: '4px 10px', display: 'flex', alignItems: 'center', gap: 8, flexShrink: 0, borderBottom: '1px solid #333' }}>
        <Text style={{ color: '#999', fontSize: 12 }}>KVM Console</Text>
        <div style={{ flex: 1 }} />
        <Space size={4}>
          {status === 'connected' && (
            <>
              <Button size="small" onClick={() => rfbRef.current?.sendCtrlAltDel()} style={{ background: '#333', borderColor: '#555', color: '#ccc' }}>
                Ctrl+Alt+Del
              </Button>
              <Input
                size="small"
                placeholder="Send command..."
                value={commandText}
                onChange={(e) => setCommandText(e.target.value)}
                onPressEnter={handleSendCommand}
                prefix={<CodeOutlined style={{ color: '#666' }} />}
                suffix={
                  <Button size="small" type="text" icon={<SendOutlined />} onClick={handleSendCommand} disabled={!commandText} style={{ color: '#999' }} />
                }
                style={{ width: 260, background: '#2a2a2a', borderColor: '#444', color: '#ddd' }}
              />
            </>
          )}
          {(status === 'error' || status === 'idle') && (
            <Button size="small" icon={<ReloadOutlined />} onClick={startKvm} style={{ background: '#333', borderColor: '#555', color: '#ccc' }}>
              Reconnect
            </Button>
          )}
          <Button size="small" danger icon={<PoweroffOutlined />} onClick={handleClose}>
            {status === 'connected' ? 'Disconnect' : 'Close'}
          </Button>
        </Space>
      </div>

      {/* Main area */}
      {isLoading && (
        <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Spin size="large" tip={status === 'starting' ? 'Starting KVM session...' : 'Connecting...'} />
        </div>
      )}
      {(status === 'error' || (status === 'idle' && !isLoading)) && error && (
        <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Alert type="error" message={error} action={<Button size="small" onClick={startKvm}>Retry</Button>} />
        </div>
      )}
      <div
        ref={canvasRef}
        style={{
          flex: 1,
          overflow: 'hidden',
          display: status === 'connected' ? 'block' : 'none',
        }}
      />
    </div>
  );
};

export default KvmWindowPage;
