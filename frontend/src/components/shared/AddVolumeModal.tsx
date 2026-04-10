"use client";

import { useState } from "react";
import Modal from "./Modal";
import { apiPost, formatBytes } from "@/lib/api";
import { StorageInfo } from "@/lib/types";
import { Info, Copy, Check } from "lucide-react";

interface AddVolumeModalProps {
  isOpen: boolean;
  onClose: () => void;
  node: string;
  vmid: number;
  type: "vm" | "container";
  storages: StorageInfo[];
  onSuccess: () => void;
}

export default function AddVolumeModal({
  isOpen, onClose, node, vmid, type, storages, onSuccess,
}: AddVolumeModalProps) {
  const isVM = type === "vm";
  const prefix = isVM ? "vms" : "containers";
  const endpoint = isVM ? "add-disk" : "add-volume";

  const validStorages = storages.filter(
    (s) => s.storage !== "local" && s.enabled && s.content.includes(isVM ? "images" : "rootdir")
  );

  const [storage, setStorage] = useState(validStorages[0]?.storage || "");
  const [sizeGB, setSizeGB] = useState("20");
  const [mountPath, setMountPath] = useState("/mnt/data");
  const [phase, setPhase] = useState<"confirm" | "working" | "done" | "failed">("confirm");
  const [statusText, setStatusText] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<{ bus?: string; mount_point?: string; path?: string } | null>(null);
  const [copiedField, setCopiedField] = useState<string | null>(null);

  const handleAdd = async () => {
    if (!storage || !sizeGB) return;
    if (!isVM && !mountPath.startsWith("/")) {
      setError("Mount path must start with /");
      return;
    }

    setError(null);
    setPhase("working");
    setStatusText("Attaching volume...");

    try {
      const body: Record<string, unknown> = { storage, size_gb: parseInt(sizeGB, 10) };
      if (!isVM) body.mount_path = mountPath;

      const resp = await apiPost<{ bus?: string; mount_point?: string; path?: string; storage: string; size_gb: number }>(
        `/nodes/${node}/${prefix}/${vmid}/${endpoint}`, body
      );

      setResult(resp);
      if (isVM) {
        setStatusText(`Attached ${resp.size_gb} GB disk on ${resp.bus}.`);
      } else {
        setStatusText(`Attached ${resp.size_gb} GB volume at ${resp.path}.`);
      }
      setPhase("done");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to attach volume");
      setPhase("failed");
    }
  };

  const copyToClipboard = async (text: string, field: string) => {
    try { await navigator.clipboard.writeText(text); setCopiedField(field); setTimeout(() => setCopiedField(null), 2000); } catch {}
  };

  const handleClose = () => {
    if (phase === "working") return;
    if (phase === "done") onSuccess();
    setStorage(validStorages[0]?.storage || "");
    setSizeGB("20");
    setMountPath("/mnt/data");
    setPhase("confirm");
    setStatusText("");
    setError(null);
    setResult(null);
    onClose();
  };

  const inputClass = "w-full rounded-md border border-[#222222] bg-[#0a0a0a] px-3 py-2 text-sm text-[#e0e0e0] outline-none focus:border-[#00ff88]";
  const scsiRescan = 'echo "- - -" > /sys/class/scsi_host/host0/scan';

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title={isVM ? "Add Disk" : "Add Volume"}>
      {phase === "confirm" && (
        <div className="flex flex-col gap-4">
          {validStorages.length === 0 ? (
            <div className="text-sm text-[#888888]">No suitable storage pools found.</div>
          ) : (
            <>
              <label className="flex flex-col gap-1">
                <span className="text-xs text-[#888888]">Storage</span>
                <select value={storage} onChange={(e) => setStorage(e.target.value)} className={inputClass}>
                  {validStorages.map((s) => (
                    <option key={s.storage} value={s.storage}>{s.storage} — {formatBytes(s.avail)} free</option>
                  ))}
                </select>
              </label>

              <label className="flex flex-col gap-1">
                <span className="text-xs text-[#888888]">Size (GB)</span>
                <input type="number" min="1" value={sizeGB} onChange={(e) => setSizeGB(e.target.value)} className={inputClass} />
              </label>

              {!isVM && (
                <label className="flex flex-col gap-1">
                  <span className="text-xs text-[#888888]">Mount Path</span>
                  <input type="text" value={mountPath} onChange={(e) => setMountPath(e.target.value)} className={inputClass} placeholder="/mnt/data" />
                  <span className="text-[10px] text-[#555555]">Must start with /</span>
                </label>
              )}

              {isVM && (
                <div className="flex items-start gap-3 rounded-lg border border-[#222222] bg-[#111111] p-3">
                  <Info size={16} className="mt-0.5 shrink-0 text-[#888888]" />
                  <div className="text-xs text-[#888888]">
                    Disk will be attached as the next available SCSI slot. The guest may need a bus rescan to detect it.
                  </div>
                </div>
              )}

              {error && <div className="rounded border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">{error}</div>}

              <div className="mt-2 flex justify-end gap-3">
                <button type="button" onClick={handleClose} className="rounded-md border border-[#222222] px-4 py-2 text-sm text-[#e0e0e0] transition-colors hover:bg-[#222222]">Cancel</button>
                <button type="button" onClick={handleAdd} disabled={!storage || !sizeGB} className="rounded-md bg-[#00ff88] px-4 py-2 text-sm font-medium text-[#0a0a0a] transition-colors hover:bg-[#00cc6a] disabled:opacity-50">Attach</button>
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
                  <div className="text-xs text-[#888888]">If the guest doesn't detect the new disk, run:</div>
                  <div className="flex items-center gap-2">
                    <code className="flex-1 rounded bg-[#0a0a0a] px-3 py-2 font-mono text-xs text-[#e0e0e0]">{scsiRescan}</code>
                    <button type="button" onClick={() => copyToClipboard(scsiRescan, "cmd")}
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
