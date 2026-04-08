"use client";

import { useState, useCallback } from "react";
import { usePolling } from "@/hooks/usePolling";
import { apiFetch } from "@/lib/api";
import { TemplateInfo, ClusterResource } from "@/lib/types";
import CloneModal from "@/components/templates/CloneModal";
import { Copy } from "lucide-react";

export default function TemplatesPage() {
  const [selectedTemplate, setSelectedTemplate] = useState<TemplateInfo | null>(
    null
  );

  const { data: templates, isLoading } = usePolling(
    () => apiFetch<TemplateInfo[]>("/templates"),
    10000
  );

  const fetchNodes = useCallback(async () => {
    const resources = await apiFetch<ClusterResource[]>("/cluster/resources");
    return [
      ...new Set(resources.filter((r) => r.type === "node").map((r) => r.node)),
    ];
  }, []);

  const { data: nodes } = usePolling(fetchNodes, 30000);

  const vmTemplates = (templates ?? []).filter((t) => t.vmtype === "qemu");
  const lxcTemplates = (templates ?? []).filter((t) => t.vmtype === "lxc");

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <span className="text-[#888888]">Loading...</span>
      </div>
    );
  }

  const renderTable = (items: TemplateInfo[], title: string) => (
    <div>
      <h2 className="mb-3 text-sm font-semibold uppercase tracking-wider text-[#888888]">
        {title}
      </h2>
      <div className="overflow-x-auto rounded-lg border border-[#222222]">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-[#222222] bg-[#111111]">
            <tr>
              <th className="px-4 py-3 font-medium text-[#888888]">VMID</th>
              <th className="px-4 py-3 font-medium text-[#888888]">Name</th>
              <th className="px-4 py-3 font-medium text-[#888888]">Node</th>
              <th className="px-4 py-3 font-medium text-[#888888]">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-[#222222]">
            {items.map((t) => (
              <tr
                key={t.id}
                className="bg-[#161616] hover:bg-[#1a1a1a]"
              >
                <td className="px-4 py-3 font-mono text-[#e0e0e0]">
                  {t.vmid}
                </td>
                <td className="px-4 py-3 text-[#e0e0e0]">
                  {t.name || "-"}
                </td>
                <td className="px-4 py-3 text-[#888888]">{t.node}</td>
                <td className="px-4 py-3">
                  <button
                    onClick={() => setSelectedTemplate(t)}
                    className="inline-flex items-center gap-1.5 rounded-md border border-[#222222] px-3 py-1.5 text-xs text-[#e0e0e0] transition-colors hover:border-[#00ff88] hover:text-[#00ff88]"
                  >
                    <Copy size={13} />
                    Clone
                  </button>
                </td>
              </tr>
            ))}
            {items.length === 0 && (
              <tr>
                <td
                  colSpan={4}
                  className="px-4 py-8 text-center text-[#888888]"
                >
                  No templates found.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );

  return (
    <div className="flex flex-col gap-8">
      {renderTable(vmTemplates, "VM Templates")}
      {renderTable(lxcTemplates, "LXC Templates")}

      <CloneModal
        isOpen={!!selectedTemplate}
        onClose={() => setSelectedTemplate(null)}
        template={selectedTemplate}
        nodes={nodes ?? []}
      />
    </div>
  );
}
