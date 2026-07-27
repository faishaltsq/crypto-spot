'use client';
import { useState, useEffect } from 'react';
import { GlobalHeader } from '@/components/terminal/TerminalHeader';

export default function SystemHealthPage() {
  const [health, setHealth] = useState<any>(null);

  useEffect(() => {
    const fetchHealth = async () => {
      try {
        const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/health/system`);
        const data = await res.json();
        setHealth(data);
      } catch (err) {
        console.error('Failed to fetch system health', err);
      }
    };
    fetchHealth();
    const timer = setInterval(fetchHealth, 5000);
    return () => clearInterval(timer);
  }, []);

  if (!health) return <div className="terminal-app"><GlobalHeader /><div style={{ padding: '2rem' }}>Loading system health...</div></div>;

  return (
    <div className="terminal-app">
      <GlobalHeader />
      <div style={{ padding: '2rem', overflowY: 'auto' }}>
        <h1 style={{fontSize: '24px', marginBottom: '16px'}}>System Health</h1>
        <div style={{ background: 'var(--panel)', padding: '1rem', borderRadius: '8px', border: '1px solid var(--border)' }}>
          <div><strong>Overall Status:</strong> {health.status}</div>
          <div style={{marginTop: '8px'}}><strong>Active WebSocket Connections:</strong> {health.ws?.activeConnections}</div>
          <div style={{marginTop: '8px'}}><strong>Average Quality Score:</strong> {health.quality?.avgScore?.toFixed(2)}</div>
        </div>
      </div>
    </div>
  );
}