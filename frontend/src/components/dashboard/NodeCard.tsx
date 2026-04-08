import { ClusterResource } from "@/lib/types";
import { cpuPercent, memPercent, formatUptime } from "@/lib/api";
import StatusBadge from "@/components/shared/StatusBadge";
import ResourceGauge from "./ResourceGauge";

interface NodeCardProps {
  resource: ClusterResource;
}

export default function NodeCard({ resource }: NodeCardProps) {
  const cpu = cpuPercent(resource.cpu ?? 0);
  const mem = memPercent(resource.mem ?? 0, resource.maxmem ?? 0);

  return (
    <div className="rounded-lg border border-[#222222] bg-[#161616] p-5">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="font-mono text-sm font-semibold text-[#e0e0e0]">
          {resource.node}
        </h3>
        <StatusBadge status={resource.status} />
      </div>
      {resource.uptime !== undefined && resource.uptime > 0 && (
        <p className="mb-4 text-xs text-[#888888]">
          Uptime: {formatUptime(resource.uptime)}
        </p>
      )}
      <div className="flex items-center justify-around">
        <ResourceGauge value={cpu} label="CPU" size={80} />
        <ResourceGauge value={mem} label="RAM" size={80} />
      </div>
    </div>
  );
}
