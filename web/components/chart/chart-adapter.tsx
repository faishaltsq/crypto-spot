'use client';

import { useMemo } from 'react';
import { LightweightMarketChart } from './LightweightMarketChart';

export interface ChartAdapterProps {
  symbol: string;
  initialData: any; // Using any for now, refine with proper types if needed
}

export function ChartAdapter({ symbol, initialData }: ChartAdapterProps) {
  // Read the environment variable to determine which provider to use.
  // We fall back to lightweight if advanced isn't available or configured.
  const providerType = process.env.NEXT_PUBLIC_CHART_PROVIDER || 'auto';
  
  // Since we don't have the commercial TradingView Advanced Charts library,
  // we will default to using our LightweightMarketChart implementation.
  // In a real scenario, this adapter would dynamically import the advanced chart
  // if providerType is 'advanced' or 'auto' and the library exists.

  return (
    <LightweightMarketChart 
      symbol={symbol} 
      initialData={initialData} 
    />
  );
}
