"use client";

import { createContext, useContext, useState, useCallback, useRef, useEffect, ReactNode } from "react";
import { apiSSE } from "@/lib/api";
import { Job, JobEvent } from "@/lib/types";

interface JobWithCredentials extends Job {
  password?: string;
  sshkey?: string;
}

interface JobsContextValue {
  jobs: JobWithCredentials[];
  activeCount: number;
  addJob: (job: JobWithCredentials) => void;
  getJob: (id: string) => JobWithCredentials | undefined;
  getJobByVmid: (vmid: number) => JobWithCredentials | undefined;
  clearCompleted: () => void;
}

const STORAGE_KEY = "pve-pilot-build-info";

const JobsContext = createContext<JobsContextValue | null>(null);

export function useJobs() {
  const ctx = useContext(JobsContext);
  if (!ctx) throw new Error("useJobs must be used within JobsProvider");
  return ctx;
}

// Persist build info to localStorage (keyed by VMID)
function saveBuildInfo(job: JobWithCredentials) {
  try {
    const stored = JSON.parse(localStorage.getItem(STORAGE_KEY) || "{}");
    stored[job.new_vmid] = {
      id: job.id,
      new_vmid: job.new_vmid,
      name: job.name,
      ciuser: job.ciuser,
      password: job.password,
      sshkey: job.sshkey,
      source_vmid: job.source_vmid,
      source_node: job.source_node,
      target_node: job.target_node,
      ip_address: job.ip_address,
      status: job.status,
      created_at: job.created_at,
    };
    localStorage.setItem(STORAGE_KEY, JSON.stringify(stored));
  } catch {}
}

function loadBuildInfo(vmid: number): JobWithCredentials | undefined {
  try {
    const stored = JSON.parse(localStorage.getItem(STORAGE_KEY) || "{}");
    return stored[vmid] || undefined;
  } catch {
    return undefined;
  }
}

export function JobsProvider({ children }: { children: ReactNode }) {
  const [jobs, setJobs] = useState<JobWithCredentials[]>([]);
  const sseCleanups = useRef<Map<string, () => void>>(new Map());

  const updateJob = useCallback((jobId: string, event: JobEvent) => {
    setJobs((prev) => {
      const updated = prev.map((j) =>
        j.id === jobId
          ? {
              ...j,
              status: event.status,
              step: event.step,
              progress: event.progress,
              error: event.error || j.error,
              ip_address: event.ip_address || j.ip_address,
            }
          : j
      );

      // Persist to localStorage when completed
      if (event.status === "completed") {
        const job = updated.find((j) => j.id === jobId);
        if (job) saveBuildInfo(job);
      }

      return updated;
    });

    if (event.status === "completed" || event.status === "failed") {
      const cleanup = sseCleanups.current.get(jobId);
      if (cleanup) {
        cleanup();
        sseCleanups.current.delete(jobId);
      }
    }
  }, []);

  const addJob = useCallback(
    (job: JobWithCredentials) => {
      setJobs((prev) => [job, ...prev]);
      // Also save immediately so build info is available even if SSE fails
      saveBuildInfo(job);

      const cleanup = apiSSE<JobEvent>(`/jobs/${job.id}/events`, (event) => {
        updateJob(job.id, event);
      });
      sseCleanups.current.set(job.id, cleanup);
    },
    [updateJob]
  );

  const getJob = useCallback(
    (id: string) => jobs.find((j) => j.id === id),
    [jobs]
  );

  const getJobByVmid = useCallback(
    (vmid: number) => {
      // Check in-memory jobs first, then localStorage
      const inMemory = jobs.find((j) => j.new_vmid === vmid);
      if (inMemory) return inMemory;
      return loadBuildInfo(vmid);
    },
    [jobs]
  );

  const clearCompleted = useCallback(() => {
    setJobs((prev) => prev.filter((j) => j.status !== "completed" && j.status !== "failed"));
  }, []);

  const activeCount = jobs.filter(
    (j) => j.status === "pending" || j.status === "running"
  ).length;

  return (
    <JobsContext.Provider value={{ jobs, activeCount, addJob, getJob, getJobByVmid, clearCompleted }}>
      {children}
    </JobsContext.Provider>
  );
}
