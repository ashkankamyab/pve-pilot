"use client";

import { useState, useEffect, useCallback } from "react";
import Modal from "./Modal";
import { apiPost, apiDelete, apiFetch, formatBytes } from "@/lib/api";
import { BackupInfo, BackupSchedule, Job } from "@/lib/types";
import { useJobs } from "@/contexts/JobsContext";
import { Download, RotateCcw, Trash2, Clock, AlertTriangle, Check, Calendar, ExternalLink } from "lucide-react";
import Link from "next/link";

interface BackupModalProps {
  isOpen: boolean;
  onClose: () => void;
  node: string;
  vmid: number;
  vmName: string;
  type: "vm" | "container";
  onSuccess: () => void;
}

type Tab = "history" | "schedule";

const SCHEDULE_PRESETS = [
  { label: "Daily (4 AM)", cron: "0 4 * * *", comment: "Daily backup at 4 AM" },
  { label: "Weekly (Sun 4 AM)", cron: "0 4 * * 0", comment: "Weekly backup Sunday 4 AM" },
  { label: "Monthly (1st 4 AM)", cron: "0 4 1 * *", comment: "Monthly backup 1st day 4 AM" },
];

function formatDate(ts: number): string {
  return new Date(ts * 1000).toLocaleString();
}

function formatAge(ts: number): string {
  const diff = Math.floor(Date.now() / 1000 - ts);
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return `${Math.floor(diff / 86400)}d ago`;
}

export default function BackupModal({
  isOpen, onClose, node, vmid, vmName, type, onSuccess,
}: BackupModalProps) {
  const { addJob } = useJobs();
  const prefix = type === "vm" ? "vms" : "containers";
  const noun = type === "vm" ? "VM" : "container";

  const [tab, setTab] = useState<Tab>("history");
  const [backups, setBackups] = useState<BackupInfo[]>([]);
  const [schedules, setSchedules] = useState<BackupSchedule[]>([]);
  const [loading, setLoading] = useState(false);
  const [working, setWorking] = useState(false);
  const [workingText, setWorkingText] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Restore state
  const [restoreTarget, setRestoreTarget] = useState<BackupInfo | null>(null);
  const [restoreInPlace, setRestoreInPlace] = useState(true);
  const [restoredNewVmid, setRestoredNewVmid] = useState<number | null>(null);

  // Backup notes
  const [backupNotes, setBackupNotes] = useState("");

  const fetchBackups = useCallback(async () => {
    try {
      const list = await apiFetch<BackupInfo[]>(`/nodes/${node}/${prefix}/${vmid}/backups`);
      setBackups(list || []);
    } catch { setBackups([]); }
  }, [node, prefix, vmid]);

  const fetchSchedules = useCallback(async () => {
    try {
      const list = await apiFetch<BackupSchedule[]>("/backup-schedules");
      // Filter to schedules that include this VMID
      const relevant = (list || []).filter((s) => {
        if (!s.vmid) return false;
        return s.vmid.split(",").map(v => v.trim()).includes(String(vmid));
      });
      setSchedules(relevant);
    } catch { setSchedules([]); }
  }, [vmid]);

  useEffect(() => {
    if (!isOpen) return;
    setLoading(true);
    Promise.all([fetchBackups(), fetchSchedules()]).finally(() => setLoading(false));
  }, [isOpen, fetchBackups, fetchSchedules]);

  const handleBackupNow = async () => {
    setError(null);
    setSuccess(null);

    try {
      const resp = await apiPost<{ job_id: string }>(`/nodes/${node}/${prefix}/${vmid}/backup`, { notes: backupNotes || undefined });
      const job: Job = {
        id: resp.job_id,
        type: type === "vm" ? "backup_vm" : "backup_container",
        status: "pending",
        step: "",
        progress: 0,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        source_node: node,
        source_vmid: vmid,
        target_node: node,
        new_vmid: vmid,
        name: vmName,
        full_clone: false,
      };
      addJob(job);
      setSuccess("Backup job started — track progress in the Jobs panel.");
      setBackupNotes("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Backup failed");
    }
  };

  const handleDeleteBackup = async (volid: string) => {
    setError(null);
    setWorking(true);
    setWorkingText("Deleting backup...");

    try {
      await apiDelete(`/backups?node=${encodeURIComponent(node)}&volid=${encodeURIComponent(volid)}`);
      await fetchBackups();
      setSuccess("Backup deleted.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Delete failed");
    } finally {
      setWorking(false);
      setWorkingText("");
    }
  };

  const handleRestore = async () => {
    if (!restoreTarget) return;
    setError(null);
    setSuccess(null);

    const restoreType = type === "vm" ? "vm" : "container";

    try {
      const resp = await apiPost<{ job_id: string; vmid: number }>(`/nodes/${node}/restore/${restoreType}`, {
        archive: restoreTarget.volid,
        vmid: restoreInPlace ? vmid : 0,
        in_place: restoreInPlace,
      });
      const targetVmid = resp.vmid || vmid;
      const job: Job = {
        id: resp.job_id,
        type: type === "vm" ? "restore_vm" : "restore_container",
        status: "pending",
        step: "",
        progress: 0,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        source_node: node,
        source_vmid: vmid,
        target_node: node,
        new_vmid: targetVmid,
        name: vmName,
        full_clone: false,
      };
      addJob(job);
      if (restoreInPlace) {
        setSuccess(`Restore job started for ${vmName} — track progress in the Jobs panel.`);
      } else {
        setRestoredNewVmid(targetVmid);
        setSuccess(`Restore job started as new ${noun} (ID ${targetVmid}) — track progress in the Jobs panel.`);
      }
      setRestoreTarget(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Restore failed");
    }
  };

  const handleCreateSchedule = async (cron: string, comment: string) => {
    setError(null);
    setWorking(true);
    setWorkingText("Creating schedule...");

    try {
      await apiPost("/backup-schedules", {
        vmid: String(vmid),
        storage: "",
        schedule: cron,
        mode: "snapshot",
        compress: "zstd",
        comment: `${vmName}: ${comment}`,
        enabled: true,
        node: node,
      });
      setSuccess("Schedule created.");
      await fetchSchedules();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create schedule");
    } finally {
      setWorking(false);
      setWorkingText("");
    }
  };

  const handleDeleteSchedule = async (id: string) => {
    setError(null);
    setWorking(true);
    setWorkingText("Removing schedule...");

    try {
      await apiDelete(`/backup-schedules/${id}`);
      await fetchSchedules();
      setSuccess("Schedule removed.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete schedule");
    } finally {
      setWorking(false);
      setWorkingText("");
    }
  };

  const handleClose = () => {
    setError(null);
    setSuccess(null);
    setRestoreTarget(null);
    setRestoredNewVmid(null);
    setBackupNotes("");
    onClose();
  };

  const inputClass = "w-full rounded-md border border-[#222222] bg-[#0a0a0a] px-3 py-2 text-sm text-[#e0e0e0] outline-none focus:border-[#00ff88]";

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title={`Backups — ${vmName}`}>
      <div className="flex flex-col gap-4">
        {/* Working spinner */}
        {working && (
          <div className="flex items-center gap-3 rounded-lg border border-[#222222] bg-[#111111] px-4 py-3">
            <div className="h-4 w-4 animate-spin rounded-full border-2 border-[#333333] border-t-[#00ff88]" />
            <span className="text-sm text-[#e0e0e0]">{workingText}</span>
          </div>
        )}

        {/* Status messages */}
        {error && <div className="rounded border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">{error}</div>}
        {success && (
          <div className="rounded border border-[#00ff88]/30 bg-[#00ff88]/5 px-3 py-2 text-sm text-[#00ff88]">
            {success}
            {restoredNewVmid && (
              <Link
                href={`/${type === "vm" ? "vms" : "containers"}/${node}/${restoredNewVmid}`}
                className="ml-2 inline-flex items-center gap-1 rounded bg-[#00ff88]/20 px-2 py-0.5 text-xs font-medium text-[#00ff88] hover:bg-[#00ff88]/30 transition-colors"
              >
                Open #{restoredNewVmid} <ExternalLink size={10} />
              </Link>
            )}
          </div>
        )}

        {/* Restore confirmation overlay */}
        {restoreTarget && (
          <div className="flex flex-col gap-3 rounded-lg border border-yellow-500/30 bg-yellow-500/5 p-4">
            <div className="flex items-start gap-3">
              <AlertTriangle size={18} className="mt-0.5 shrink-0 text-yellow-500" />
              <div className="text-sm text-yellow-200">
                Restore from <strong>{formatDate(restoreTarget.ctime)}</strong> ({formatBytes(restoreTarget.size)})?
              </div>
            </div>

            <div className="flex gap-3">
              <label className="flex items-center gap-1.5">
                <input type="radio" checked={restoreInPlace} onChange={() => setRestoreInPlace(true)} className="accent-[#00ff88]" />
                <span className="text-xs text-[#e0e0e0]">Restore in-place (delete current)</span>
              </label>
              <label className="flex items-center gap-1.5">
                <input type="radio" checked={!restoreInPlace} onChange={() => setRestoreInPlace(false)} className="accent-[#00ff88]" />
                <span className="text-xs text-[#e0e0e0]">Restore as new</span>
              </label>
            </div>

            <div className="flex justify-end gap-2">
              <button onClick={() => setRestoreTarget(null)} className="rounded-md border border-[#222222] px-3 py-1.5 text-xs text-[#e0e0e0] hover:bg-[#222222]">Cancel</button>
              <button onClick={handleRestore} disabled={working} className="rounded-md bg-orange-500 px-3 py-1.5 text-xs font-medium text-white hover:bg-orange-600 disabled:opacity-50">Restore</button>
            </div>
          </div>
        )}

        {/* Backup Now */}
        {!restoreTarget && (
          <div className="flex items-end gap-2">
            <div className="flex-1">
              <input type="text" value={backupNotes} onChange={(e) => setBackupNotes(e.target.value)} className={inputClass} placeholder="Backup notes (optional)" />
            </div>
            <button onClick={handleBackupNow} disabled={working}
              className="inline-flex shrink-0 items-center gap-2 rounded-md bg-[#00ff88] px-4 py-2 text-sm font-medium text-[#0a0a0a] transition-colors hover:bg-[#00cc6a] disabled:opacity-50">
              <Download size={14} /> Backup Now
            </button>
          </div>
        )}

        {/* Tabs */}
        <div className="flex border-b border-[#222222]">
          <button onClick={() => setTab("history")}
            className={`px-4 py-2 text-xs font-medium transition-colors ${tab === "history" ? "border-b-2 border-[#00ff88] text-[#00ff88]" : "text-[#888888] hover:text-[#e0e0e0]"}`}>
            <Clock size={12} className="mr-1.5 inline" /> History
          </button>
          <button onClick={() => setTab("schedule")}
            className={`px-4 py-2 text-xs font-medium transition-colors ${tab === "schedule" ? "border-b-2 border-[#00ff88] text-[#00ff88]" : "text-[#888888] hover:text-[#e0e0e0]"}`}>
            <Calendar size={12} className="mr-1.5 inline" /> Schedule
          </button>
        </div>

        {/* History tab */}
        {tab === "history" && (
          <div className="flex flex-col gap-2 max-h-64 overflow-y-auto">
            {loading && <div className="text-xs text-[#555555] text-center py-4">Loading...</div>}
            {!loading && backups.length === 0 && (
              <div className="text-xs text-[#555555] text-center py-4">No backups found for this {noun}.</div>
            )}
            {backups.map((b) => (
              <div key={b.volid} className="flex items-center gap-3 rounded-md bg-[#111111] px-3 py-2.5">
                <div className="flex-1 min-w-0">
                  <div className="text-xs text-[#e0e0e0]">{formatDate(b.ctime)}</div>
                  <div className="flex items-center gap-2 text-[10px] text-[#555555]">
                    <span>{formatBytes(b.size)}</span>
                    <span>·</span>
                    <span>{b.format}</span>
                    <span>·</span>
                    <span>{formatAge(b.ctime)}</span>
                  </div>
                  {b.notes && <div className="text-[10px] text-[#888888] truncate mt-0.5">{b.notes}</div>}
                </div>
                <button onClick={() => setRestoreTarget(b)} disabled={working}
                  className="shrink-0 rounded border border-[#222222] px-2 py-1 text-[10px] text-[#888888] hover:border-orange-400 hover:text-orange-400 disabled:opacity-50"
                  title="Restore from this backup">
                  <RotateCcw size={11} className="mr-1 inline" /> Restore
                </button>
                <button onClick={() => handleDeleteBackup(b.volid)} disabled={working}
                  className="shrink-0 rounded border border-[#222222] px-2 py-1 text-[10px] text-[#888888] hover:border-red-400 hover:text-red-400 disabled:opacity-50"
                  title="Delete backup">
                  <Trash2 size={11} />
                </button>
              </div>
            ))}
          </div>
        )}

        {/* Schedule tab */}
        {tab === "schedule" && (
          <div className="flex flex-col gap-3">
            {/* Existing schedules */}
            {schedules.length > 0 && (
              <div className="flex flex-col gap-2">
                <div className="text-xs text-[#888888] uppercase tracking-wider">Active Schedules</div>
                {schedules.map((s) => (
                  <div key={s.id} className="flex items-center justify-between rounded-md bg-[#111111] px-3 py-2">
                    <div>
                      <div className="text-xs text-[#e0e0e0]">{s.comment || s.schedule}</div>
                      <div className="text-[10px] text-[#555555] font-mono">{s.schedule}</div>
                    </div>
                    <button onClick={() => handleDeleteSchedule(s.id)} disabled={working}
                      className="rounded border border-[#222222] px-2 py-1 text-[10px] text-[#888888] hover:border-red-400 hover:text-red-400 disabled:opacity-50">
                      <Trash2 size={11} />
                    </button>
                  </div>
                ))}
              </div>
            )}

            {/* Preset buttons */}
            <div className="text-xs text-[#888888] uppercase tracking-wider">Add Schedule</div>
            <div className="flex flex-col gap-2">
              {SCHEDULE_PRESETS.map((preset) => (
                <button key={preset.cron} onClick={() => handleCreateSchedule(preset.cron, preset.comment)} disabled={working}
                  className="flex items-center gap-3 rounded-md border border-[#222222] bg-[#111111] px-4 py-3 text-left transition-colors hover:border-[#00ff88] hover:bg-[#161616] disabled:opacity-50">
                  <Calendar size={14} className="text-[#888888] shrink-0" />
                  <div>
                    <div className="text-sm text-[#e0e0e0]">{preset.label}</div>
                    <div className="text-[10px] text-[#555555] font-mono">{preset.cron}</div>
                  </div>
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Close */}
        <div className="flex justify-end">
          <button onClick={handleClose} className="rounded-md border border-[#222222] px-4 py-2 text-sm text-[#e0e0e0] transition-colors hover:bg-[#222222]">Close</button>
        </div>
      </div>
    </Modal>
  );
}
