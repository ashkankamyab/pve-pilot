"use client";

import { useState, useCallback } from "react";
import { usePolling } from "@/hooks/usePolling";
import { apiFetch, apiPost, cpuPercent, formatBytes, formatUptime } from "@/lib/api";
import { ClusterResource, VMStatus } from "@/lib/types";
import StatusBadge from "@/components/shared/StatusBadge";
import PowerButtons from "@/components/shared/PowerButtons";

interface VMWithNode extends VMStatus {
  node: string;
}

export default function VMsPage() {
  const [search, setSearch] = useState("");
  const [actionLoading, setActionLoading] = useState<number | null>(null);

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

    return results.flat();
  }, []);

  const { data: vms, isLoading, refresh } = usePolling(fetchVMs, 5000);

  const handleAction = async (
    node: string,
    vmid: number,
    action: string
  ) => {
    setActionLoading(vmid);
    try {
      await apiPost(`/nodes/${node}/vms/${vmid}/${action}`);
      setTimeout(refresh, 1500);
    } finally {
      setActionLoading(null);
    }
  };

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
    <div className="flex flex-col gap-4">
      <input
        type="text"
        placeholder="Search VMs..."
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className="w-full max-w-sm rounded-md border border-[#222222] bg-[#161616] px-3 py-2 text-sm text-[#e0e0e0] outline-none placeholder:text-[#888888] focus:border-[#00ff88]"
      />

      <div className="overflow-x-auto rounded-lg border border-[#222222]">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-[#222222] bg-[#111111]">
            <tr>
              <th className="px-4 py-3 font-medium text-[#888888]">VMID</th>
              <th className="px-4 py-3 font-medium text-[#888888]">Name</th>
              <th className="px-4 py-3 font-medium text-[#888888]">Node</th>
              <th className="px-4 py-3 font-medium text-[#888888]">Status</th>
              <th className="px-4 py-3 font-medium text-[#888888]">CPU %</th>
              <th className="px-4 py-3 font-medium text-[#888888]">RAM</th>
              <th className="px-4 py-3 font-medium text-[#888888]">Uptime</th>
              <th className="px-4 py-3 font-medium text-[#888888]">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-[#222222]">
            {filtered.map((vm) => (
              <tr key={`${vm.node}-${vm.vmid}`} className="bg-[#161616] hover:bg-[#1a1a1a]">
                <td className="px-4 py-3 font-mono text-[#e0e0e0]">{vm.vmid}</td>
                <td className="px-4 py-3 text-[#e0e0e0]">{vm.name}</td>
                <td className="px-4 py-3 text-[#888888]">{vm.node}</td>
                <td className="px-4 py-3">
                  <StatusBadge status={vm.status} />
                </td>
                <td className="px-4 py-3 text-[#e0e0e0]">
                  {cpuPercent(vm.cpu)}%
                </td>
                <td className="px-4 py-3 text-[#e0e0e0]">
                  {formatBytes(vm.mem)} / {formatBytes(vm.maxmem)}
                </td>
                <td className="px-4 py-3 text-[#888888]">
                  {vm.uptime > 0 ? formatUptime(vm.uptime) : "-"}
                </td>
                <td className="px-4 py-3">
                  <PowerButtons
                    status={vm.status}
                    onStart={() => handleAction(vm.node, vm.vmid, "start")}
                    onStop={() => handleAction(vm.node, vm.vmid, "stop")}
                    onReboot={() => handleAction(vm.node, vm.vmid, "reboot")}
                    isLoading={actionLoading === vm.vmid}
                  />
                </td>
              </tr>
            ))}
            {filtered.length === 0 && (
              <tr>
                <td colSpan={8} className="px-4 py-8 text-center text-[#888888]">
                  No virtual machines found.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
