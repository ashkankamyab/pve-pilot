"use client";

import { useState } from "react";
import Modal from "@/components/shared/Modal";
import { apiPost } from "@/lib/api";
import { TemplateInfo } from "@/lib/types";

interface CreateFromTemplateModalProps {
  isOpen: boolean;
  onClose: () => void;
  templates: TemplateInfo[];
  nodes: string[];
  onSuccess: () => void;
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
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const selectedTemplate = templates.find(
    (t) => `${t.node}-${t.vmid}` === selectedTemplateId
  );

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedTemplate || !newId || !name) return;

    setSubmitting(true);
    setError(null);

    const typePrefix = selectedTemplate.vmtype === "qemu" ? "vms" : "containers";

    try {
      await apiPost(
        `/nodes/${selectedTemplate.node}/${typePrefix}/${selectedTemplate.vmid}/clone`,
        {
          newid: parseInt(newId, 10),
          name,
          full: true,
          target: targetNode || undefined,
        }
      );
      resetForm();
      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create from template");
    } finally {
      setSubmitting(false);
    }
  };

  const resetForm = () => {
    setSelectedTemplateId("");
    setNewId("");
    setName("");
    setTargetNode("");
    setCores("2");
    setMemory("2048");
    setError(null);
  };

  const handleClose = () => {
    resetForm();
    onClose();
  };

  const inputClass =
    "w-full rounded-md border border-[#222222] bg-[#0a0a0a] px-3 py-2 text-sm text-[#e0e0e0] outline-none focus:border-[#00ff88]";

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title="Create from Template">
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

        <p className="text-xs text-[#888888]">
          The VM will be created as a full clone. You can adjust cores and memory after creation via the Proxmox UI.
        </p>

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
            disabled={submitting || !selectedTemplateId}
            className="rounded-md bg-[#00ff88] px-4 py-2 text-sm font-medium text-[#0a0a0a] transition-colors hover:bg-[#00cc6a] disabled:opacity-50"
          >
            {submitting ? "Creating..." : "Create"}
          </button>
        </div>
      </form>
    </Modal>
  );
}
