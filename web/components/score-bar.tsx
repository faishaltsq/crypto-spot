interface Props {
  value: number;
}

export function ScoreBar({ value }: Props) {
  const safe = Math.max(0, Math.min(value, 100));
  return (
    <div className="scoreCell" aria-label={`Score ${safe.toFixed(1)}`}>
      <strong>{safe.toFixed(1)}</strong>
      <div className="scoreTrack">
        <span style={{ width: `${safe}%` }} />
      </div>
    </div>
  );
}
