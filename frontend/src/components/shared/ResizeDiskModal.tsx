"use client";

import { useState } from "react";
import Modal from "./Modal";
import { apiPost } from "@/lib/api";
import { Info, Copy, Check } from "lucide-react";

interface DiskInfo {
  name: string;
  currentSize: string; // e.g. "32G"
  currentGB: number;
}

interface ResizeDiskModalProps {
  isOpen: boolean;
  onClose: () => void;
  node: string;
  vmid: number;
  type: "vm" | "container";
  disks: DiskInfo[];
  onSuccess: () => void;
}

export default function ResizeDiskModal({
  isOpen, onClose, node, vmid, type, disks, onSuccess,
}: ResizeDiskModalProps) {
  const isVM = type === "vm";
  const prefix = isVM ? "vms" : "containers";
  const defaultDisk = disks.length > 0 ? disks[0].name : "";

  const [selectedDisk, setSelectedDisk] = useState(defaultDisk);
  const [newSizeGB, setNewSizeGB] = useState("");
  const [phase, setPhase] = useState<"confirm" | "working" | "done" | "failed">("confirm");
  const [statusText, setStatusText] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [copiedField, setCopiedField] = useState<string | null>(null);

  const selected = disks.find((d) => d.name === selectedDisk);
  const minSize = selected ? selected.currentGB : 1;

  const handleResize = async () => {
    if (!selectedDisk || !newSizeGB) return;
    const sizeNum = parseInt(newSizeGB, 10);
    if (sizeNum <= minSize) {
      setError(`New size must be greater than current size (${minSize} GB). Disks can only grow.`);
      return;
    }

    setError(null);
    setPhase("working");
    setStatusText(`Resizing ${selectedDisk} to ${sizeNum}G...`);

    try {
      await apiPost(`/nodes/${node}/${prefix}/${vmid}/resize-disk`, {
        disk: selectedDisk,
        size: `${sizeNum}G`,
      });
      setStatusText(`${selectedDisk} resized to ${sizeNum} GB.`);
      setPhase("done");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Resize failed");
      setPhase("failed");
    }
  };

  const copyToClipboard = async (text: string, field: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedField(field);
      setTimeout(() => setCopiedField(null), 2000);
    } catch {}
  };

  const handleClose = () => {
    if (phase === "working") return;
    if (phase === "done") onSuccess();
    setSelectedDisk(defaultDisk);
    setNewSizeGB("");
    setPhase("confirm");
    setStatusText("");
    setError(null);
    onClose();
  };

  const inputClass = "w-full rounded-md border border-[#222222] bg-[#0a0a0a] px-3 py-2 text-sm text-[#e0e0e0] outline-none focus:border-[#00ff88]";
  const guestCmd = "growpart /dev/sda 1 && resize2fs /dev/sda1";

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title="Resize Disk">
      {phase === "confirm" && (
        <div className="flex flex-col gap-4">
          {disks.length === 0 ? (
            <div className="text-sm text-[#888888]">No disks found.</div>
          ) : (
            <>
              <label className="flex flex-col gap-1">
                <span className="text-xs text-[#888888]">Disk</span>
                <select value={selectedDisk} onChange={(e) => { setSelectedDisk(e.target.value); setNewSizeGB(""); }} className={inputClass}>
                  {disks.map((d) => (
                    <option key={d.name} value={d.name}>{d.name} — {d.currentSize}</option>
                  ))}
                </select>
              </label>

              <label className="flex flex-col gap-1">
                <span className="text-xs text-[#888888]">New Size (GB)</span>
                <input type="number" min={minSize + 1} value={newSizeGB} onChange={(e) => setNewSizeGB(e.target.value)} className={inputClass} placeholder={`Min: ${minSize + 1} GB`} />
                <span className="text-[10px] text-[#555555]">Current: {selected?.currentSize || "—"}. Grow only — shrinking is not supported.</span>
              </label>

              {isVM && (
                <div className="flex items-start gap-3 rounded-lg border border-[#222222] bg-[#111111] p-3">
                  <Info size={16} className="mt-0.5 shrink-0 text-[#888888]" />
                  <div className="text-xs text-[#888888]">
                    After resizing, run inside the VM to expand the filesystem:
                    <code className="mt-1 block rounded bg-[#0a0a0a] px-2 py-1 font-mono text-[#e0e0e0]">{guestCmd}</code>
                  </div>
                </div>
              )}

              {!isVM && selectedDisk === "rootfs" && (
                <div className="flex items-start gap-3 rounded-lg border border-[#00ff88]/20 bg-[#00ff88]/5 p-3">
                  <Info size={16} className="mt-0.5 shrink-0 text-[#00ff88]" />
                  <div className="text-xs text-[#e0e0e0]">Rootfs filesystem will expand automatically.</div>
                </div>
              )}

              {error && <div className="rounded border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">{error}</div>}

              <div className="mt-2 flex justify-end gap-3">
                <button type="button" onClick={handleClose} className="rounded-md border border-[#222222] px-4 py-2 text-sm text-[#e0e0e0] transition-colors hover:bg-[#222222]">Cancel</button>
                <button type="button" onClick={handleResize} disabled={!newSizeGB} className="rounded-md bg-[#00ff88] px-4 py-2 text-sm font-medium text-[#0a0a0a] transition-colors hover:bg-[#00cc6a] disabled:opacity-50">Resize</button>
              </div>
            </>
          )}
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
            <>
              <div className="rounded border border-[#00ff88]/30 bg-[#00ff88]/5 px-3 py-2 text-sm text-[#00ff88]">{statusText}</div>
              {isVM && (
                <div className="flex flex-col gap-2 rounded-lg border border-[#222222] bg-[#111111] p-3">
                  <div className="text-xs text-[#888888]">Run inside the VM to expand the filesystem:</div>
                  <div className="flex items-center gap-2">
                    <code className="flex-1 rounded bg-[#0a0a0a] px-3 py-2 font-mono text-xs text-[#e0e0e0]">{guestCmd}</code>
                    <button type="button" onClick={() => copyToClipboard(guestCmd, "cmd")}
                      className="shrink-0 rounded border border-[#222222] px-2 py-1 text-[10px] text-[#888888] hover:border-[#00ff88] hover:text-[#00ff88]">
                      {copiedField === "cmd" ? <><Check size={10} /> Copied</> : <><Copy size={10} /> Copy</>}
                    </button>
                  </div>
                </div>
              )}
            </>
          )}
          <div className="flex justify-end">
            <button type="button" onClick={handleClose} className="rounded-md bg-[#00ff88] px-4 py-2 text-sm font-medium text-[#0a0a0a] transition-colors hover:bg-[#00cc6a]">Done</button>
          </div>
        </div>
      )}
    </Modal>
  );
}
