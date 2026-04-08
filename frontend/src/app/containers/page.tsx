"use client";

import { useState, useCallback } from "react";
import { usePolling } from "@/hooks/usePolling";
import { apiFetch, apiPost, apiDelete, cpuPercent, formatBytes, formatUptime } from "@/lib/api";
import { ClusterResource, ContainerStatus, TemplateInfo } from "@/lib/types";
import StatusBadge from "@/components/shared/StatusBadge";
import PowerButtons from "@/components/shared/PowerButtons";
import ConfirmDialog from "@/components/shared/ConfirmDialog";
import CreateFromTemplateModal from "@/components/templates/CreateFromTemplateModal";
import { Plus, Trash2 } from "lucide-react";

interface ContainerWithNode extends ContainerStatus {
  node: string;
}

export default function ContainersPage() {
  const [search, setSearch] = useState("");
  const [actionLoading, setActionLoading] = useState<number | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<ContainerWithNode | null>(null);
  const [showCreateFromTemplate, setShowCreateFromTemplate] = useState(false);

  const fetchContainers = useCallback(async () => {
    const resources = await apiFetch<ClusterResource[]>("/cluster/resources");
    const nodeNames = [
      ...new Set(resources.filter((r) => r.type === "node").map((r) => r.node)),
    ];

    const results = await Promise.all(
      nodeNames.map(async (node) => {
        try {
          const cts = await apiFetch<ContainerStatus[]>(
            `/nodes/${node}/containers`
          );
          return cts
            .filter((ct) => !ct.template)
            .map((ct) => ({ ...ct, node }));
        } catch {
          return [];
        }
      })
    );

    return results.flat();
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

  const { data: containers, isLoading, refresh } = usePolling(
    fetchContainers,
    5000
  );
  const { data: nodes } = usePolling(fetchNodes, 30000);
  const { data: templates } = usePolling(fetchTemplates, 30000);

  const handleAction = async (
    node: string,
    vmid: number,
    action: string
  ) => {
    setActionLoading(vmid);
    try {
      await apiPost(`/nodes/${node}/containers/${vmid}/${action}`);
      setTimeout(refresh, 1500);
    } finally {
      setActionLoading(null);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setActionLoading(deleteTarget.vmid);
    try {
      await apiDelete(`/nodes/${deleteTarget.node}/containers/${deleteTarget.vmid}`);
      setTimeout(refresh, 2000);
    } finally {
      setActionLoading(null);
      setDeleteTarget(null);
    }
  };

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
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-3">
        <input
          type="text"
          placeholder="Search containers..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="w-full max-w-sm rounded-md border border-[#222222] bg-[#161616] px-3 py-2 text-sm text-[#e0e0e0] outline-none placeholder:text-[#888888] focus:border-[#00ff88]"
        />
        <button
          onClick={() => setShowCreateFromTemplate(true)}
          className="inline-flex items-center gap-2 whitespace-nowrap rounded-md bg-[#00ff88] px-4 py-2 text-sm font-medium text-[#0a0a0a] transition-colors hover:bg-[#00cc6a]"
        >
          <Plus size={16} />
          Create from Template
        </button>
      </div>

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
            {filtered.map((ct) => (
              <tr
                key={`${ct.node}-${ct.vmid}`}
                className="bg-[#161616] hover:bg-[#1a1a1a]"
              >
                <td className="px-4 py-3 font-mono text-[#e0e0e0]">
                  {ct.vmid}
                </td>
                <td className="px-4 py-3 text-[#e0e0e0]">{ct.name}</td>
                <td className="px-4 py-3 text-[#888888]">{ct.node}</td>
                <td className="px-4 py-3">
                  <StatusBadge status={ct.status} />
                </td>
                <td className="px-4 py-3 text-[#e0e0e0]">
                  {cpuPercent(ct.cpu)}%
                </td>
                <td className="px-4 py-3 text-[#e0e0e0]">
                  {formatBytes(ct.mem)} / {formatBytes(ct.maxmem)}
                </td>
                <td className="px-4 py-3 text-[#888888]">
                  {ct.uptime > 0 ? formatUptime(ct.uptime) : "-"}
                </td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-1">
                    <PowerButtons
                      status={ct.status}
                      onStart={() => handleAction(ct.node, ct.vmid, "start")}
                      onStop={() => handleAction(ct.node, ct.vmid, "stop")}
                      onReboot={() => handleAction(ct.node, ct.vmid, "reboot")}
                      isLoading={actionLoading === ct.vmid}
                    />
                    <button
                      onClick={() => setDeleteTarget(ct)}
                      disabled={ct.status === "running" || actionLoading === ct.vmid}
                      title="Delete Container"
                      className="rounded p-1.5 text-[#888888] transition-colors hover:bg-[#222222] hover:text-red-400 disabled:cursor-not-allowed disabled:opacity-30"
                    >
                      <Trash2 size={15} />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
            {filtered.length === 0 && (
              <tr>
                <td
                  colSpan={8}
                  className="px-4 py-8 text-center text-[#888888]"
                >
                  No containers found.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <ConfirmDialog
        isOpen={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        title="Delete Container"
        message={`Are you sure you want to permanently delete container ${deleteTarget?.vmid} (${deleteTarget?.name})? This action cannot be undone.`}
      />

      <CreateFromTemplateModal
        isOpen={showCreateFromTemplate}
        onClose={() => setShowCreateFromTemplate(false)}
        templates={(templates ?? []).filter((t) => t.vmtype === "lxc")}
        nodes={nodes ?? []}
        onSuccess={() => {
          setShowCreateFromTemplate(false);
          setTimeout(refresh, 2000);
        }}
      />
    </div>
  );
}
