"use client";

import { useState, useRef, useEffect } from "react";
import { ChevronDown } from "lucide-react";
import DistroIcon, { detectDistro, DISTRO_USERS } from "./DistroIcon";
import { TemplateInfo } from "@/lib/types";

interface TemplateSelectProps {
  templates: TemplateInfo[];
  value: string; // "node-vmid"
  onChange: (id: string) => void;
}

function templateLabel(t: TemplateInfo): string {
  const distro = detectDistro(t.name || "");
  const user = DISTRO_USERS[distro];
  return t.name || `VM ${t.vmid}`;
}

export default function TemplateSelect({ templates, value, onChange }: TemplateSelectProps) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  const selected = templates.find((t) => `${t.node}-${t.vmid}` === value);

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  // Close on Escape
  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [open]);

  return (
    <div className="relative" ref={ref}>
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex w-full items-center gap-2 rounded-md border border-[#222222] bg-[#0a0a0a] px-3 py-2 text-left text-sm outline-none focus:border-[#00ff88] transition-colors"
      >
        {selected ? (
          <>
            <DistroIcon name={selected.name || ""} size={20} />
            <span className="flex-1 truncate text-[#e0e0e0]">{templateLabel(selected)}</span>
            <span className="text-xs text-[#555555]">{selected.node}</span>
          </>
        ) : (
          <span className="flex-1 text-[#555555]">Select an image...</span>
        )}
        <ChevronDown size={14} className={`text-[#555555] transition-transform ${open ? "rotate-180" : ""}`} />
      </button>

      {open && (
        <div className="absolute left-0 right-0 top-full z-50 mt-1 max-h-64 overflow-y-auto rounded-lg border border-[#222222] bg-[#161616] shadow-2xl">
          {templates.length === 0 ? (
            <div className="px-3 py-4 text-center text-xs text-[#555555]">No images available</div>
          ) : (
            templates.map((t) => {
              const id = `${t.node}-${t.vmid}`;
              const isSelected = id === value;
              const distro = detectDistro(t.name || "");
              const user = DISTRO_USERS[distro];

              return (
                <button
                  key={id}
                  type="button"
                  onClick={() => {
                    onChange(id);
                    setOpen(false);
                  }}
                  className={`flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors hover:bg-[#1e1e1e] ${
                    isSelected ? "bg-[#1a1a1a]" : ""
                  }`}
                >
                  <DistroIcon name={t.name || ""} size={24} />
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm text-[#e0e0e0] truncate">{templateLabel(t)}</span>
                    </div>
                    <div className="flex items-center gap-2 text-[10px] text-[#555555]">
                      <span>VMID {t.vmid}</span>
                      <span>·</span>
                      <span>{t.node}</span>
                      <span>·</span>
                      <span>user: {user}</span>
                    </div>
                  </div>
                  {isSelected && (
                    <div className="h-2 w-2 rounded-full bg-[#00ff88]" />
                  )}
                </button>
              );
            })
          )}
        </div>
      )}
    </div>
  );
}
