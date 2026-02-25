import React, { useRef, useState, useCallback, useEffect } from 'react';
import { Button, Space, Alert, Spin, Tooltip } from 'antd';
import {
  DesktopOutlined,
  FullscreenOutlined,
  FullscreenExitOutlined,
  PoweroffOutlined,
  LoadingOutlined,
} from '@ant-design/icons';
import { serverAPI } from '../../../api';

// @ts-expect-error noVNC has no type declarations
import RFB from '@novnc/novnc/lib/rfb';

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

    try {
      const { data: resp } = await serverAPI.kvmStart(serverId);
      if (!resp.success || !resp.data) {
        throw new Error(resp.error || 'Failed to start KVM session');
      }

      const { session_id } = resp.data as { session_id: string };
      setSessionId(session_id);
      setStatus('connecting');

      // Build WS URL from current browser location (avoids proxy host mismatch)
      const wsProtocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
      const wsUrl = `${wsProtocol}://${window.location.host}/api/v1/kvm/ws?session=${session_id}`;

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
