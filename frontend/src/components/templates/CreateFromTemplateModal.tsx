"use client";

import { useState, useEffect, useCallback } from "react";
import Modal from "@/components/shared/Modal";
import { apiPost, apiFetch, formatBytes } from "@/lib/api";
import { TemplateInfo, JobStep, StorageInfo } from "@/lib/types";
import { useJobs } from "@/contexts/JobsContext";
import { detectDistro, DISTRO_USERS } from "@/components/shared/DistroIcon";
import TemplateSelect from "@/components/shared/TemplateSelect";
import { Terminal, Copy, Check, Plus, X, HardDrive } from "lucide-react";

interface CreateFromTemplateModalProps {
  isOpen: boolean;
  onClose: () => void;
  templates: TemplateInfo[];
  nodes: string[];
  onSuccess: () => void;
}

interface ExtraVolume {
  storage: string;
  size: string;
}

const STEP_PROGRESS: Record<string, number> = {
  "": 0, cloning: 10, configuring: 30, resizing: 45, adding_disks: 55, starting: 70, waiting_for_running: 85, ready: 100,
};

const STEP_LABELS: Record<string, string> = {
  "": "Queued...", cloning: "Cloning template...", configuring: "Configuring cloud-init...",
  resizing: "Resizing disk...", adding_disks: "Adding extra volumes...", starting: "Starting VM...",
  waiting_for_running: "Waiting for VM to come online...", ready: "Ready!",
};

const STEP_LIST: { key: JobStep | "adding_disks"; label: string }[] = [
  { key: "cloning", label: "Clone template" }, { key: "configuring", label: "Configure cloud-init" },
  { key: "resizing", label: "Resize disk" }, { key: "adding_disks", label: "Add extra volumes" },
  { key: "starting", label: "Start VM" }, { key: "waiting_for_running", label: "Wait for running" },
  { key: "ready", label: "Ready" },
];

const VOLUME_SIZES = [10, 20, 30, 50, 100, 150, 200, 300, 500, 1000];

function generatePassword(): string {
  const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%&*";
  let password = "";
  const array = new Uint32Array(16);
  crypto.getRandomValues(array);
  for (let i = 0; i < 16; i++) {
    password += chars[array[i] % chars.length];
  }
  return password;
}

export default function CreateFromTemplateModal({
  isOpen, onClose, templates, nodes, onSuccess,
}: CreateFromTemplateModalProps) {
  const { addJob, getJob } = useJobs();

  const [selectedTemplateId, setSelectedTemplateId] = useState("");
  const [name, setName] = useState("");
  const [targetNode, setTargetNode] = useState("");
  const [storage, setStorage] = useState("");
  const [ciUser, setCiUser] = useState("");
  const [cores, setCores] = useState("2");
  const [memory, setMemory] = useState("2048");
  const [passwordMode, setPasswordMode] = useState<"set" | "generate">("set");
  const [password, setPassword] = useState("");
  const [generatedPassword, setGeneratedPassword] = useState("");
  const [sshKey, setSshKey] = useState("");
  const [diskSize, setDiskSize] = useState("30");
  const [extraVolumes, setExtraVolumes] = useState<ExtraVolume[]>([]);
  const [userData, setUserData] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [copiedField, setCopiedField] = useState<string | null>(null);
  const [activeJobId, setActiveJobId] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [nextVmid, setNextVmid] = useState<number | null>(null);
  const [storages, setStorages] = useState<StorageInfo[]>([]);

  const selectedTemplate = templates.find(
    (t) => `${t.node}-${t.vmid}` === selectedTemplateId
  );

  useEffect(() => {
    if (!isOpen) return;
    apiFetch<{ vmid: number }>("/next-vmid").then((r) => setNextVmid(r.vmid)).catch(() => {});
  }, [isOpen]);

  useEffect(() => {
    const node = selectedTemplate?.node;
    if (!node) return;
    apiFetch<StorageInfo[]>(`/nodes/${node}/storage`).then((list) => {
      const valid = list.filter(
        (s) => s.storage !== "local" && s.enabled && s.content.includes("images")
      );
      setStorages(valid);
      if (valid.length > 0 && !storage) {
        setStorage(valid[0].storage);
      }
    }).catch(() => setStorages([]));
  }, [selectedTemplate?.node]);

  useEffect(() => {
    if (selectedTemplate?.name) {
      const distro = detectDistro(selectedTemplate.name);
      setCiUser(DISTRO_USERS[distro]);
    }
  }, [selectedTemplate?.name]);

  const activeJob = activeJobId ? getJob(activeJobId) : null;
  const showProgress = activeJob != null;
  const isTerminal = activeJob?.status === "completed" || activeJob?.status === "failed";

  useEffect(() => {
    if (activeJob?.status === "completed") onSuccess();
  }, [activeJob?.status, onSuccess]);

  const addExtraVolume = () => {
    setExtraVolumes((prev) => [...prev, { storage: storages[0]?.storage || "", size: "50" }]);
  };

  const removeExtraVolume = (index: number) => {
    setExtraVolumes((prev) => prev.filter((_, i) => i !== index));
  };

  const updateExtraVolume = (index: number, field: keyof ExtraVolume, value: string) => {
    setExtraVolumes((prev) => prev.map((v, i) => i === index ? { ...v, [field]: value } : v));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedTemplate || !nextVmid || !name) return;

    setError(null);
    setSubmitting(true);

    const typePrefix = selectedTemplate.vmtype === "qemu" ? "vms" : "containers";
    const effectivePassword = passwordMode === "generate" ? generatedPassword : password;

    const extraVols = extraVolumes
      .filter((v) => v.storage && parseInt(v.size, 10) > 0)
      .map((v) => ({ storage: v.storage, size_gb: parseInt(v.size, 10) }));

    try {
      const response = await apiPost<{ job_id: string; vmid: number; node: string }>(
        `/nodes/${selectedTemplate.node}/${typePrefix}/${selectedTemplate.vmid}/provision`,
        {
          newid: nextVmid,
          name,
          full: true,
          target: targetNode || undefined,
          storage: storage || undefined,
          ciuser: ciUser || undefined,
          password: effectivePassword || undefined,
          sshkeys: sshKey || undefined,
          disk_size: parseInt(diskSize, 10),
          extra_volumes: extraVols.length > 0 ? extraVols : undefined,
          user_data: userData || undefined,
        }
      );

      addJob({
        id: response.job_id,
        type: selectedTemplate.vmtype === "qemu" ? "vm" : "container",
        status: "pending", step: "", progress: 0,
        created_at: new Date().toISOString(), updated_at: new Date().toISOString(),
        source_node: selectedTemplate.node, source_vmid: selectedTemplate.vmid!,
        target_node: response.node, new_vmid: response.vmid, name,
        ciuser: ciUser || undefined, disk_size: parseInt(diskSize, 10), full_clone: true,
        password: effectivePassword || undefined, sshkey: sshKey || undefined,
      });

      setActiveJobId(response.job_id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create from template");
    } finally {
      setSubmitting(false);
    }
  };

  const handleGeneratePassword = () => {
    const pwd = generatePassword();
    setGeneratedPassword(pwd);
    setPasswordMode("generate");
  };

  const copyToClipboard = async (text: string, field: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedField(field);
      setTimeout(() => setCopiedField(null), 2000);
    } catch {}
  };

  const resetForm = useCallback(() => {
    setSelectedTemplateId(""); setName(""); setTargetNode(""); setStorage(""); setCiUser("");
    setCores("2"); setMemory("2048"); setPasswordMode("set"); setPassword(""); setGeneratedPassword("");
    setSshKey(""); setDiskSize("30"); setExtraVolumes([]); setUserData(""); setError(null); setActiveJobId(null);
    setSubmitting(false); setNextVmid(null); setStorages([]);
  }, []);

  const handleClose = () => { if (isTerminal) resetForm(); onClose(); };
  const handleDone = () => { resetForm(); onClose(); };

  const effectivePassword = activeJob?.password || (passwordMode === "generate" ? generatedPassword : password);
  const currentStep = activeJob?.step || "";
  const currentProgress = STEP_PROGRESS[currentStep] ?? 0;
  const sshUser = activeJob?.ciuser || ciUser || "root";

  const inputClass = "w-full rounded-md border border-[#222222] bg-[#0a0a0a] px-3 py-2 text-sm text-[#e0e0e0] outline-none focus:border-[#00ff88]";

  function CopyBtn({ text, field, small }: { text: string; field: string; small?: boolean }) {
    const copied = copiedField === field;
    return (
      <button type="button" onClick={() => copyToClipboard(text, field)}
        className={`inline-flex items-center gap-1 rounded border border-[#222222] text-[#888888] transition-colors hover:border-[#00ff88] hover:text-[#00ff88] ${small ? "px-1.5 py-0.5 text-[10px]" : "px-2 py-1 text-xs"}`}>
        {copied ? <Check size={small ? 10 : 12} /> : <Copy size={small ? 10 : 12} />}
        {copied ? "Copied" : "Copy"}
      </button>
    );
  }

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title="Create from Template">
      {!showProgress ? (
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          {error && <div className="rounded border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">{error}</div>}

          <div className="flex flex-col gap-1">
            <span className="text-xs text-[#888888]">Image</span>
            <TemplateSelect templates={templates} value={selectedTemplateId} onChange={setSelectedTemplateId} />
          </div>

          <div className="flex flex-col gap-1">
            <span className="text-xs text-[#888888]">Name</span>
            <input type="text" required value={name} onChange={(e) => setName(e.target.value)} className={inputClass} placeholder="my-new-vm" />
          </div>

          {nodes.length > 1 && (
            <label className="flex flex-col gap-1">
              <span className="text-xs text-[#888888]">Target Node</span>
              <select value={targetNode} onChange={(e) => setTargetNode(e.target.value)} className={inputClass}>
                <option value="">Same as template</option>
                {nodes.map((n) => (<option key={n} value={n}>{n}</option>))}
              </select>
            </label>
          )}

          <div className="grid grid-cols-2 gap-3">
            <label className="flex flex-col gap-1">
              <span className="text-xs text-[#888888]">Cores</span>
              <input type="number" min="1" value={cores} onChange={(e) => setCores(e.target.value)} className={inputClass} />
            </label>
            <label className="flex flex-col gap-1">
              <span className="text-xs text-[#888888]">Memory (MB)</span>
              <input type="number" min="128" step="128" value={memory} onChange={(e) => setMemory(e.target.value)} className={inputClass} />
            </label>
          </div>

          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between">
              <span className="text-xs text-[#888888]">OS Disk</span>
              <span className="text-xs font-mono text-[#e0e0e0]">{diskSize} GB</span>
            </div>
            <input
              type="range"
              min="15"
              max="500"
              step="5"
              value={diskSize}
              onChange={(e) => setDiskSize(e.target.value)}
              className="w-full accent-[#00ff88]"
            />
            <div className="flex justify-between text-[10px] text-[#555555]">
              <span>15 GB</span>
              <span>500 GB</span>
            </div>
          </div>

          {/* Storage section */}
          <div className="flex flex-col gap-3 rounded-lg border border-[#222222] bg-[#111111] p-3">
            <div className="flex items-center gap-2 text-xs text-[#888888]">
              <HardDrive size={12} /> Storage &amp; Volumes
            </div>

            {storages.length === 0 ? (
              <div className="text-xs text-[#555555]">Select a template to configure storage</div>
            ) : (
              <>
                {/* OS volume storage */}
                <div className="flex items-center gap-2">
                  <span className="text-xs text-[#555555] w-16 shrink-0">OS Disk</span>
                  <select value={storage} onChange={(e) => setStorage(e.target.value)} className={inputClass}>
                    {storages.map((s) => (
                      <option key={s.storage} value={s.storage}>{s.storage} — {formatBytes(s.avail)} free</option>
                    ))}
                  </select>
                </div>

                {/* Extra volumes */}
                {extraVolumes.map((vol, i) => (
                  <div key={i} className="flex items-center gap-2">
                    <span className="text-xs text-[#555555] w-16 shrink-0">Data {i + 1}</span>
                    <select
                      value={vol.storage}
                      onChange={(e) => updateExtraVolume(i, "storage", e.target.value)}
                      className={inputClass}
                    >
                      {storages.map((s) => (
                        <option key={s.storage} value={s.storage}>{s.storage} — {formatBytes(s.avail)} free</option>
                      ))}
                    </select>
                    <select
                      value={vol.size}
                      onChange={(e) => updateExtraVolume(i, "size", e.target.value)}
                      className="w-24 shrink-0 rounded-md border border-[#222222] bg-[#0a0a0a] px-2 py-2 text-sm text-[#e0e0e0] outline-none focus:border-[#00ff88]"
                    >
                      {VOLUME_SIZES.map((s) => (<option key={s} value={s}>{s} GB</option>))}
                    </select>
                    <button type="button" onClick={() => removeExtraVolume(i)} className="rounded p-1 text-[#555555] hover:text-red-400 transition-colors">
                      <X size={14} />
                    </button>
                  </div>
                ))}

                <button
                  type="button"
                  onClick={addExtraVolume}
                  className="flex items-center gap-1.5 self-start rounded-md border border-dashed border-[#333333] px-3 py-1.5 text-xs text-[#888888] transition-colors hover:border-[#00ff88] hover:text-[#00ff88]"
                >
                  <Plus size={12} /> Add data volume
                </button>
              </>
            )}
          </div>

          <label className="flex flex-col gap-1">
            <span className="text-xs text-[#888888]">SSH User</span>
            <input type="text" value={ciUser} onChange={(e) => setCiUser(e.target.value)} className={inputClass} placeholder="auto-detected from template" />
          </label>

          <div className="flex flex-col gap-2">
            <span className="text-xs text-[#888888]">Password</span>
            <div className="flex gap-3">
              <label className="flex items-center gap-1.5">
                <input type="radio" name="passwordMode" checked={passwordMode === "set"} onChange={() => setPasswordMode("set")} className="accent-[#00ff88]" />
                <span className="text-sm text-[#e0e0e0]">Set password</span>
              </label>
              <label className="flex items-center gap-1.5">
                <input type="radio" name="passwordMode" checked={passwordMode === "generate"} onChange={() => setPasswordMode("generate")} className="accent-[#00ff88]" />
                <span className="text-sm text-[#e0e0e0]">Generate random</span>
              </label>
            </div>
            {passwordMode === "set" && <input type="text" value={password} onChange={(e) => setPassword(e.target.value)} className={inputClass} placeholder="Enter password" />}
            {passwordMode === "generate" && (
              <div className="flex flex-col gap-2">
                <button type="button" onClick={handleGeneratePassword} className="w-fit rounded-md border border-[#222222] px-3 py-1.5 text-xs text-[#e0e0e0] transition-colors hover:border-[#00ff88] hover:text-[#00ff88]">Generate Password</button>
                {generatedPassword && (
                  <div className="flex items-center gap-2">
                    <code className="flex-1 rounded border border-[#222222] bg-[#0a0a0a] px-3 py-2 font-mono text-sm text-[#00ff88]">{generatedPassword}</code>
                    <CopyBtn text={generatedPassword} field="gen-password" />
                  </div>
                )}
              </div>
            )}
          </div>

          <label className="flex flex-col gap-1">
            <span className="text-xs text-[#888888]">SSH Public Key</span>
            <textarea value={sshKey} onChange={(e) => setSshKey(e.target.value)} className={`${inputClass} min-h-[60px] resize-y font-mono text-xs`} placeholder="ssh-ed25519 AAAA... user@host" rows={2} />
            <span className="text-[10px] text-[#555555]">Recommended: use an SSH key instead of a password for more secure access.</span>
          </label>

          {/* User Data / cloud-init script */}
          <label className="flex flex-col gap-1">
            <span className="text-xs text-[#888888]">User Data (optional)</span>
            <textarea
              value={userData}
              onChange={(e) => setUserData(e.target.value)}
              className={`${inputClass} min-h-[80px] resize-y font-mono text-xs`}
              placeholder={"#cloud-config\nruncmd:\n  - apt-get update\n  - apt-get install -y docker.io"}
              rows={4}
            />
            <span className="text-[10px] text-[#555555]">Cloud-init script that runs on first boot. Use <code className="text-[#888888]">#cloud-config</code> YAML or a <code className="text-[#888888]">#!/bin/bash</code> shell script.</span>
          </label>

          <div className="mt-2 flex justify-end gap-3">
            <button type="button" onClick={handleClose} className="rounded-md border border-[#222222] px-4 py-2 text-sm text-[#e0e0e0] transition-colors hover:bg-[#222222]">Cancel</button>
            <button type="submit" disabled={!selectedTemplateId || !nextVmid || submitting} className="rounded-md bg-[#00ff88] px-4 py-2 text-sm font-medium text-[#0a0a0a] transition-colors hover:bg-[#00cc6a] disabled:opacity-50">
              {submitting ? "Submitting..." : "Create"}
            </button>
          </div>
        </form>
      ) : (
        <div className="flex flex-col gap-5">
          {activeJob?.status === "failed" && activeJob.error && (
            <div className="rounded border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">{activeJob.error}</div>
          )}

          <div className="flex flex-col gap-2">
            <div className="flex items-center justify-between">
              <span className="text-sm text-[#e0e0e0]">{activeJob?.status === "failed" ? "Failed" : (STEP_LABELS[currentStep] || "Queued...")}</span>
              <span className="text-xs text-[#888888]">{currentProgress}%</span>
            </div>
            <div className="h-2 w-full overflow-hidden rounded-full bg-[#222222]">
              <div className={`h-full rounded-full transition-all duration-500 ease-out ${activeJob?.status === "failed" ? "bg-red-500" : "bg-[#00ff88]"}`} style={{ width: `${currentProgress}%` }} />
            </div>
          </div>

          <div className="flex flex-col gap-1.5">
            {STEP_LIST.map((s, stepIndex) => {
              const currentIndex = STEP_LIST.findIndex((x) => x.key === currentStep);
              const isDone = currentStep === "ready" || (currentStep === "waiting_for_running" && s.key !== "ready") || currentIndex > stepIndex;
              const isCurrent = currentStep === s.key || (currentStep === "waiting_for_running" && s.key === "ready");
              return (
                <div key={s.key} className="flex items-center gap-2">
                  <div className={`h-2 w-2 rounded-full ${isDone ? "bg-[#00ff88]" : isCurrent ? "bg-[#00ff88] animate-pulse" : "bg-[#333333]"}`} />
                  <span className={`text-xs ${isDone ? "text-[#00ff88]" : isCurrent ? "text-[#e0e0e0]" : "text-[#555555]"}`}>{s.label}</span>
                </div>
              );
            })}
          </div>

          {activeJob?.status === "completed" && (
            <div className="flex flex-col gap-3 rounded-lg border border-[#222222] bg-[#111111] p-4">
              {activeJob.ip_address ? (
                <>
                  <div>
                    <div className="mb-1.5 flex items-center gap-1.5 text-xs text-[#888888]"><Terminal size={12} /> SSH Connection</div>
                    <div className="flex items-center gap-2 rounded-md bg-[#0a0a0a] px-3 py-2">
                      <code className="flex-1 font-mono text-sm text-[#00ff88]">ssh {sshUser}@{activeJob.ip_address}</code>
                      <CopyBtn text={`ssh ${sshUser}@${activeJob.ip_address}`} field="ssh" />
                    </div>
                  </div>
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-[#888888]">IP Address</span>
                    <div className="flex items-center gap-2">
                      <code className="font-mono text-[#e0e0e0]">{activeJob.ip_address}</code>
                      <CopyBtn text={activeJob.ip_address} field="ip" small />
                    </div>
                  </div>
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-[#888888]">User</span>
                    <code className="font-mono text-[#e0e0e0]">{sshUser}</code>
                  </div>
                </>
              ) : (
                <div className="text-xs text-[#888888]">VM is running. IP address not available (qemu-guest-agent may not be installed).</div>
              )}
              {effectivePassword && (
                <div className="flex items-center justify-between text-xs">
                  <span className="text-[#888888]">Password</span>
                  <div className="flex items-center gap-2">
                    <code className="font-mono text-[#e0e0e0]">{effectivePassword}</code>
                    <CopyBtn text={effectivePassword} field="final-password" small />
                  </div>
                </div>
              )}
            </div>
          )}

          <div className="mt-2 flex justify-end gap-3">
            {!isTerminal && <button type="button" onClick={handleClose} className="rounded-md border border-[#222222] px-4 py-2 text-sm text-[#e0e0e0] transition-colors hover:bg-[#222222]">Minimize</button>}
            {isTerminal && <button type="button" onClick={handleDone} className="rounded-md bg-[#00ff88] px-4 py-2 text-sm font-medium text-[#0a0a0a] transition-colors hover:bg-[#00cc6a]">Done</button>}
          </div>
        </div>
      )}
    </Modal>
  );
}
