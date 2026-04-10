"use client";

import { useState } from "react";
import Modal from "./Modal";
import { apiPost, formatBytes } from "@/lib/api";
import { AlertTriangle, Info } from "lucide-react";

interface ScaleModalProps {
  isOpen: boolean;
  onClose: () => void;
  node: string;
  vmid: number;
  vmName: string;
  type: "vm" | "container";
  currentCores: number;
  currentMemoryBytes: number;
  isRunning: boolean;
  onSuccess: () => void;
}

export default function ScaleModal({
  isOpen, onClose, node, vmid, vmName, type, currentCores, currentMemoryBytes, isRunning, onSuccess,
}: ScaleModalProps) {
  const currentMemoryMB = Math.round(currentMemoryBytes / (1024 * 1024));
  const [cores, setCores] = useState(String(currentCores));
  const [memory, setMemory] = useState(String(currentMemoryMB));
  const [phase, setPhase] = useState<"confirm" | "working" | "done" | "failed">("confirm");
  const [statusText, setStatusText] = useState("");
  const [error, setError] = useState<string | null>(null);

  const isVM = type === "vm";
  const noun = isVM ? "VM" : "container";
  const prefix = isVM ? "vms" : "containers";

  const handleScale = async () => {
    setError(null);
    setPhase("working");

    if (isVM && isRunning) {
      setStatusText("Stopping VM...");
    } else {
      setStatusText("Applying resources...");
    }

    try {
      const resp = await apiPost<{ restarted?: boolean; cores: number; memory: number }>(
        `/nodes/${node}/${prefix}/${vmid}/scale`,
        { cores: parseInt(cores, 10), memory: parseInt(memory, 10) }
      );

      if (isVM && resp.restarted) {
        setStatusText(`VM restarted with ${resp.cores} cores and ${resp.memory} MB RAM.`);
      } else {
        setStatusText(`Resources updated: ${resp.cores} cores, ${resp.memory} MB RAM.`);
      }
      setPhase("done");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Scale failed");
      setPhase("failed");
    }
  };

  const handleClose = () => {
    if (phase === "working") return;
    if (phase === "done") onSuccess();
    setCores(String(currentCores));
    setMemory(String(currentMemoryMB));
    setPhase("confirm");
    setStatusText("");
    setError(null);
    onClose();
  };

  const inputClass = "w-full rounded-md border border-[#222222] bg-[#0a0a0a] px-3 py-2 text-sm text-[#e0e0e0] outline-none focus:border-[#00ff88]";

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title={`Scale ${noun}`}>
      {phase === "confirm" && (
        <div className="flex flex-col gap-4">
          {isVM && isRunning && (
            <div className="flex items-start gap-3 rounded-lg border border-yellow-500/30 bg-yellow-500/5 p-3">
              <AlertTriangle size={18} className="mt-0.5 shrink-0 text-yellow-500" />
              <div className="text-sm text-yellow-200">
                This VM is running. Scaling will <strong>stop and restart</strong> the VM. Unsaved data may be lost.
              </div>
            </div>
          )}

          {type === "container" && (
            <div className="flex items-start gap-3 rounded-lg border border-[#00ff88]/30 bg-[#00ff88]/5 p-3">
              <Info size={18} className="mt-0.5 shrink-0 text-[#00ff88]" />
              <div className="text-sm text-[#e0e0e0]">
                Resources will be applied <strong>immediately</strong> without restart.
              </div>
            </div>
          )}

          <div className="grid grid-cols-2 gap-3">
            <label className="flex flex-col gap-1">
              <span className="text-xs text-[#888888]">Cores</span>
              <input type="number" min="1" value={cores} onChange={(e) => setCores(e.target.value)} className={inputClass} />
              <span className="text-[10px] text-[#555555]">Current: {currentCores}</span>
            </label>
            <label className="flex flex-col gap-1">
              <span className="text-xs text-[#888888]">Memory (MB)</span>
              <input type="number" min="128" step="128" value={memory} onChange={(e) => setMemory(e.target.value)} className={inputClass} />
              <span className="text-[10px] text-[#555555]">Current: {formatBytes(currentMemoryBytes)}</span>
            </label>
          </div>

          {error && <div className="rounded border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">{error}</div>}

          <div className="mt-2 flex justify-end gap-3">
            <button type="button" onClick={handleClose} className="rounded-md border border-[#222222] px-4 py-2 text-sm text-[#e0e0e0] transition-colors hover:bg-[#222222]">Cancel</button>
            <button type="button" onClick={handleScale} className="rounded-md bg-[#00ff88] px-4 py-2 text-sm font-medium text-[#0a0a0a] transition-colors hover:bg-[#00cc6a]">
              {isVM && isRunning ? "Scale & Restart" : "Apply"}
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
            <button type="button" onClick={handleClose} className="rounded-md bg-[#00ff88] px-4 py-2 text-sm font-medium text-[#0a0a0a] transition-colors hover:bg-[#00cc6a]">Done</button>
          </div>
        </div>
      )}
    </Modal>
  );
}
