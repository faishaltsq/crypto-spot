interface Props {
  children: React.ReactNode;
  tone?: "positive" | "negative" | "warning" | "neutral";
  title?: string;
}

export function StatusPill({ children, tone = "neutral", title }: Props) {
  return <span className={`pill ${tone}`} title={title}>{children}</span>;
}
