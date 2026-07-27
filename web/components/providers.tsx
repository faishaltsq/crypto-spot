'use client';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactNode, useEffect, useState } from 'react';
import { useWorkspace } from '@/stores/workspace';
import { GlobalRealtime } from './GlobalRealtime';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      staleTime: 60 * 1000,
    },
  },
});

export function Providers({ children }: { children: ReactNode }) {
  const [mounted, setMounted] = useState(false);
  const theme = useWorkspace((state) => state.theme);

  useEffect(() => {
    setMounted(true);
    document.documentElement.setAttribute('data-theme', theme);
  }, [theme]);

  return (
    <QueryClientProvider client={queryClient}>
      <GlobalRealtime />
      <div style={{ visibility: mounted ? 'visible' : 'hidden' }}>
        {children}
      </div>
    </QueryClientProvider>
  );
}
