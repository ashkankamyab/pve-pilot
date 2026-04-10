"use client";

import { useState, useCallback } from "react";
import Link from "next/link";
import { usePolling } from "@/hooks/usePolling";
import { apiFetch, formatBytes } from "@/lib/api";
import { ClusterResource, VMStatus, TemplateInfo } from "@/lib/types";
import StatusBadge from "@/components/shared/StatusBadge";
import CreateFromTemplateModal from "@/components/templates/CreateFromTemplateModal";
import DistroIcon from "@/components/shared/DistroIcon";
import { Plus, Search } from "lucide-react";

interface VMWithNode extends VMStatus {
  node: string;
}

export default function VMsPage() {
  const [search, setSearch] = useState("");
  const [showCreateFromTemplate, setShowCreateFromTemplate] = useState(false);

  const fetchVMs = useCallback(async () => {
    const resources = await apiFetch<ClusterResource[]>("/cluster/resources");
    const nodeNames = [
      ...new Set(resources.filter((r) => r.type === "node").map((r) => r.node)),
    ];

    const results = await Promise.all(
      nodeNames.map(async (node) => {
        try {
          const vms = await apiFetch<VMStatus[]>(`/nodes/${node}/vms`);
          return vms
            .filter((vm) => !vm.template)
            .map((vm) => ({ ...vm, node }));
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

  const { data: vms, isLoading, refresh } = usePolling(fetchVMs, 5000);
  const { data: nodes } = usePolling(fetchNodes, 30000);
  const { data: templates } = usePolling(fetchTemplates, 30000);

  const filtered = (vms ?? []).filter(
    (vm) =>
      vm.name.toLowerCase().includes(search.toLowerCase()) ||
      vm.vmid.toString().includes(search)
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
            placeholder="Search VMs..."
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
          <p className="text-sm">No virtual machines found.</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {filtered.map((vm) => (
            <VMCard key={`${vm.node}-${vm.vmid}`} vm={vm} />
          ))}
        </div>
      )}

      <CreateFromTemplateModal
        isOpen={showCreateFromTemplate}
        onClose={() => setShowCreateFromTemplate(false)}
        templates={(templates ?? []).filter((t) => t.vmtype === "qemu")}
        nodes={nodes ?? []}
        onSuccess={() => {
          setTimeout(refresh, 2000);
        }}
      />
    </div>
  );
}

function VMCard({ vm }: { vm: VMWithNode }) {
  const isRunning = vm.status === "running";
  const memPercent = vm.maxmem > 0 ? Math.round((vm.mem / vm.maxmem) * 100) : 0;
  const cpuPercent = Math.round(vm.cpu * 100);

  return (
    <Link
      href={`/vms/${vm.node}/${vm.vmid}`}
      className="group flex flex-col rounded-lg border border-[#222222] bg-[#161616] p-4 transition-all hover:border-[#333333] hover:bg-[#1a1a1a]"
    >
      {/* Header */}
      <div className="flex items-start gap-3">
        <DistroIcon name={vm.name} size={32} />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <h3 className="truncate text-sm font-medium text-[#e0e0e0] group-hover:text-[#00ff88] transition-colors">
              {vm.name}
            </h3>
          </div>
          <div className="flex items-center gap-2 mt-0.5">
            <span className="text-xs text-[#555555] font-mono">ID {vm.vmid}</span>
            <span className="text-[#333333]">·</span>
            <span className="text-xs text-[#555555]">{vm.node}</span>
          </div>
        </div>
        <StatusBadge status={vm.status} />
      </div>

      {/* Stats */}
      {isRunning && (
        <div className="mt-4 flex flex-col gap-2">
          <div className="flex items-center justify-between text-xs">
            <span className="text-[#888888]">CPU</span>
            <span className="text-[#e0e0e0] font-mono">{cpuPercent}%</span>
          </div>
          <div className="h-1 w-full overflow-hidden rounded-full bg-[#222222]">
            <div
              className="h-full rounded-full bg-[#00ff88] transition-all"
              style={{ width: `${cpuPercent}%` }}
            />
          </div>

          <div className="flex items-center justify-between text-xs">
            <span className="text-[#888888]">RAM</span>
            <span className="text-[#e0e0e0] font-mono">{formatBytes(vm.mem)} / {formatBytes(vm.maxmem)}</span>
          </div>
          <div className="h-1 w-full overflow-hidden rounded-full bg-[#222222]">
            <div
              className="h-full rounded-full bg-[#00ff88] transition-all"
              style={{ width: `${memPercent}%` }}
            />
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
