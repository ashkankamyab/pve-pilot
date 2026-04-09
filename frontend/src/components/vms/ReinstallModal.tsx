"use client";

import { useState, useEffect, useCallback } from "react";
import Modal from "@/components/shared/Modal";
import { apiPost, apiDelete, apiFetch } from "@/lib/api";
import { TemplateInfo } from "@/lib/types";
import { useJobs } from "@/contexts/JobsContext";
import TemplateSelect from "@/components/shared/TemplateSelect";
import { detectDistro, DISTRO_USERS } from "@/components/shared/DistroIcon";
import { AlertTriangle } from "lucide-react";

interface ReinstallModalProps {
  isOpen: boolean;
  onClose: () => void;
  vmid: number;
  vmName: string;
  node: string;
  currentDistroHint: string;
  templates: TemplateInfo[];
  buildPassword?: string;
  buildSshKey?: string;
  buildCiUser?: string;
  onSuccess: () => void;
}

export default function ReinstallModal({
  isOpen, onClose, vmid, vmName, node, currentDistroHint,
  templates, buildPassword, buildSshKey, buildCiUser, onSuccess,
}: ReinstallModalProps) {
  const { addJob } = useJobs();

  const [selectedTemplateId, setSelectedTemplateId] = useState("");
  const [ciUser, setCiUser] = useState(buildCiUser || "");
  const [error, setError] = useState<string | null>(null);
  const [phase, setPhase] = useState<"confirm" | "working" | "done" | "failed">("confirm");
  const [statusText, setStatusText] = useState("");

  const selectedTemplate = templates.find(
    (t) => `${t.node}-${t.vmid}` === selectedTemplateId
  );

  // Pre-select template matching current distro
  useEffect(() => {
    if (!isOpen || selectedTemplateId) return;
    const match = templates.find((t) => {
      const d = detectDistro(t.name || "");
      const current = detectDistro(currentDistroHint);
      return d === current;
    });
    if (match) {
      setSelectedTemplateId(`${match.node}-${match.vmid}`);
    }
  }, [isOpen, templates, currentDistroHint, selectedTemplateId]);

  // Auto-detect CI user when template changes
  useEffect(() => {
    if (selectedTemplate?.name) {
      setCiUser(DISTRO_USERS[detectDistro(selectedTemplate.name)]);
    }
  }, [selectedTemplate?.name]);

  const handleReinstall = async () => {
    if (!selectedTemplate) return;
    setError(null);
    setPhase("working");

    try {
      // Step 1: Stop VM (ignore errors if already stopped)
      setStatusText("Stopping VM...");
      try {
        await apiPost(`/nodes/${node}/vms/${vmid}/stop`);
        // Wait for it to actually stop
        for (let i = 0; i < 30; i++) {
          await new Promise((r) => setTimeout(r, 2000));
          try {
            const status = await apiFetch<{ status: string }>(`/nodes/${node}/vms/${vmid}/status`);
            if (status.status === "stopped") break;
          } catch { break; }
        }
      } catch {
        // Already stopped, continue
      }

      // Step 2: Delete VM
      setStatusText("Deleting old VM...");
      await apiDelete(`/nodes/${node}/vms/${vmid}`);

      // Wait for deletion to complete
      for (let i = 0; i < 30; i++) {
        await new Promise((r) => setTimeout(r, 2000));
        try {
          await apiFetch(`/nodes/${node}/vms/${vmid}/status`);
        } catch {
          break; // VM gone
        }
      }

      // Step 3: Re-provision with same VMID
      setStatusText("Creating new VM...");
      const typePrefix = selectedTemplate.vmtype === "qemu" ? "vms" : "containers";
      const response = await apiPost<{ job_id: string; vmid: number; node: string }>(
        `/nodes/${selectedTemplate.node}/${typePrefix}/${selectedTemplate.vmid}/provision`,
        {
          newid: vmid,
          name: vmName,
          full: true,
          storage: undefined,
          ciuser: ciUser || undefined,
          password: buildPassword || undefined,
          sshkeys: buildSshKey || undefined,
          disk_size: 30,
        }
      );

      addJob({
        id: response.job_id,
        type: selectedTemplate.vmtype === "qemu" ? "vm" : "container",
        status: "pending", step: "", progress: 0,
        created_at: new Date().toISOString(), updated_at: new Date().toISOString(),
        source_node: selectedTemplate.node, source_vmid: selectedTemplate.vmid!,
        target_node: response.node, new_vmid: response.vmid, name: vmName,
        ciuser: ciUser || undefined, disk_size: 30, full_clone: true,
        password: buildPassword || undefined, sshkey: buildSshKey || undefined,
      });

      setPhase("done");
      setStatusText("Reinstall started! Check Jobs panel for progress.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Reinstall failed");
      setPhase("failed");
    }
  };

  const handleClose = () => {
    if (phase === "working") return;
    setSelectedTemplateId("");
    setCiUser(buildCiUser || "");
    setError(null);
    setPhase("confirm");
    setStatusText("");
    if (phase === "done") onSuccess();
    onClose();
  };

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title="Reinstall OS">
      {phase === "confirm" && (
        <div className="flex flex-col gap-4">
          <div className="flex items-start gap-3 rounded-lg border border-yellow-500/30 bg-yellow-500/5 p-3">
            <AlertTriangle size={18} className="mt-0.5 shrink-0 text-yellow-500" />
            <div className="text-sm text-yellow-200">
              This will <strong>destroy all data</strong> on VM {vmid} ({vmName}) and create a fresh install.
              The VMID and name will be preserved.
            </div>
          </div>

          <div className="flex flex-col gap-1">
            <span className="text-xs text-[#888888]">New Template</span>
            <TemplateSelect templates={templates} value={selectedTemplateId} onChange={setSelectedTemplateId} />
          </div>

          <div className="flex flex-col gap-1">
            <span className="text-xs text-[#888888]">SSH User</span>
            <input
              type="text"
              value={ciUser}
              onChange={(e) => setCiUser(e.target.value)}
              className="w-full rounded-md border border-[#222222] bg-[#0a0a0a] px-3 py-2 text-sm text-[#e0e0e0] outline-none focus:border-[#00ff88]"
            />
          </div>

          {(buildPassword || buildSshKey) && (
            <div className="text-xs text-[#555555]">
              Previous credentials (password, SSH key) will be reused.
            </div>
          )}

          {error && <div className="rounded border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">{error}</div>}

          <div className="mt-2 flex justify-end gap-3">
            <button type="button" onClick={handleClose} className="rounded-md border border-[#222222] px-4 py-2 text-sm text-[#e0e0e0] transition-colors hover:bg-[#222222]">Cancel</button>
            <button
              type="button"
              disabled={!selectedTemplateId}
              onClick={handleReinstall}
              className="rounded-md bg-red-500 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-red-600 disabled:opacity-50"
            >
              Reinstall
            </button>
          </div>
        </div>
      )}

      {phase === "working" && (
        <div className="flex flex-col items-center gap-3 py-8">
          <div className="h-6 w-6 animate-spin rounded-full border-2 border-[#333333] border-t-[#00ff88]" />
          <span className="text-sm text-[#e0e0e0]">{statusText}</span>
        </div>
      )}

      {(phase === "done" || phase === "failed") && (
        <div className="flex flex-col gap-4">
          {error && <div className="rounded border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">{error}</div>}
          {phase === "done" && (
            <div className="rounded border border-[#00ff88]/30 bg-[#00ff88]/5 px-3 py-2 text-sm text-[#00ff88]">{statusText}</div>
          )}
          <div className="flex justify-end">
            <button type="button" onClick={handleClose} className="rounded-md bg-[#00ff88] px-4 py-2 text-sm font-medium text-[#0a0a0a] transition-colors hover:bg-[#00cc6a]">
              Done
            </button>
          </div>
        </div>
      )}
    </Modal>
  );
}
