'use client';

import { ReactNode } from 'react';
import { useWorkspace } from '@/stores/workspace';
import { GlobalHeader } from './TerminalHeader';
// Because react-resizable-panels has undergone some naming changes in recent versions (v4)
import { PanelGroup, Panel, PanelResizeHandle } from 'react-resizable-panels';

interface TerminalLayoutProps {
  symbol: string;
  leftPanel: ReactNode;
  chartPanel: ReactNode;
  rightPanel: ReactNode;
  diagnosticPanel: ReactNode;
}

function ResizeHandle({ direction = 'horizontal' }: { direction?: 'horizontal' | 'vertical' }) {
  return (
      <PanelResizeHandle
        style={{
          [direction === 'horizontal' ? 'width' : 'height']: '4px',
          backgroundColor: 'transparent',
          cursor: direction === 'horizontal' ? 'col-resize' : 'row-resize',
        }}
        onMouseEnter={(e: any) => (e.currentTarget.style.backgroundColor = 'rgba(128, 128, 128, 0.2)')}
        onMouseLeave={(e: any) => (e.currentTarget.style.backgroundColor = 'transparent')}
      />
  );
}

export function TerminalLayout({
  symbol,
  leftPanel,
  chartPanel,
  rightPanel,
  diagnosticPanel,
}: TerminalLayoutProps) {
  const { showLeftPanel, showRightPanel, showDiagnostic, density } = useWorkspace();

  return (
    <div className={`terminal-app ${density}`}>
      <GlobalHeader symbol={symbol} />
      
      <div className="terminal-workspace">
        <PanelGroup direction="horizontal">
          {showLeftPanel && (
            <>
                <Panel defaultSize={20} minSize={10} maxSize={30}>
                  <aside className="terminal-left-panel" style={{ height: '100%', overflow: 'hidden' }}>
                    {leftPanel}
                  </aside>
                </Panel>
              <ResizeHandle direction="horizontal" />
            </>
          )}
          
          <Panel>
            <main className="terminal-main-panel" style={{ height: '100%' }}>
                <PanelGroup direction="vertical">
                <Panel>
                  <div className="terminal-chart-area" style={{ height: '100%', overflow: 'hidden' }}>
                    {chartPanel}
                  </div>
                </Panel>
                
                {showDiagnostic && (
                  <>
                    <ResizeHandle direction="vertical" />
                      <Panel defaultSize={30} minSize={15} maxSize={60}>
                        <div className="terminal-diagnostic-area" style={{ height: '100%', overflow: 'hidden' }}>
                          {diagnosticPanel}
                        </div>
                      </Panel>
                  </>
                )}
              </PanelGroup>
            </main>
          </Panel>
          
          {showRightPanel && (
            <>
              <ResizeHandle direction="horizontal" />
                <Panel defaultSize={20} minSize={10} maxSize={30}>
                  <aside className="terminal-right-panel" style={{ height: '100%', overflow: 'hidden' }}>
                    {rightPanel}
                  </aside>
                </Panel>
            </>
          )}
        </PanelGroup>
      </div>
    </div>
  );
}
