'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Activity, Clock, Settings, Maximize, RefreshCw, LayoutTemplate, Sun, Moon, Monitor } from 'lucide-react';
import { useWorkspace } from '@/stores/workspace';

interface TerminalHeaderProps {
  symbol: string;
}

export function GlobalHeader({ symbol = '' }: { symbol?: string }) {
	const pathname = usePathname();
  const { 
    showLeftPanel, toggleLeftPanel, 
    showRightPanel, toggleRightPanel,
    showDiagnostic, toggleDiagnostic,
    theme, setTheme
  } = useWorkspace();

  return (
    <header className="terminal-topbar">
      <div className="terminal-topbar-left">
        <Link href="/" className="logo">
          <Activity size={20} color="var(--accent)" />
          <span>Crypto Spot Signal</span>
        </Link>
        <span className="mode-badge">LIVE</span>
      </div>

      <nav className="terminal-topbar-center">
        <Link href={symbol ? `/terminal/${symbol}` : '/terminal'} className="nav-item">Terminal</Link>
        <Link href="/signals" className="nav-item">Signals</Link>
        <Link href="/performance" className="nav-item">Performance</Link>
        <Link href="/data-quality" className={`nav-item ${pathname === '/data-quality' ? 'active' : ''}`}>Data Quality</Link>
        <Link href="/system-health" className="nav-item">System Health</Link>
        <Link href="/compare" className="nav-item">Compare</Link>
        <Link href="/watchlist" className="nav-item">Watchlist</Link>
        <Link href="/settings" className="nav-item">Settings</Link>
      </nav>

      <div className="terminal-topbar-right">
        <div className="status-indicator">
          <span className="status-dot positive"></span>
          <span>Gate.io Connected</span>
        </div>
        
        <div className="action-buttons">
          <button 
            className={`icon-btn ${showLeftPanel ? 'active' : ''}`}
            onClick={toggleLeftPanel}
            aria-label="Toggle Left Panel"
          >
            <LayoutTemplate size={16} />
          </button>
          <button 
            className={`icon-btn ${showRightPanel ? 'active' : ''}`}
            onClick={toggleRightPanel}
            title="Toggle Right Panel"
            aria-label="Toggle Right Panel"
          >
            <LayoutTemplate size={16} style={{ transform: 'scaleX(-1)' }}/>
          </button>
          <button 
            className={`icon-btn ${showDiagnostic ? 'active' : ''}`}
            onClick={toggleDiagnostic}
            title="Toggle Bottom Diagnostic"
            aria-label="Toggle Bottom Diagnostic"
          >
            <LayoutTemplate size={16} style={{ transform: 'rotate(90deg)' }}/>
          </button>
          
          <button 
            className="icon-btn"
            onClick={() => setTheme(theme === 'dark' ? 'light' : theme === 'light' ? 'system' : 'dark')}
            title="Toggle Theme"
            aria-label="Toggle Theme"
          >
            {theme === 'dark' ? <Moon size={16} /> : theme === 'light' ? <Sun size={16} /> : <Monitor size={16} />}
          </button>
        </div>
      </div>
    </header>
  );
}
