"use client";

import { useState, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import { usePolling } from "@/hooks/usePolling";
import { apiFetch, apiPost, apiDelete, formatBytes, formatUptime, cpuPercent, memPercent } from "@/lib/api";
import { ContainerStatus, ContainerConfig, ContainerNetInterface, StorageInfo } from "@/lib/types";
import StatusBadge from "@/components/shared/StatusBadge";
import ConfirmDialog from "@/components/shared/ConfirmDialog";
import DistroIcon from "@/components/shared/DistroIcon";
import ScaleModal from "@/components/shared/ScaleModal";
import ResizeDiskModal from "@/components/shared/ResizeDiskModal";
import AddVolumeModal from "@/components/shared/AddVolumeModal";
import { useJobs } from "@/contexts/JobsContext";
import {
  ArrowLeft, Play, Square, RotateCcw, Trash2,
  Cpu, MemoryStick, HardDrive, Clock, Network,
  KeyRound, Terminal, Eye, EyeOff, Copy, Check,
  ArrowUpDown, ArrowDownUp, Box, SlidersHorizontal, Plus,
} from "lucide-react";

function parseContainerDisks(config: ContainerConfig | null): { name: string; currentSize: string; currentGB: number }[] {
  if (!config) return [];
  const disks: { name: string; currentSize: string; currentGB: number }[] = [];
  for (const [key, value] of Object.entries(config)) {
    if ((key === "rootfs" || /^mp\d+$/.test(key)) && typeof value === "string") {
      const sizeMatch = value.match(/size=(\d+)G/);
      if (sizeMatch) {
        disks.push({ name: key, currentSize: `${sizeMatch[1]}G`, currentGB: parseInt(sizeMatch[1], 10) });
      }
    }
  }
  return disks;
}

export default function ContainerDetailPage() {
  const params = useParams<{ node: string; vmid: string }>();
  const router = useRouter();
  const node = params.node;
  const vmid = parseInt(params.vmid, 10);
  const { getJobByVmid } = useJobs();

  const [actionLoading, setActionLoading] = useState(false);
  const [confirmAction, setConfirmAction] = useState<"stop" | "reboot" | "delete" | null>(null);
  const [showScale, setShowScale] = useState(false);
  const [showResizeDisk, setShowResizeDisk] = useState(false);
  const [showAddVolume, setShowAddVolume] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [copiedField, setCopiedField] = useState<string | null>(null);

  const fetchCT = useCallback(() => apiFetch<ContainerStatus>(`/nodes/${node}/containers/${vmid}/status`), [node, vmid]);
  const fetchConfig = useCallback(async () => {
    try { return await apiFetch<ContainerConfig>(`/nodes/${node}/containers/${vmid}/config`); }
    catch { return null; }
  }, [node, vmid]);
  const fetchInterfaces = useCallback(async () => {
    try { return await apiFetch<ContainerNetInterface[]>(`/nodes/${node}/containers/${vmid}/interfaces`); }
    catch { return []; }
  }, [node, vmid]);
  const fetchStorages = useCallback(async () => {
    try { return await apiFetch<StorageInfo[]>(`/nodes/${node}/storage`); }
    catch { return []; }
  }, [node]);
  const fetchSettings = useCallback(() => apiFetch<{ dns_domain: string }>("/settings"), []);

  const { data: ct, isLoading, refresh } = usePolling(fetchCT, 3000);
  const { data: config } = usePolling(fetchConfig, 10000);
  const { data: interfaces } = usePolling(fetchInterfaces, 5000);
  const { data: storageList } = usePolling(fetchStorages, 30000);
  const { data: settings } = usePolling(fetchSettings, 60000);
  const buildJob = getJobByVmid(vmid);

  const handleAction = async (action: string) => {
    setActionLoading(true);
    setConfirmAction(null);
    try {
      if (action === "delete") {
        await apiDelete(`/nodes/${node}/containers/${vmid}`);
        router.push("/containers");
        return;
      }
      await apiPost(`/nodes/${node}/containers/${vmid}/${action}`);
      setTimeout(refresh, 1500);
    } finally {
      setActionLoading(false);
    }
  };

  const copy = async (text: string, field: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedField(field);
      setTimeout(() => setCopiedField(null), 2000);
    } catch {}
  };

  if (isLoading || !ct) {
    return <div className="flex items-center justify-center py-20"><span className="text-[#888888]">Loading...</span></div>;
  }

  const isRunning = ct.status === "running";
  const cpu = cpuPercent(ct.cpu);
  const mem = memPercent(ct.mem, ct.maxmem);
  const disk = ct.maxdisk > 0 ? memPercent(ct.disk, ct.maxdisk) : 0;

  // Parse IP addresses from runtime interfaces (preferred) or config net0
  const runtimeIps = (interfaces ?? [])
    .filter((i) => i.name !== "lo")
    .flatMap((i) => {
      const ips: string[] = [];
      if (i.inet) {
        // inet can be "192.168.2.100/24"
        const ip = String(i.inet).split("/")[0];
        if (ip && ip !== "127.0.0.1") ips.push(ip);
      }
      return ips;
    });

  // Fallback: parse config net0 ip=... (might be "dhcp" for DHCP-assigned)
  const configIp = (() => {
    if (!config?.net0) return null;
    const match = String(config.net0).match(/ip=([^,]+)/);
    if (!match) return null;
    const ip = match[1].split("/")[0];
    return ip === "dhcp" ? null : ip;
  })();

  const primaryIp = runtimeIps[0] || configIp;
  const sshUser = buildJob?.ciuser || "root";
  const hostname = config?.hostname || ct.name;

  const btnBase = "inline-flex items-center gap-2 rounded-md border px-3 py-1.5 text-xs font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-30";

  return (
    <div className="flex flex-col gap-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <button onClick={() => router.push("/containers")} className="rounded-md p-2 text-[#888888] transition-colors hover:bg-[#222222] hover:text-[#e0e0e0]">
          <ArrowLeft size={18} />
        </button>
        <DistroIcon name={ct.name} size={40} />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-3">
            <h1 className="text-xl font-semibold text-[#e0e0e0] truncate">{ct.name}</h1>
            <StatusBadge status={ct.status} />
            <span className="inline-flex items-center gap-1 rounded border border-[#222222] bg-[#111111] px-1.5 py-0.5 text-[10px] text-[#888888]">
              <Box size={10} /> LXC
            </span>
          </div>
          <div className="flex items-center gap-2 mt-0.5 text-xs text-[#555555]">
            <span className="font-mono">CT {ct.vmid}</span>
            <span>·</span>
            <span>{node}</span>
            {ct.uptime > 0 && <><span>·</span><span>{formatUptime(ct.uptime)}</span></>}
          </div>
        </div>
        <div className="flex items-center gap-1.5 shrink-0">
          <button disabled={isRunning || actionLoading} onClick={() => handleAction("start")} className={`${btnBase} border-[#222222] text-[#e0e0e0] hover:border-[#00ff88] hover:text-[#00ff88]`}><Play size={13} /> Start</button>
          <button disabled={!isRunning || actionLoading} onClick={() => setConfirmAction("stop")} className={`${btnBase} border-[#222222] text-[#e0e0e0] hover:border-red-400 hover:text-red-400`}><Square size={13} /> Stop</button>
          <button disabled={!isRunning || actionLoading} onClick={() => setConfirmAction("reboot")} className={`${btnBase} border-[#222222] text-[#e0e0e0] hover:border-yellow-400 hover:text-yellow-400`}><RotateCcw size={13} /> Reboot</button>
          <div className="w-px h-6 bg-[#222222] mx-1" />
          <button disabled={actionLoading} onClick={() => setShowScale(true)} className={`${btnBase} border-[#222222] text-[#e0e0e0] hover:border-blue-400 hover:text-blue-400`}><SlidersHorizontal size={13} /> Scale</button>
          <button disabled={actionLoading} onClick={() => setShowResizeDisk(true)} className={`${btnBase} border-[#222222] text-[#e0e0e0] hover:border-blue-400 hover:text-blue-400`}><HardDrive size={13} /> Resize</button>
          <button disabled={actionLoading} onClick={() => setShowAddVolume(true)} className={`${btnBase} border-[#222222] text-[#e0e0e0] hover:border-blue-400 hover:text-blue-400`}><Plus size={13} /> Add Volume</button>
          <div className="w-px h-6 bg-[#222222] mx-1" />
          <button disabled={isRunning || actionLoading} onClick={() => setConfirmAction("delete")} className={`${btnBase} border-[#222222] text-[#555555] hover:border-red-500 hover:bg-red-500/10 hover:text-red-400`}><Trash2 size={13} /></button>
        </div>
      </div>



      {/* Two-column layout */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Left: Resources */}
        <div className="lg:col-span-2 flex flex-col gap-5">
          <div className="grid grid-cols-3 gap-4">
            <GaugeCard icon={<Cpu size={15} />} label="CPU" value={`${cpu}%`} detail={`${ct.cpus || 1} core`} percent={cpu} />
            <GaugeCard icon={<MemoryStick size={15} />} label="Memory" value={formatBytes(ct.mem)} detail={`of ${formatBytes(ct.maxmem)}`} percent={mem} />
            <GaugeCard icon={<HardDrive size={15} />} label="Disk" value={formatBytes(ct.disk)} detail={`of ${formatBytes(ct.maxdisk)}`} percent={disk} />
          </div>

          {/* Network + Disk side by side */}
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            {/* Network card */}
            <div className="rounded-lg border border-[#222222] bg-[#161616] p-4">
              <div className="mb-3 flex items-center gap-2 text-xs font-medium text-[#888888] uppercase tracking-wider">
                <Network size={13} /> Network
              </div>

              {/* Interfaces from runtime */}
              {(interfaces ?? []).filter((i) => i.name !== "lo").length > 0 && (
                <div className="flex flex-col gap-2 mb-3">
                  {(interfaces ?? []).filter((i) => i.name !== "lo").map((iface) => {
                    const ip = iface.inet ? String(iface.inet).split("/")[0] : null;
                    return (
                      <div key={iface.name} className="flex items-center gap-3 rounded-md bg-[#111111] px-3 py-2">
                        <span className="font-mono text-xs text-[#555555]">{iface.name}</span>
                        {iface.hwaddr && <span className="font-mono text-[10px] text-[#444444]">{String(iface.hwaddr)}</span>}
                        <div className="flex-1" />
                        {ip && <code className="font-mono text-xs text-[#00ff88]">{ip}</code>}
                      </div>
                    );
                  })}
                </div>
              )}

              {/* I/O */}
              {isRunning && (
                <div className="grid grid-cols-2 gap-2">
                  <IOCard icon={<ArrowDownUp size={11} />} label="In" value={formatBytes(ct.netin)} />
                  <IOCard icon={<ArrowUpDown size={11} />} label="Out" value={formatBytes(ct.netout)} />
                </div>
              )}
            </div>

            {/* Storage card */}
            <div className="rounded-lg border border-[#222222] bg-[#161616] p-4">
              <div className="mb-3 flex items-center gap-2 text-xs font-medium text-[#888888] uppercase tracking-wider">
                <HardDrive size={13} /> Storage
              </div>

              {config?.rootfs && (
                <div className="mb-3 rounded-md bg-[#111111] px-3 py-2">
                  <div className="mb-1 flex items-center justify-between">
                    <span className="font-mono text-xs text-[#e0e0e0]">rootfs</span>
                    <span className="text-[10px] text-[#555555]">{disk}%</span>
                  </div>
                  <div className="flex items-center gap-3">
                    <div className="flex-1 h-1.5 overflow-hidden rounded-full bg-[#222222]">
                      <div className={`h-full rounded-full transition-all ${disk > 80 ? "bg-red-500" : disk > 60 ? "bg-yellow-500" : "bg-[#00ff88]"}`} style={{ width: `${disk}%` }} />
                    </div>
                    <span className="text-[10px] font-mono text-[#888888] shrink-0">{formatBytes(ct.disk)} / {formatBytes(ct.maxdisk)}</span>
                  </div>
                  <div className="mt-1 text-[10px] text-[#555555] font-mono truncate">{String(config.rootfs)}</div>
                </div>
              )}

              {/* LXC doesn't have guest-agent style filesystem stats — show mp0, mp1 etc from config if present */}
              {config && (
                <div className="flex flex-col gap-1.5 mb-3">
                  {Object.entries(config)
                    .filter(([k]) => /^mp\d+$/.test(k))
                    .map(([key, value]) => (
                      <div key={key} className="rounded-md bg-[#111111] px-3 py-2">
                        <div className="flex items-center justify-between mb-1">
                          <span className="font-mono text-xs text-[#e0e0e0]">{key}</span>
                        </div>
                        <div className="text-[10px] text-[#555555] font-mono truncate">{String(value)}</div>
                      </div>
                    ))}
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Right: Details */}
        <div className="flex flex-col gap-5">
          <div className="rounded-lg border border-[#222222] bg-[#161616] p-4">
            <div className="mb-4 text-xs font-medium text-[#888888] uppercase tracking-wider">Details</div>
            <div className="flex flex-col gap-2.5">
              <InfoRow label="Hostname" value={hostname} mono />
              {settings?.dns_domain && (
                <InfoRow label="FQDN" value={`${hostname}.${settings.dns_domain}`} mono copyable={`${hostname}.${settings.dns_domain}`} field="fqdn" copied={copiedField} onCopy={copy} />
              )}
              {settings?.dns_domain && <InfoRow label="DNS Domain" value={settings.dns_domain} />}
              <InfoRow label="Status" value={ct.status} />
              <InfoRow label="IP Address" value={primaryIp || "—"} mono green={!!primaryIp} copyable={primaryIp || undefined} field="ip-detail" copied={copiedField} onCopy={copy} />
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
              <InfoRow label="Type" value="LXC" />
              <InfoRow label="OS" value={config?.ostype ? String(config.ostype) : "—"} />
              <InfoRow label="Unprivileged" value={config?.unprivileged ? "yes" : "no"} />
              <div className="my-1 h-px bg-[#222222]" />
              <InfoRow label="Uptime" value={ct.uptime > 0 ? formatUptime(ct.uptime) : "—"} />
              <InfoRow label="Base Image" value={buildJob ? String(buildJob.source_vmid) : "—"} mono />
              <div className="my-1 h-px bg-[#222222]" />
              <InfoRow label="CTID" value={String(ct.vmid)} mono />
              <InfoRow label="Node" value={node} />
              <InfoRow label="Cores" value={String(ct.cpus || config?.cores || 1)} mono />
              <InfoRow label="RAM" value={formatBytes(ct.maxmem)} mono />
              <InfoRow label="Disk" value={formatBytes(ct.maxdisk)} mono />
            </div>
          </div>

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

      <ConfirmDialog isOpen={confirmAction === "stop"} onClose={() => setConfirmAction(null)} onConfirm={() => handleAction("stop")} title="Confirm Stop" message="Are you sure you want to stop this container?" />
      <ConfirmDialog isOpen={confirmAction === "reboot"} onClose={() => setConfirmAction(null)} onConfirm={() => handleAction("reboot")} title="Confirm Reboot" message="Are you sure you want to reboot this container?" />
      <ConfirmDialog isOpen={confirmAction === "delete"} onClose={() => setConfirmAction(null)} onConfirm={() => handleAction("delete")} title="Delete Container" message={`Are you sure you want to permanently delete container ${ct.vmid} (${ct.name})? This action cannot be undone.`} />

      <ScaleModal isOpen={showScale} onClose={() => setShowScale(false)} node={node} vmid={vmid} vmName={ct.name}
        type="container" currentCores={ct.cpus || 1} currentMemoryBytes={ct.maxmem} isRunning={isRunning} onSuccess={refresh} />

      <ResizeDiskModal isOpen={showResizeDisk} onClose={() => setShowResizeDisk(false)} node={node} vmid={vmid}
        type="container" disks={parseContainerDisks(config)} onSuccess={refresh} />

      <AddVolumeModal isOpen={showAddVolume} onClose={() => setShowAddVolume(false)} node={node} vmid={vmid}
        type="container" storages={storageList ?? []} onSuccess={refresh} />
    </div>
  );
}

/* ── Shared components (duplicated from VM detail for now) ── */

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
