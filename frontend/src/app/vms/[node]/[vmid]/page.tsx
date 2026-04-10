"use client";

import { useState, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import { usePolling } from "@/hooks/usePolling";
import { apiFetch, apiPost, apiDelete, formatBytes, formatUptime, cpuPercent, memPercent } from "@/lib/api";
import { VMStatus, NetworkInterface, TemplateInfo, FilesystemInfo, StorageInfo } from "@/lib/types";
import StatusBadge from "@/components/shared/StatusBadge";
import ConfirmDialog from "@/components/shared/ConfirmDialog";
import DistroIcon, { detectDistro, DISTRO_USERS } from "@/components/shared/DistroIcon";
import ReinstallModal from "@/components/vms/ReinstallModal";
import ScaleModal from "@/components/shared/ScaleModal";
import ResizeDiskModal from "@/components/shared/ResizeDiskModal";
import AddVolumeModal from "@/components/shared/AddVolumeModal";
import BackupModal from "@/components/shared/BackupModal";
import { useJobs } from "@/contexts/JobsContext";
import {
  ArrowLeft, Play, Square, RotateCcw, Trash2, RefreshCw,
  Cpu, MemoryStick, HardDrive, Clock, Network,
  KeyRound, User, Terminal, Eye, EyeOff, Copy, Check,
  ArrowUpDown, ArrowDownUp, SlidersHorizontal, Plus, History,
} from "lucide-react";

function parseDiskInfo(config: Record<string, unknown> | null): { name: string; currentSize: string; currentGB: number }[] {
  if (!config) return [];
  const disks: { name: string; currentSize: string; currentGB: number }[] = [];
  for (const [key, value] of Object.entries(config)) {
    if (/^(scsi|virtio|sata|ide)\d+$/.test(key) && typeof value === "string" && !value.includes("media=cdrom")) {
      const sizeMatch = value.match(/size=(\d+)G/);
      if (sizeMatch) {
        disks.push({ name: key, currentSize: `${sizeMatch[1]}G`, currentGB: parseInt(sizeMatch[1], 10) });
      }
    }
  }
  return disks;
}

export default function VMDetailPage() {
  const params = useParams<{ node: string; vmid: string }>();
  const router = useRouter();
  const node = params.node;
  const vmid = parseInt(params.vmid, 10);
  const { getJobByVmid } = useJobs();

  const [actionLoading, setActionLoading] = useState(false);
  const [confirmAction, setConfirmAction] = useState<"stop" | "reboot" | "delete" | null>(null);
  const [showReinstall, setShowReinstall] = useState(false);
  const [showScale, setShowScale] = useState(false);
  const [showResizeDisk, setShowResizeDisk] = useState(false);
  const [showAddDisk, setShowAddDisk] = useState(false);
  const [showBackups, setShowBackups] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [copiedField, setCopiedField] = useState<string | null>(null);

  const fetchVM = useCallback(() => apiFetch<VMStatus>(`/nodes/${node}/vms/${vmid}/status`), [node, vmid]);
  const fetchInterfaces = useCallback(async () => {
    try { return await apiFetch<NetworkInterface[]>(`/nodes/${node}/vms/${vmid}/interfaces`); }
    catch { return []; }
  }, [node, vmid]);
  const fetchFilesystems = useCallback(async () => {
    try { return await apiFetch<FilesystemInfo[]>(`/nodes/${node}/vms/${vmid}/filesystems`); }
    catch { return []; }
  }, [node, vmid]);
  const fetchVMConfig = useCallback(async () => {
    try { return await apiFetch<Record<string, unknown>>(`/nodes/${node}/vms/${vmid}/config`); }
    catch { return null; }
  }, [node, vmid]);
  const fetchStorages = useCallback(async () => {
    try { return await apiFetch<StorageInfo[]>(`/nodes/${node}/storage`); }
    catch { return []; }
  }, [node]);
  const fetchTemplates = useCallback(() => apiFetch<TemplateInfo[]>("/templates"), []);
  const fetchSettings = useCallback(() => apiFetch<{ dns_domain: string }>("/settings"), []);

  const { data: vm, isLoading, refresh } = usePolling(fetchVM, 3000);
  const { data: interfaces } = usePolling(fetchInterfaces, 5000);
  const { data: filesystems } = usePolling(fetchFilesystems, 5000);
  const { data: vmConfig } = usePolling(fetchVMConfig, 10000);
  const { data: storageList } = usePolling(fetchStorages, 30000);
  const { data: templates } = usePolling(fetchTemplates, 60000);
  const { data: settings } = usePolling(fetchSettings, 60000);
  const buildJob = getJobByVmid(vmid);

  const handleAction = async (action: string) => {
    setActionLoading(true);
    setConfirmAction(null);
    try {
      if (action === "delete") { await apiDelete(`/nodes/${node}/vms/${vmid}`); router.push("/vms"); return; }
      await apiPost(`/nodes/${node}/vms/${vmid}/${action}`);
      setTimeout(refresh, 1500);
    } finally { setActionLoading(false); }
  };

  const copy = async (text: string, field: string) => {
    try { await navigator.clipboard.writeText(text); setCopiedField(field); setTimeout(() => setCopiedField(null), 2000); } catch {}
  };

  if (isLoading || !vm) return <div className="flex items-center justify-center py-20"><span className="text-[#888888]">Loading...</span></div>;

  const isRunning = vm.status === "running";
  const cpu = cpuPercent(vm.cpu);
  const mem = memPercent(vm.mem, vm.maxmem);

  // Use real filesystem data from guest agent if available, fallback to Proxmox API
  const rootFs = (filesystems ?? []).find((f) => f.mountpoint === "/");
  const diskUsed = rootFs?.["used-bytes"] ?? vm.disk;
  const diskTotal = rootFs?.["total-bytes"] ?? vm.maxdisk;
  const disk = diskTotal > 0 ? memPercent(diskUsed, diskTotal) : 0;
  const ipAddresses = (interfaces ?? []).filter((i) => i.name !== "lo")
    .flatMap((i) => (i["ip-addresses"] || []).filter((a) => a["ip-address-type"] === "ipv4" && a["ip-address"] !== "127.0.0.1").map((a) => a["ip-address"]));
  const primaryIp = ipAddresses[0];
  const sshUser = buildJob?.ciuser || DISTRO_USERS[detectDistro(vm.name)];

  const btnBase = "inline-flex items-center gap-2 rounded-md border px-3 py-1.5 text-xs font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-30";

  return (
    <div className="flex flex-col gap-6">
      {/* Header bar */}
      <div className="flex items-center gap-4">
        <button onClick={() => router.push("/vms")} className="rounded-md p-2 text-[#888888] transition-colors hover:bg-[#222222] hover:text-[#e0e0e0]">
          <ArrowLeft size={18} />
        </button>
        <DistroIcon name={vm.name} size={40} />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-3">
            <h1 className="text-xl font-semibold text-[#e0e0e0] truncate">{vm.name}</h1>
            <StatusBadge status={vm.status} />
          </div>
          <div className="flex items-center gap-2 mt-0.5 text-xs text-[#555555]">
            <span className="font-mono">ID {vm.vmid}</span>
            <span>·</span>
            <span>{node}</span>
            {vm.uptime > 0 && <><span>·</span><span>{formatUptime(vm.uptime)}</span></>}
          </div>
        </div>
        <div className="flex items-center gap-1.5 shrink-0">
          <button disabled={isRunning || actionLoading} onClick={() => handleAction("start")} className={`${btnBase} border-[#222222] text-[#e0e0e0] hover:border-[#00ff88] hover:text-[#00ff88]`}><Play size={13} /> Start</button>
          <button disabled={!isRunning || actionLoading} onClick={() => setConfirmAction("stop")} className={`${btnBase} border-[#222222] text-[#e0e0e0] hover:border-red-400 hover:text-red-400`}><Square size={13} /> Stop</button>
          <button disabled={!isRunning || actionLoading} onClick={() => setConfirmAction("reboot")} className={`${btnBase} border-[#222222] text-[#e0e0e0] hover:border-yellow-400 hover:text-yellow-400`}><RotateCcw size={13} /> Reboot</button>
          <button disabled={actionLoading} onClick={() => setShowReinstall(true)} className={`${btnBase} border-[#222222] text-[#e0e0e0] hover:border-orange-400 hover:text-orange-400`}><RefreshCw size={13} /> Reinstall</button>
          <div className="w-px h-6 bg-[#222222] mx-1" />
          <button disabled={actionLoading} onClick={() => setShowScale(true)} className={`${btnBase} border-[#222222] text-[#e0e0e0] hover:border-blue-400 hover:text-blue-400`}><SlidersHorizontal size={13} /> Scale</button>
          <button disabled={actionLoading} onClick={() => setShowResizeDisk(true)} className={`${btnBase} border-[#222222] text-[#e0e0e0] hover:border-blue-400 hover:text-blue-400`}><HardDrive size={13} /> Resize</button>
          <button disabled={actionLoading} onClick={() => setShowAddDisk(true)} className={`${btnBase} border-[#222222] text-[#e0e0e0] hover:border-blue-400 hover:text-blue-400`}><Plus size={13} /> Add Disk</button>
          <div className="w-px h-6 bg-[#222222] mx-1" />
          <button disabled={actionLoading} onClick={() => setShowBackups(true)} className={`${btnBase} border-[#222222] text-[#e0e0e0] hover:border-purple-400 hover:text-purple-400`}><History size={13} /> Backups</button>
          <div className="w-px h-6 bg-[#222222] mx-1" />
          <button disabled={isRunning || actionLoading} onClick={() => setConfirmAction("delete")} className={`${btnBase} border-[#222222] text-[#555555] hover:border-red-500 hover:bg-red-500/10 hover:text-red-400`}><Trash2 size={13} /></button>
        </div>
      </div>



      {/* Two-column layout */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Left: Resources (2 cols wide) */}
        <div className="lg:col-span-2 flex flex-col gap-5">
          {/* Resource gauges — CPU, Memory, Disk */}
          <div className="grid grid-cols-3 gap-4">
            <GaugeCard icon={<Cpu size={15} />} label="CPU" value={`${cpu}%`} detail={`${vm.cpus} vCPU`} percent={cpu} />
            <GaugeCard icon={<MemoryStick size={15} />} label="Memory" value={formatBytes(vm.mem)} detail={`of ${formatBytes(vm.maxmem)}`} percent={mem} />
            <GaugeCard icon={<HardDrive size={15} />} label="Disk" value={formatBytes(diskUsed)} detail={`of ${formatBytes(diskTotal)}`} percent={disk} />
          </div>

          {/* Network + Disk side by side */}
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            {/* Network card */}
            <div className="rounded-lg border border-[#222222] bg-[#161616] p-4">
              <div className="mb-3 flex items-center gap-2 text-xs font-medium text-[#888888] uppercase tracking-wider">
                <Network size={13} /> Network
              </div>

              {/* Interfaces */}
              {(interfaces ?? []).filter((i) => i.name !== "lo").length > 0 && (
                <div className="flex flex-col gap-2 mb-3">
                  {(interfaces ?? []).filter((i) => i.name !== "lo").map((iface) => (
                    <div key={iface.name} className="flex items-center gap-3 rounded-md bg-[#111111] px-3 py-2">
                      <span className="font-mono text-xs text-[#555555]">{iface.name}</span>
                      <span className="font-mono text-[10px] text-[#444444]">{iface["hardware-address"]}</span>
                      <div className="flex-1" />
                      {(iface["ip-addresses"] || [])
                        .filter((a) => a["ip-address-type"] === "ipv4" && a["ip-address"] !== "127.0.0.1")
                        .map((addr) => (
                          <code key={addr["ip-address"]} className="font-mono text-xs text-[#00ff88]">{addr["ip-address"]}</code>
                        ))}
                    </div>
                  ))}
                </div>
              )}

              {/* I/O */}
              {isRunning && (
                <div className="grid grid-cols-2 gap-2">
                  <IOCard icon={<ArrowDownUp size={11} />} label="In" value={formatBytes(vm.netin)} />
                  <IOCard icon={<ArrowUpDown size={11} />} label="Out" value={formatBytes(vm.netout)} />
                </div>
              )}
            </div>

            {/* Disk / Filesystems card */}
            <div className="rounded-lg border border-[#222222] bg-[#161616] p-4">
              <div className="mb-3 flex items-center gap-2 text-xs font-medium text-[#888888] uppercase tracking-wider">
                <HardDrive size={13} /> Disk
              </div>

              {/* Filesystems */}
              {(filesystems ?? []).filter((f) => f["total-bytes"] && f["total-bytes"] > 0).length > 0 ? (
                <div className="flex flex-col gap-2 mb-3">
                  {(filesystems ?? []).filter((f) => f["total-bytes"] && f["total-bytes"] > 0).map((fs) => {
                    const used = fs["used-bytes"] || 0;
                    const total = fs["total-bytes"] || 1;
                    const pct = Math.round((used / total) * 100);
                    return (
                      <div key={fs.mountpoint} className="rounded-md bg-[#111111] px-3 py-2">
                        <div className="flex items-center justify-between mb-1">
                          <span className="font-mono text-xs text-[#e0e0e0]">{fs.mountpoint}</span>
                          <span className="text-[10px] text-[#555555]">{fs.type} · {pct}%</span>
                        </div>
                        <div className="flex items-center gap-3">
                          <div className="flex-1 h-1.5 overflow-hidden rounded-full bg-[#222222]">
                            <div className={`h-full rounded-full transition-all ${pct > 80 ? "bg-red-500" : pct > 60 ? "bg-yellow-500" : "bg-[#00ff88]"}`} style={{ width: `${pct}%` }} />
                          </div>
                          <span className="text-[10px] font-mono text-[#888888] shrink-0">{formatBytes(used)} / {formatBytes(total)}</span>
                        </div>
                      </div>
                    );
                  })}
                </div>
              ) : (
                <div className="text-xs text-[#555555] mb-3">
                  {formatBytes(vm.maxdisk)} allocated
                </div>
              )}

              {/* Disk I/O */}
              {isRunning && (
                <div className="grid grid-cols-2 gap-2">
                  <IOCard icon={<HardDrive size={11} />} label="Read" value={formatBytes(vm.diskread)} />
                  <IOCard icon={<HardDrive size={11} />} label="Write" value={formatBytes(vm.diskwrite)} />
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Right sidebar: Access & Build info */}
        <div className="flex flex-col gap-5">
          {/* VM Details card */}
          <div className="rounded-lg border border-[#222222] bg-[#161616] p-4">
            <div className="mb-4 text-xs font-medium text-[#888888] uppercase tracking-wider">Details</div>
            <div className="flex flex-col gap-2.5">
              <InfoRow label="Hostname" value={vm.name} mono />
              {settings?.dns_domain && (
                <InfoRow label="FQDN" value={`${vm.name}.${settings.dns_domain}`} mono copyable={`${vm.name}.${settings.dns_domain}`} field="fqdn" copied={copiedField} onCopy={copy} />
              )}
              {settings?.dns_domain && (
                <InfoRow label="DNS Domain" value={settings.dns_domain} />
              )}
              <InfoRow label="Status" value={vm.status} />
              <InfoRow label="IP Address" value={primaryIp || "—"} mono green={!!primaryIp} copyable={primaryIp} field="ip-detail" copied={copiedField} onCopy={copy} />
              <InfoRow label="SSH User" value={sshUser} mono />
              {buildJob?.password && (
                <div className="flex items-center justify-between">
                  <span className="text-xs text-[#888888]">Password</span>
                  <div className="flex items-center gap-1.5">
                    <code className="font-mono text-xs text-[#e0e0e0]">{showPassword ? buildJob.password : "••••••••"}</code>
                    <button onClick={() => setShowPassword(!showPassword)} className="rounded p-0.5 text-[#555555] hover:text-[#e0e0e0] transition-colors">
                      {showPassword ? <EyeOff size={12} /> : <Eye size={12} />}
                    </button>
                    <CopyBtn text={buildJob.password} field="pw" copied={copiedField} onCopy={copy} small />
                  </div>
                </div>
              )}
              <div className="my-1 h-px bg-[#222222]" />
              <InfoRow label="Uptime" value={vm.uptime > 0 ? formatUptime(vm.uptime) : "—"} />
              <InfoRow label="Base Image" value={buildJob ? `${buildJob.source_vmid}` : "—"} mono />
              <div className="my-1 h-px bg-[#222222]" />
              <InfoRow label="VMID" value={String(vm.vmid)} mono />
              <InfoRow label="Node" value={node} />
              <InfoRow label="vCPU" value={String(vm.cpus)} mono />
              <InfoRow label="RAM" value={formatBytes(vm.maxmem)} mono />
              <InfoRow label="Disk" value={formatBytes(vm.maxdisk)} mono />
            </div>
          </div>

          {/* SSH Key card */}
          {buildJob?.sshkey && (
            <div className="rounded-lg border border-[#222222] bg-[#161616] p-4">
              <div className="mb-2 flex items-center justify-between">
                <span className="text-xs font-medium text-[#888888] uppercase tracking-wider">SSH Key</span>
                <CopyBtn text={buildJob.sshkey} field="sshkey" copied={copiedField} onCopy={copy} small />
              </div>
              <code className="block break-all rounded-md bg-[#0a0a0a] px-3 py-2 font-mono text-[10px] leading-relaxed text-[#666666]">
                {buildJob.sshkey}
              </code>
            </div>
          )}
        </div>
      </div>

      {/* Dialogs */}
      <ConfirmDialog isOpen={confirmAction === "stop"} onClose={() => setConfirmAction(null)} onConfirm={() => handleAction("stop")} title="Confirm Stop" message="Are you sure you want to stop this VM? Any unsaved data may be lost." />
      <ConfirmDialog isOpen={confirmAction === "reboot"} onClose={() => setConfirmAction(null)} onConfirm={() => handleAction("reboot")} title="Confirm Reboot" message="Are you sure you want to reboot this VM?" />
      <ConfirmDialog isOpen={confirmAction === "delete"} onClose={() => setConfirmAction(null)} onConfirm={() => handleAction("delete")} title="Delete VM" message={`Are you sure you want to permanently delete VM ${vm.vmid} (${vm.name})? This action cannot be undone.`} />
      <ReinstallModal isOpen={showReinstall} onClose={() => setShowReinstall(false)} vmid={vmid} vmName={vm.name} node={node} currentDistroHint={vm.name}
        templates={(templates ?? []).filter((t) => t.vmtype === "qemu")} buildPassword={buildJob?.password} buildSshKey={buildJob?.sshkey} buildCiUser={buildJob?.ciuser} onSuccess={refresh} />

      <ScaleModal isOpen={showScale} onClose={() => setShowScale(false)} node={node} vmid={vmid} vmName={vm.name}
        type="vm" currentCores={vm.cpus} currentMemoryBytes={vm.maxmem} isRunning={isRunning} onSuccess={refresh} />

      <ResizeDiskModal isOpen={showResizeDisk} onClose={() => setShowResizeDisk(false)} node={node} vmid={vmid}
        type="vm" disks={parseDiskInfo(vmConfig)} onSuccess={refresh} />

      <AddVolumeModal isOpen={showAddDisk} onClose={() => setShowAddDisk(false)} node={node} vmid={vmid}
        type="vm" storages={storageList ?? []} onSuccess={refresh} />

      <BackupModal isOpen={showBackups} onClose={() => setShowBackups(false)} node={node} vmid={vmid}
        vmName={vm.name} type="vm" onSuccess={refresh} />
    </div>
  );
}

/* ── Shared components ── */

function CopyBtn({ text, field, copied, onCopy, small }: { text: string; field: string; copied: string | null; onCopy: (t: string, f: string) => void; small?: boolean }) {
  const done = copied === field;
  return (
    <button onClick={() => onCopy(text, field)}
      className={`inline-flex shrink-0 items-center gap-1 rounded border border-[#222222] text-[#666666] transition-colors hover:border-[#00ff88] hover:text-[#00ff88] ${small ? "px-1 py-0.5 text-[9px]" : "px-2 py-1 text-[10px]"}`}>
      {done ? <Check size={small ? 9 : 10} /> : <Copy size={small ? 9 : 10} />}
      {done ? "Copied" : "Copy"}
    </button>
  );
}

function InfoRow({ label, value, mono, green, copyable, field, copied, onCopy }: {
  label: string; value: string; mono?: boolean; green?: boolean;
  copyable?: string; field?: string; copied?: string | null; onCopy?: (t: string, f: string) => void;
}) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-xs text-[#888888]">{label}</span>
      <div className="flex items-center gap-1.5">
        <span className={`text-xs ${mono ? "font-mono" : ""} ${green ? "text-[#00ff88]" : "text-[#e0e0e0]"}`}>{value}</span>
        {copyable && field && onCopy && copied !== undefined && (
          <CopyBtn text={copyable} field={field} copied={copied} onCopy={onCopy} small />
        )}
      </div>
    </div>
  );
}

function GaugeCard({ icon, label, value, detail, percent }: { icon: React.ReactNode; label: string; value: string; detail: string; percent?: number }) {
  return (
    <div className="rounded-lg border border-[#222222] bg-[#161616] p-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-xs text-[#888888]">{icon} {label}</div>
        {percent !== undefined && <span className="text-[10px] font-mono text-[#555555]">{percent}%</span>}
      </div>
      <div className="mt-2 text-xl font-bold text-[#e0e0e0] font-mono leading-none">{value}</div>
      <div className="mt-1 text-[10px] text-[#555555]">{detail}</div>
      {percent !== undefined && (
        <div className="mt-3 h-1.5 w-full overflow-hidden rounded-full bg-[#222222]">
          <div className={`h-full rounded-full transition-all duration-700 ${percent > 80 ? "bg-red-500" : percent > 60 ? "bg-yellow-500" : "bg-[#00ff88]"}`} style={{ width: `${Math.min(percent, 100)}%` }} />
        </div>
      )}
    </div>
  );
}

function IOCard({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) {
  return (
    <div className="rounded-md border border-[#1a1a1a] bg-[#131313] px-3 py-2.5">
      <div className="flex items-center gap-1.5 text-[10px] text-[#555555]">{icon} {label}</div>
      <div className="mt-1 font-mono text-xs text-[#e0e0e0]">{value}</div>
    </div>
  );
}
