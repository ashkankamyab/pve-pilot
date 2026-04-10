"use client";

import { useState, useCallback } from "react";
import Link from "next/link";
import { usePolling } from "@/hooks/usePolling";
import { apiFetch, formatBytes } from "@/lib/api";
import { ClusterResource, ContainerStatus, TemplateInfo } from "@/lib/types";
import StatusBadge from "@/components/shared/StatusBadge";
import CreateFromTemplateModal from "@/components/templates/CreateFromTemplateModal";
import DistroIcon from "@/components/shared/DistroIcon";
import { Plus, Search } from "lucide-react";

interface ContainerWithNode extends ContainerStatus {
  node: string;
}

export default function ContainersPage() {
  const [search, setSearch] = useState("");
  const [showCreateFromTemplate, setShowCreateFromTemplate] = useState(false);

  const fetchContainers = useCallback(async () => {
    const resources = await apiFetch<ClusterResource[]>("/cluster/resources");
    const nodeNames = [
      ...new Set(resources.filter((r) => r.type === "node").map((r) => r.node)),
    ];

    const results = await Promise.all(
      nodeNames.map(async (node) => {
        try {
          const cts = await apiFetch<ContainerStatus[]>(`/nodes/${node}/containers`);
          return cts
            .filter((ct) => !ct.template)
            .map((ct) => ({ ...ct, node }));
        } catch {
          return [];
        }
      })
    );

    return results.flat().sort((a, b) => a.vmid - b.vmid);
  }, []);

  const fetchNodes = useCallback(async () => {
    const resources = await apiFetch<ClusterResource[]>("/cluster/resources");
    return [
      ...new Set(resources.filter((r) => r.type === "node").map((r) => r.node)),
    ];
  }, []);

  const fetchTemplates = useCallback(async () => {
    return apiFetch<TemplateInfo[]>("/templates");
  }, []);

  const { data: containers, isLoading, refresh } = usePolling(fetchContainers, 5000);
  const { data: nodes } = usePolling(fetchNodes, 30000);
  const { data: templates } = usePolling(fetchTemplates, 30000);

  const filtered = (containers ?? []).filter(
    (ct) =>
      ct.name.toLowerCase().includes(search.toLowerCase()) ||
      ct.vmid.toString().includes(search)
  );

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <span className="text-[#888888]">Loading...</span>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="flex items-center gap-3">
        <div className="relative flex-1 max-w-sm">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-[#555555]" />
          <input
            type="text"
            placeholder="Search containers..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-md border border-[#222222] bg-[#161616] pl-8 pr-3 py-2 text-sm text-[#e0e0e0] outline-none placeholder:text-[#555555] focus:border-[#00ff88]"
          />
        </div>
        <button
          onClick={() => setShowCreateFromTemplate(true)}
          className="inline-flex items-center gap-2 whitespace-nowrap rounded-md bg-[#00ff88] px-4 py-2 text-sm font-medium text-[#0a0a0a] transition-colors hover:bg-[#00cc6a]"
        >
          <Plus size={16} />
          Create from Template
        </button>
      </div>

      {filtered.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-[#888888]">
          <p className="text-sm">No containers found.</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {filtered.map((ct) => (
            <ContainerCard key={`${ct.node}-${ct.vmid}`} ct={ct} />
          ))}
        </div>
      )}

      <CreateFromTemplateModal
        isOpen={showCreateFromTemplate}
        onClose={() => setShowCreateFromTemplate(false)}
        templates={(templates ?? []).filter((t) => t.vmtype === "lxc")}
        nodes={nodes ?? []}
        defaultType="lxc"
        onSuccess={() => {
          setTimeout(refresh, 2000);
        }}
      />
    </div>
  );
}

function ContainerCard({ ct }: { ct: ContainerWithNode }) {
  const isRunning = ct.status === "running";
  const memPercent = ct.maxmem > 0 ? Math.round((ct.mem / ct.maxmem) * 100) : 0;
  const cpuPercent = Math.round(ct.cpu * 100);

  return (
    <Link
      href={`/containers/${ct.node}/${ct.vmid}`}
      className="group flex flex-col rounded-lg border border-[#222222] bg-[#161616] p-4 transition-all hover:border-[#333333] hover:bg-[#1a1a1a]"
    >
      <div className="flex items-start gap-3">
        <DistroIcon name={ct.name} size={32} />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <h3 className="truncate text-sm font-medium text-[#e0e0e0] group-hover:text-[#00ff88] transition-colors">
              {ct.name}
            </h3>
          </div>
          <div className="flex items-center gap-2 mt-0.5">
            <span className="text-xs text-[#555555] font-mono">CT {ct.vmid}</span>
            <span className="text-[#333333]">·</span>
            <span className="text-xs text-[#555555]">{ct.node}</span>
          </div>
        </div>
        <StatusBadge status={ct.status} />
      </div>

      {isRunning && (
        <div className="mt-4 flex flex-col gap-2">
          <div className="flex items-center justify-between text-xs">
            <span className="text-[#888888]">CPU</span>
            <span className="text-[#e0e0e0] font-mono">{cpuPercent}%</span>
          </div>
          <div className="h-1 w-full overflow-hidden rounded-full bg-[#222222]">
            <div className="h-full rounded-full bg-[#00ff88] transition-all" style={{ width: `${cpuPercent}%` }} />
          </div>

          <div className="flex items-center justify-between text-xs">
            <span className="text-[#888888]">RAM</span>
            <span className="text-[#e0e0e0] font-mono">{formatBytes(ct.mem)} / {formatBytes(ct.maxmem)}</span>
          </div>
          <div className="h-1 w-full overflow-hidden rounded-full bg-[#222222]">
            <div className="h-full rounded-full bg-[#00ff88] transition-all" style={{ width: `${memPercent}%` }} />
          </div>
        </div>
      )}

      {!isRunning && (
        <div className="mt-4 flex items-center justify-center py-3">
          <span className="text-xs text-[#555555]">Stopped</span>
        </div>
      )}
    </Link>
  );
}
