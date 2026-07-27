'use client';

import { useParams, useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { TerminalLayout } from '@/components/terminal/TerminalLayout';
import { VirtualPairList } from '@/components/pairs/VirtualPairList';
import { ChartAdapter } from '@/components/chart/chart-adapter';
import { RightSignalPanel } from '@/components/signals/RightSignalPanel';
import { PairDiagnostic } from '@/components/diagnostics/PairDiagnostic';
import { API_URL } from '@/lib/api';
import { useEffect } from 'react';

// Fetcher function
async function fetchTerminalSnapshot(symbol: string) {
  const res = await fetch(`${API_URL}/api/v1/terminal/${symbol}`);
  if (!res.ok) {
    if (res.status === 404) return null;
    throw new Error('Failed to fetch terminal data');
  }
  return res.json();
}

export default function TerminalPage() {
  const params = useParams();
  const router = useRouter();
  
  // Extract symbol from URL, default to BTC_USDT if not present
  const symbolParam = params.symbol?.[0];
  const symbol = symbolParam ? decodeURIComponent(symbolParam) : 'BTC_USDT';

  // Automatically redirect if no symbol is in the URL
  useEffect(() => {
    if (!symbolParam) {
      router.replace(`/terminal/BTC_USDT`);
    }
  }, [symbolParam, router]);

  const { data, isLoading, error } = useQuery({
    queryKey: ['terminal', symbol],
    queryFn: () => fetchTerminalSnapshot(symbol),
    enabled: !!symbolParam,
  });

  if (!symbolParam) {
    return <div className="terminal-app"><div style={{padding: 20}}>Loading terminal...</div></div>;
  }

  return (
    <TerminalLayout
      symbol={symbol}
      leftPanel={<VirtualPairList activeSymbol={symbol} />}
      chartPanel={
        <div style={{ display: 'flex', flexDirection: 'column', flex: 1, height: '100%' }}>
          {isLoading ? (
            <div style={{ padding: 16 }}>Loading chart...</div>
          ) : error || !data ? (
            <div style={{ padding: 16 }}>Error or Pair Not Found</div>
          ) : (
            <ChartAdapter symbol={symbol} initialData={data} />
          )}
        </div>
      }
      rightPanel={<RightSignalPanel symbol={symbol} initialSignals={data?.signals} />}
      diagnosticPanel={
        data ? (
          <PairDiagnostic symbol={symbol} diagnosticData={data.diagnostic} />
        ) : (
          <div style={{ padding: 16 }}>Loading diagnostic...</div>
        )
      }
    />
  );
}
