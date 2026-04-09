"use client";

import { useState, useRef, useEffect } from "react";
import { Loader2, CheckCircle2, XCircle, ChevronDown } from "lucide-react";
import { useJobs } from "@/contexts/JobsContext";
import { Job, JobStep } from "@/lib/types";

const STEP_LABELS: Record<string, string> = {
  "": "Queued",
  cloning: "Cloning",
  configuring: "Configuring",
  resizing: "Resizing",
  starting: "Starting",
  waiting_for_running: "Waiting for running",
  ready: "Ready",
};

function JobRow({ job }: { job: Job }) {
  const statusIcon =
    job.status === "completed" ? (
      <CheckCircle2 size={14} className="text-[#00ff88]" />
    ) : job.status === "failed" ? (
      <XCircle size={14} className="text-red-400" />
    ) : (
      <Loader2 size={14} className="animate-spin text-[#00ff88]" />
    );

  return (
    <div className="flex items-center gap-3 border-b border-[#222222] px-4 py-3 last:border-b-0">
      {statusIcon}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-[#e0e0e0] truncate">
            {job.name}
          </span>
          <span className="text-xs text-[#555555]">#{job.new_vmid}</span>
        </div>
        <div className="text-xs text-[#888888]">
          {job.status === "failed"
            ? job.error || "Failed"
            : STEP_LABELS[job.step] || "Queued"}
        </div>
        {job.status === "completed" && job.ip_address && (
          <div className="text-xs font-mono text-[#00ff88]">{job.ip_address}</div>
        )}
      </div>
      <div className="w-16">
        <div className="h-1.5 w-full overflow-hidden rounded-full bg-[#222222]">
          <div
            className={`h-full rounded-full transition-all duration-500 ${
              job.status === "failed" ? "bg-red-500" : "bg-[#00ff88]"
            }`}
            style={{ width: `${job.progress}%` }}
          />
        </div>
      </div>
    </div>
  );
}

export default function JobsPanel() {
  const { jobs, activeCount, clearCompleted } = useJobs();
  const [isOpen, setIsOpen] = useState(false);
  const panelRef = useRef<HTMLDivElement>(null);

  // Close panel when clicking outside
  useEffect(() => {
    if (!isOpen) return;
    const handleClick = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [isOpen]);

  if (jobs.length === 0) return null;

  const hasCompleted = jobs.some(
    (j) => j.status === "completed" || j.status === "failed"
  );

  return (
    <div className="relative" ref={panelRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-2 rounded-md border border-[#222222] px-3 py-1.5 text-sm text-[#e0e0e0] transition-colors hover:border-[#00ff88] hover:text-[#00ff88]"
      >
        {activeCount > 0 && (
          <Loader2 size={14} className="animate-spin text-[#00ff88]" />
        )}
        <span>
          {activeCount > 0
            ? `${activeCount} job${activeCount > 1 ? "s" : ""} running`
            : "Jobs"}
        </span>
        <ChevronDown size={14} className={`transition-transform ${isOpen ? "rotate-180" : ""}`} />
      </button>

      {isOpen && (
        <div className="absolute right-0 top-full z-50 mt-2 w-80 rounded-lg border border-[#222222] bg-[#161616] shadow-2xl">
          <div className="flex items-center justify-between border-b border-[#222222] px-4 py-2.5">
            <span className="text-xs font-medium text-[#888888] uppercase tracking-wider">
              Provision Jobs
            </span>
            {hasCompleted && (
              <button
                onClick={clearCompleted}
                className="text-xs text-[#555555] hover:text-[#e0e0e0] transition-colors"
              >
                Clear finished
              </button>
            )}
          </div>
          <div className="max-h-64 overflow-y-auto">
            {jobs.map((job) => (
              <JobRow key={job.id} job={job} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
