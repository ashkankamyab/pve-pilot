import { ClusterSummary } from "@/lib/types";
import { formatBytes } from "@/lib/api";

interface OverviewStatsProps {
  summary: ClusterSummary;
}

interface StatCardProps {
  label: string;
  value: string | number;
  sub?: string;
}

function StatCard({ label, value, sub }: StatCardProps) {
  return (
    <div className="rounded-lg border border-[#222222] bg-[#161616] p-4">
      <p className="text-xs text-[#888888]">{label}</p>
      <p className="mt-1 text-2xl font-bold text-[#e0e0e0]">{value}</p>
      {sub && <p className="mt-0.5 text-xs text-[#888888]">{sub}</p>}
    </div>
  );
}

export default function OverviewStats({ summary }: OverviewStatsProps) {
  return (
    <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
      <StatCard
        label="Nodes"
        value={summary.nodes_online}
        sub={`${summary.nodes} total`}
      />
      <StatCard
        label="Virtual Machines"
        value={summary.vms_running}
        sub={`${summary.vms_total} total`}
      />
      <StatCard
        label="Containers"
        value={summary.containers_running}
        sub={`${summary.containers_total} total`}
      />
      <StatCard
        label="Storage"
        value={formatBytes(summary.disk_used)}
        sub={`of ${formatBytes(summary.disk_total)}`}
      />
    </div>
  );
}
