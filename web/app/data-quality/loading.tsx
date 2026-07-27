export default function LoadingDataQuality() {
  return <main className="dq-page" aria-busy="true"><div className="dq-shell"><p className="dq-kicker">DATA QUALITY</p><h1>Loading quality reports...</h1><div className="dq-skeleton-grid">{Array.from({ length: 8 }).map((_, index) => <div className="dq-skeleton" key={index} />)}</div></div></main>;
}
