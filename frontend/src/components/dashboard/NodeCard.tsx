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
      <div className="mb-1 flex items-center justify-between">
        <h3 className="font-mono text-sm font-semibold text-[#e0e0e0]">
          {resource.node}
        </h3>
        <StatusBadge status={resource.status} />
      </div>

      {resource.ip && (
        <p className="mb-3 font-mono text-xs text-[#888888]">{resource.ip}</p>
      )}

      {resource.uptime !== undefined && resource.uptime > 0 && (
        <p className="mb-4 text-xs text-[#888888]">
          Uptime: {formatUptime(resource.uptime)}
        </p>
      )}

      <div className="grid grid-cols-2 gap-4">
        <div className="flex justify-center">
          <ResourceGauge value={cpu} label="CPU" size={80} />
        </div>
        <div className="flex justify-center">
          <ResourceGauge value={mem} label="RAM" size={80} />
        </div>
      </div>
    </div>
  );
}
