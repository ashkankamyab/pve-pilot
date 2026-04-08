"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import Modal from "@/components/shared/Modal";
import { apiPost, apiFetch } from "@/lib/api";
import { TemplateInfo, NetworkInterface, VMStatus } from "@/lib/types";

interface CreateFromTemplateModalProps {
  isOpen: boolean;
  onClose: () => void;
  templates: TemplateInfo[];
  nodes: string[];
  onSuccess: () => void;
}

type ProvisionStep = "idle" | "cloning" | "configuring" | "resizing" | "starting" | "polling" | "ready";

const STEP_PROGRESS: Record<ProvisionStep, number> = {
  idle: 0,
  cloning: 30,
  configuring: 50,
  resizing: 70,
  starting: 90,
  polling: 95,
  ready: 100,
};

const STEP_LABELS: Record<ProvisionStep, string> = {
  idle: "",
  cloning: "Cloning template...",
  configuring: "Configuring cloud-init...",
  resizing: "Resizing disk...",
  starting: "Starting VM...",
  polling: "Waiting for VM to come online...",
  ready: "Ready!",
};

const DISK_SIZES = [20, 30, 32, 40, 50, 60, 90, 100];

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
  isOpen,
  onClose,
  templates,
  nodes,
  onSuccess,
}: CreateFromTemplateModalProps) {
  const [selectedTemplateId, setSelectedTemplateId] = useState("");
  const [newId, setNewId] = useState("");
  const [name, setName] = useState("");
  const [targetNode, setTargetNode] = useState("");
  const [cores, setCores] = useState("2");
  const [memory, setMemory] = useState("2048");
  const [passwordMode, setPasswordMode] = useState<"set" | "generate">("set");
  const [password, setPassword] = useState("");
  const [generatedPassword, setGeneratedPassword] = useState("");
  const [sshKey, setSshKey] = useState("");
  const [diskSize, setDiskSize] = useState("32");
  const [step, setStep] = useState<ProvisionStep>("idle");
  const [error, setError] = useState<string | null>(null);
  const [ipAddress, setIpAddress] = useState<string | null>(null);
  const [copiedField, setCopiedField] = useState<string | null>(null);
  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const selectedTemplate = templates.find(
    (t) => `${t.node}-${t.vmid}` === selectedTemplateId
  );

  const isProvisioning = step !== "idle" && step !== "ready";

  const stopPolling = useCallback(() => {
    if (pollingRef.current) {
      clearInterval(pollingRef.current);
      pollingRef.current = null;
    }
  }, []);

  useEffect(() => {
    return () => stopPolling();
  }, [stopPolling]);

  const pollForStatus = useCallback((node: string, vmid: number, vmtype: string) => {
    stopPolling();
    setStep("polling");

    pollingRef.current = setInterval(async () => {
      try {
        const typePrefix = vmtype === "qemu" ? "vms" : "containers";
        const status = await apiFetch<VMStatus>(`/nodes/${node}/${typePrefix}/${vmid}/status`);

        if (status.status === "running") {
          // VM is running, now try to get IP
          if (vmtype === "qemu") {
            try {
              const interfaces = await apiFetch<NetworkInterface[]>(
                `/nodes/${node}/vms/${vmid}/interfaces`
              );
              const ip = findIPAddress(interfaces);
              if (ip) {
                setIpAddress(ip);
                setStep("ready");
                stopPolling();
                return;
              }
            } catch {
              // Guest agent may not be ready yet, keep polling
            }
          }

          // For LXC or if no IP found yet, check if we've been running for a bit
          // Set ready after detecting running status
          setStep("ready");
          stopPolling();
        }
      } catch {
        // VM may still be starting, keep polling
      }
    }, 2000);
  }, [stopPolling]);

  const findIPAddress = (interfaces: NetworkInterface[]): string | null => {
    for (const iface of interfaces) {
      if (iface.name === "lo") continue;
      for (const addr of iface["ip-addresses"] || []) {
        if (addr["ip-address-type"] === "ipv4" && addr["ip-address"] !== "127.0.0.1") {
          return addr["ip-address"];
        }
      }
    }
    return null;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedTemplate || !newId || !name) return;

    setError(null);
    setIpAddress(null);
    setStep("cloning");

    const typePrefix = selectedTemplate.vmtype === "qemu" ? "vms" : "containers";
    const effectivePassword = passwordMode === "generate" ? generatedPassword : password;

    try {
      setStep("cloning");

      const response = await apiPost<{ vmid: number; node: string }>(
        `/nodes/${selectedTemplate.node}/${typePrefix}/${selectedTemplate.vmid}/provision`,
        {
          newid: parseInt(newId, 10),
          name,
          full: true,
          target: targetNode || undefined,
          password: effectivePassword || undefined,
          sshkeys: sshKey || undefined,
          disk_size: parseInt(diskSize, 10),
        }
      );

      // Provision endpoint handles clone + configure + resize + start
      // Now poll for running status and IP
      const effectiveNode = response.node || targetNode || selectedTemplate.node;
      pollForStatus(effectiveNode, parseInt(newId, 10), selectedTemplate.vmtype);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create from template");
      setStep("idle");
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
    } catch {
      // Fallback
    }
  };

  const resetForm = () => {
    setSelectedTemplateId("");
    setNewId("");
    setName("");
    setTargetNode("");
    setCores("2");
    setMemory("2048");
    setPasswordMode("set");
    setPassword("");
    setGeneratedPassword("");
    setSshKey("");
    setDiskSize("32");
    setStep("idle");
    setError(null);
    setIpAddress(null);
    stopPolling();
  };

  const handleClose = () => {
    if (isProvisioning) return; // Don't allow closing during provisioning
    resetForm();
    if (step === "ready") {
      onSuccess();
    }
    onClose();
  };

  const handleDone = () => {
    resetForm();
    onSuccess();
    onClose();
  };

  const inputClass =
    "w-full rounded-md border border-[#222222] bg-[#0a0a0a] px-3 py-2 text-sm text-[#e0e0e0] outline-none focus:border-[#00ff88]";

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title="Create from Template">
      {step === "idle" ? (
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          {error && (
            <div className="rounded border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">
              {error}
            </div>
          )}

          <label className="flex flex-col gap-1">
            <span className="text-xs text-[#888888]">Template</span>
            <select
              required
              value={selectedTemplateId}
              onChange={(e) => setSelectedTemplateId(e.target.value)}
              className={inputClass}
            >
              <option value="">Select a template...</option>
              {templates.map((t) => (
                <option key={`${t.node}-${t.vmid}`} value={`${t.node}-${t.vmid}`}>
                  {t.vmid} - {t.name || "unnamed"} ({t.node})
                </option>
              ))}
            </select>
          </label>

          <label className="flex flex-col gap-1">
            <span className="text-xs text-[#888888]">New VMID</span>
            <input
              type="number"
              required
              value={newId}
              onChange={(e) => setNewId(e.target.value)}
              className={inputClass}
              placeholder="100"
            />
          </label>

          <label className="flex flex-col gap-1">
            <span className="text-xs text-[#888888]">Name</span>
            <input
              type="text"
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              className={inputClass}
              placeholder="my-new-vm"
            />
          </label>

          {nodes.length > 1 && (
            <label className="flex flex-col gap-1">
              <span className="text-xs text-[#888888]">Target Node</span>
              <select
                value={targetNode}
                onChange={(e) => setTargetNode(e.target.value)}
                className={inputClass}
              >
                <option value="">Same as template</option>
                {nodes.map((n) => (
                  <option key={n} value={n}>
                    {n}
                  </option>
                ))}
              </select>
            </label>
          )}

          <div className="grid grid-cols-2 gap-3">
            <label className="flex flex-col gap-1">
              <span className="text-xs text-[#888888]">Cores</span>
              <input
                type="number"
                min="1"
                value={cores}
                onChange={(e) => setCores(e.target.value)}
                className={inputClass}
              />
            </label>

            <label className="flex flex-col gap-1">
              <span className="text-xs text-[#888888]">Memory (MB)</span>
              <input
                type="number"
                min="128"
                step="128"
                value={memory}
                onChange={(e) => setMemory(e.target.value)}
                className={inputClass}
              />
            </label>
          </div>

          {/* Disk Size */}
          <label className="flex flex-col gap-1">
            <span className="text-xs text-[#888888]">Disk Size (GB)</span>
            <select
              value={diskSize}
              onChange={(e) => setDiskSize(e.target.value)}
              className={inputClass}
            >
              {DISK_SIZES.map((size) => (
                <option key={size} value={size}>
                  {size} GB
                </option>
              ))}
            </select>
          </label>

          {/* Password Section */}
          <div className="flex flex-col gap-2">
            <span className="text-xs text-[#888888]">Password</span>
            <div className="flex gap-3">
              <label className="flex items-center gap-1.5">
                <input
                  type="radio"
                  name="passwordMode"
                  checked={passwordMode === "set"}
                  onChange={() => setPasswordMode("set")}
                  className="accent-[#00ff88]"
                />
                <span className="text-sm text-[#e0e0e0]">Set password</span>
              </label>
              <label className="flex items-center gap-1.5">
                <input
                  type="radio"
                  name="passwordMode"
                  checked={passwordMode === "generate"}
                  onChange={() => setPasswordMode("generate")}
                  className="accent-[#00ff88]"
                />
                <span className="text-sm text-[#e0e0e0]">Generate random</span>
              </label>
            </div>

            {passwordMode === "set" && (
              <input
                type="text"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className={inputClass}
                placeholder="Enter password"
              />
            )}

            {passwordMode === "generate" && (
              <div className="flex flex-col gap-2">
                <button
                  type="button"
                  onClick={handleGeneratePassword}
                  className="w-fit rounded-md border border-[#222222] px-3 py-1.5 text-xs text-[#e0e0e0] transition-colors hover:border-[#00ff88] hover:text-[#00ff88]"
                >
                  Generate Password
                </button>
                {generatedPassword && (
                  <div className="flex items-center gap-2">
                    <code className="flex-1 rounded border border-[#222222] bg-[#0a0a0a] px-3 py-2 font-mono text-sm text-[#00ff88]">
                      {generatedPassword}
                    </code>
                    <button
                      type="button"
                      onClick={() => copyToClipboard(generatedPassword, "gen-password")}
                      className="rounded border border-[#222222] px-2 py-2 text-xs text-[#888888] transition-colors hover:border-[#00ff88] hover:text-[#00ff88]"
                    >
                      {copiedField === "gen-password" ? "Copied!" : "Copy"}
                    </button>
                  </div>
                )}
              </div>
            )}
          </div>

          {/* SSH Public Key */}
          <label className="flex flex-col gap-1">
            <span className="text-xs text-[#888888]">SSH Public Key (optional)</span>
            <textarea
              value={sshKey}
              onChange={(e) => setSshKey(e.target.value)}
              className={`${inputClass} min-h-[60px] resize-y font-mono text-xs`}
              placeholder="ssh-rsa AAAA... user@host"
              rows={3}
            />
          </label>

          <div className="mt-2 flex justify-end gap-3">
            <button
              type="button"
              onClick={handleClose}
              className="rounded-md border border-[#222222] px-4 py-2 text-sm text-[#e0e0e0] transition-colors hover:bg-[#222222]"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!selectedTemplateId}
              className="rounded-md bg-[#00ff88] px-4 py-2 text-sm font-medium text-[#0a0a0a] transition-colors hover:bg-[#00cc6a] disabled:opacity-50"
            >
              Create
            </button>
          </div>
        </form>
      ) : (
        /* Provisioning Progress View */
        <div className="flex flex-col gap-5">
          {error && (
            <div className="rounded border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">
              {error}
            </div>
          )}

          {/* Progress Bar */}
          <div className="flex flex-col gap-2">
            <div className="flex items-center justify-between">
              <span className="text-sm text-[#e0e0e0]">{STEP_LABELS[step]}</span>
              <span className="text-xs text-[#888888]">{STEP_PROGRESS[step]}%</span>
            </div>
            <div className="h-2 w-full overflow-hidden rounded-full bg-[#222222]">
              <div
                className="h-full rounded-full bg-[#00ff88] transition-all duration-500 ease-out"
                style={{ width: `${STEP_PROGRESS[step]}%` }}
              />
            </div>
          </div>

          {/* Step indicators */}
          <div className="flex flex-col gap-1.5">
            {(["cloning", "configuring", "resizing", "starting", "ready"] as ProvisionStep[]).map((s) => {
              const stepIndex = ["cloning", "configuring", "resizing", "starting", "ready"].indexOf(s);
              const currentIndex = ["cloning", "configuring", "resizing", "starting", "ready"].indexOf(step);
              const isDone = step === "ready" || (step === "polling" && s !== "ready") || currentIndex > stepIndex;
              const isCurrent = step === s || (step === "polling" && s === "ready");

              return (
                <div key={s} className="flex items-center gap-2">
                  <div
                    className={`h-2 w-2 rounded-full ${
                      isDone
                        ? "bg-[#00ff88]"
                        : isCurrent
                        ? "bg-[#00ff88] animate-pulse"
                        : "bg-[#333333]"
                    }`}
                  />
                  <span
                    className={`text-xs ${
                      isDone
                        ? "text-[#00ff88]"
                        : isCurrent
                        ? "text-[#e0e0e0]"
                        : "text-[#555555]"
                    }`}
                  >
                    {s === "cloning" && "Clone template"}
                    {s === "configuring" && "Configure cloud-init"}
                    {s === "resizing" && "Resize disk"}
                    {s === "starting" && "Start VM"}
                    {s === "ready" && "Ready"}
                  </span>
                </div>
              );
            })}
          </div>

          {/* IP Address display */}
          {step === "ready" && ipAddress && (
            <div className="rounded border border-[#00ff88]/30 bg-[#00ff88]/5 px-4 py-3">
              <div className="mb-1 text-xs text-[#888888]">IP Address</div>
              <div className="flex items-center gap-2">
                <code className="font-mono text-base text-[#00ff88]">{ipAddress}</code>
                <button
                  type="button"
                  onClick={() => copyToClipboard(ipAddress, "ip")}
                  className="rounded border border-[#222222] px-2 py-1 text-xs text-[#888888] transition-colors hover:border-[#00ff88] hover:text-[#00ff88]"
                >
                  {copiedField === "ip" ? "Copied!" : "Copy"}
                </button>
              </div>
            </div>
          )}

          {step === "ready" && !ipAddress && (
            <div className="rounded border border-[#222222] bg-[#111111] px-4 py-3">
              <div className="text-xs text-[#888888]">
                VM is running. IP address not available (qemu-guest-agent may not be installed).
              </div>
            </div>
          )}

          {/* Show password reminder if one was set */}
          {step === "ready" && (passwordMode === "generate" ? generatedPassword : password) && (
            <div className="rounded border border-[#222222] bg-[#111111] px-4 py-3">
              <div className="mb-1 text-xs text-[#888888]">Password</div>
              <div className="flex items-center gap-2">
                <code className="font-mono text-sm text-[#e0e0e0]">
                  {passwordMode === "generate" ? generatedPassword : password}
                </code>
                <button
                  type="button"
                  onClick={() =>
                    copyToClipboard(
                      passwordMode === "generate" ? generatedPassword : password,
                      "final-password"
                    )
                  }
                  className="rounded border border-[#222222] px-2 py-1 text-xs text-[#888888] transition-colors hover:border-[#00ff88] hover:text-[#00ff88]"
                >
                  {copiedField === "final-password" ? "Copied!" : "Copy"}
                </button>
              </div>
            </div>
          )}

          {/* Close button only when ready */}
          {step === "ready" && (
            <div className="mt-2 flex justify-end">
              <button
                type="button"
                onClick={handleDone}
                className="rounded-md bg-[#00ff88] px-4 py-2 text-sm font-medium text-[#0a0a0a] transition-colors hover:bg-[#00cc6a]"
              >
                Done
              </button>
            </div>
          )}
        </div>
      )}
    </Modal>
  );
}
