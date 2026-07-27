'use client';

export default function DataQualityError({ reset }: { error: Error; reset: () => void }) {
  return <main className="dq-page"><div className="dq-shell"><section className="dq-error" role="alert"><h1>Data Quality unavailable</h1><p>Quality reports could not be loaded. Backend may be reconnecting.</p><button onClick={reset}>Retry</button></section></div></main>;
}
