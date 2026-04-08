"use client";

import { useState } from "react";
import Modal from "@/components/shared/Modal";
import { apiPost } from "@/lib/api";
import { TemplateInfo, CloneRequest } from "@/lib/types";

interface CloneModalProps {
  isOpen: boolean;
  onClose: () => void;
  template: TemplateInfo | null;
  nodes: string[];
}

export default function CloneModal({
  isOpen,
  onClose,
  template,
  nodes,
}: CloneModalProps) {
  const [newId, setNewId] = useState("");
  const [name, setName] = useState("");
  const [fullClone, setFullClone] = useState(true);
  const [targetNode, setTargetNode] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!template) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newId || !name) return;

    setSubmitting(true);
    setError(null);

    const body: CloneRequest = {
      newid: parseInt(newId, 10),
      name,
      full: fullClone,
      target: targetNode || undefined,
    };

    const typePrefix = template.vmtype === "qemu" ? "vms" : "containers";

    try {
      await apiPost(
        `/nodes/${template.node}/${typePrefix}/${template.vmid}/clone`,
        body
      );
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Clone failed");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={`Clone ${template.name || template.vmid}`}>
      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        {error && (
          <div className="rounded border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">
            {error}
          </div>
        )}

        <label className="flex flex-col gap-1">
          <span className="text-xs text-[#888888]">New VMID</span>
          <input
            type="number"
            required
            value={newId}
            onChange={(e) => setNewId(e.target.value)}
            className="rounded-md border border-[#222222] bg-[#0a0a0a] px-3 py-2 text-sm text-[#e0e0e0] outline-none focus:border-[#00ff88]"
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
            className="rounded-md border border-[#222222] bg-[#0a0a0a] px-3 py-2 text-sm text-[#e0e0e0] outline-none focus:border-[#00ff88]"
            placeholder="my-new-vm"
          />
        </label>

        {nodes.length > 1 && (
          <label className="flex flex-col gap-1">
            <span className="text-xs text-[#888888]">Target Node</span>
            <select
              value={targetNode}
              onChange={(e) => setTargetNode(e.target.value)}
              className="rounded-md border border-[#222222] bg-[#0a0a0a] px-3 py-2 text-sm text-[#e0e0e0] outline-none focus:border-[#00ff88]"
            >
              <option value="">Same node</option>
              {nodes.map((n) => (
                <option key={n} value={n}>
                  {n}
                </option>
              ))}
            </select>
          </label>
        )}

        <label className="flex items-center gap-3">
          <button
            type="button"
            role="switch"
            aria-checked={fullClone}
            onClick={() => setFullClone(!fullClone)}
            className={`relative h-5 w-9 rounded-full transition-colors ${
              fullClone ? "bg-[#00ff88]" : "bg-[#222222]"
            }`}
          >
            <span
              className={`absolute top-0.5 left-0.5 h-4 w-4 rounded-full bg-white transition-transform ${
                fullClone ? "translate-x-4" : ""
              }`}
            />
          </button>
          <span className="text-sm text-[#e0e0e0]">Full Clone</span>
        </label>

        <div className="mt-2 flex justify-end gap-3">
          <button
            type="button"
            onClick={onClose}
            className="rounded-md border border-[#222222] px-4 py-2 text-sm text-[#e0e0e0] transition-colors hover:bg-[#222222]"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={submitting}
            className="rounded-md bg-[#00ff88] px-4 py-2 text-sm font-medium text-[#0a0a0a] transition-colors hover:bg-[#00cc6a] disabled:opacity-50"
          >
            {submitting ? "Cloning..." : "Clone"}
          </button>
        </div>
      </form>
    </Modal>
  );
}
