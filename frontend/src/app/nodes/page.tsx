"use client";

import { useCallback } from "react";
import { usePolling } from "@/hooks/usePolling";
import { apiFetch, formatBytes, formatUptime, cpuPercent, memPercent } from "@/lib/api";
import { ClusterResource } from "@/lib/types";
import StatusBadge from "@/components/shared/StatusBadge";
import MetricBar from "@/components/shared/MetricBar";
import ResourceGauge from "@/components/dashboard/ResourceGauge";

interface NodeStatus {
  cpu: number;
  cpuinfo: {
    cpus: number;
    model: string;
    sockets: number;
    cores: number;
    threads?: number;
  };
  memory: {
    total: number;
    used: number;
    free: number;
  };
  rootfs: {
    total: number;
    used: number;
    free: number;
    avail: number;
  };
  uptime: number;
  kversion: string;
  pveversion: string;
}

interface NodeInfo {
  name: string;
  status: string;
  detail: NodeStatus | null;
}

export default function NodesPage() {
  const fetchNodes = useCallback(async () => {
    const resources = await apiFetch<ClusterResource[]>("/cluster/resources");
    const nodeResources = resources.filter((r) => r.type === "node");

    const results = await Promise.all(
      nodeResources.map(async (nr) => {
        try {
          const detail = await apiFetch<NodeStatus>(
            `/nodes/${nr.node}/status`
          );
          return { name: nr.node, status: nr.status, detail };
        } catch {
          return { name: nr.node, status: nr.status, detail: null };
        }
      })
    );

    return results;
  }, []);

  const { data: nodes, isLoading } = usePolling(fetchNodes, 5000);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <span className="text-[#888888]">Loading...</span>
      </div>
    );
  }

  return (
    <div className="grid gap-4 lg:grid-cols-2">
      {(nodes ?? []).map((node) => (
        <div
          key={node.name}
          className="rounded-lg border border-[#222222] bg-[#161616] p-5"
        >
          <div className="mb-4 flex items-center justify-between">
            <h3 className="font-mono text-base font-semibold text-[#e0e0e0]">
              {node.name}
            </h3>
            <StatusBadge status={node.status} />
          </div>

          {node.detail ? (
            <div className="flex flex-col gap-4">
              {/* CPU Info */}
              <div className="text-xs text-[#888888]">
                <p>{node.detail.cpuinfo.model}</p>
                <p>
                  {node.detail.cpuinfo.sockets}S / {node.detail.cpuinfo.cores}C
                  {node.detail.cpuinfo.threads
                    ? ` / ${node.detail.cpuinfo.threads}T`
                    : ""}
                  {" "}&middot; {node.detail.cpuinfo.cpus} vCPUs
                </p>
              </div>

              {/* Gauges row */}
              <div className="flex items-center justify-around">
                <ResourceGauge
                  value={cpuPercent(node.detail.cpu)}
                  label="CPU"
                  size={80}
                />
                <ResourceGauge
                  value={memPercent(
                    node.detail.memory.used,
                    node.detail.memory.total
                  )}
                  label="RAM"
                  size={80}
                />
              </div>

              {/* Disk */}
              <MetricBar
                used={node.detail.rootfs.used / 1073741824}
                total={node.detail.rootfs.total / 1073741824}
                label="Root Disk"
                unit="GB"
              />

              {/* Footer info */}
              <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-[#888888]">
                <span>Uptime: {formatUptime(node.detail.uptime)}</span>
                <span>Kernel: {node.detail.kversion.split(" ")[0]}</span>
                <span>PVE: {node.detail.pveversion}</span>
              </div>
            </div>
          ) : (
            <p className="text-sm text-[#888888]">Unable to fetch node details.</p>
          )}
        </div>
      ))}
      {(nodes ?? []).length === 0 && (
        <div className="col-span-full py-8 text-center text-[#888888]">
          No nodes found.
        </div>
      )}
    </div>
  );
}
