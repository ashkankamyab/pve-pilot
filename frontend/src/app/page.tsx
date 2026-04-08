"use client";

import { usePolling } from "@/hooks/usePolling";
import { apiFetch } from "@/lib/api";
import { ClusterSummary, ClusterResource } from "@/lib/types";
import OverviewStats from "@/components/dashboard/OverviewStats";
import NodeCard from "@/components/dashboard/NodeCard";

export default function DashboardPage() {
  const { data: summary, isLoading: loadingSummary } = usePolling(
    () => apiFetch<ClusterSummary>("/cluster/summary"),
    5000
  );

  const { data: resources, isLoading: loadingResources } = usePolling(
    () => apiFetch<ClusterResource[]>("/cluster/resources"),
    5000
  );

  const nodes = resources?.filter((r) => r.type === "node") ?? [];

  if (loadingSummary || loadingResources) {
    return (
      <div className="flex items-center justify-center py-20">
        <span className="text-[#888888]">Loading...</span>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      {summary && <OverviewStats summary={summary} />}

      <div>
        <h2 className="mb-4 text-sm font-semibold uppercase tracking-wider text-[#888888]">
          Nodes
        </h2>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {nodes.map((node) => (
            <NodeCard key={node.id} resource={node} />
          ))}
        </div>
      </div>
    </div>
  );
}
