import { PerformanceDashboard } from "@/components/performance-dashboard";
import { GlobalHeader } from "@/components/terminal/TerminalHeader";

export const metadata = {
  title: "Proof-of-Edge Dashboard | Crypto Signal",
  description: "Bukti terukur kualitas signal: Win Rate, Return per horizon, MFE/MAE, dan Data Quality Gate.",
};

export default function PerformancePage() {
  return (
    <div className="terminal-app">
      <GlobalHeader />
      <div style={{ padding: '20px', overflowY: 'auto' }}>
        <PerformanceDashboard />
      </div>
    </div>
  );
}
