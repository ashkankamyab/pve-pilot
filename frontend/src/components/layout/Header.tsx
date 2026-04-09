"use client";

import { usePathname } from "next/navigation";
import { usePolling } from "@/hooks/usePolling";
import { apiFetch } from "@/lib/api";
import JobsPanel from "@/components/jobs/JobsPanel";

const pageTitles: Record<string, string> = {
  "/": "Dashboard",
  "/nodes": "Nodes",
  "/vms": "Virtual Machines",
  "/containers": "Containers",
  "/templates": "Templates",
  "/storage": "Storage",
};

export default function Header() {
  const pathname = usePathname();
  const title = pageTitles[pathname] || "PVE Pilot";

  const { data: health, error } = usePolling(
    () => apiFetch<{ status: string }>("/health"),
    10000
  );

  const isConnected = health && !error;

  return (
    <header className="flex h-14 items-center justify-between border-b border-[#222222] bg-[#0a0a0a] px-6">
      <h1 className="text-lg font-semibold text-[#e0e0e0]">{title}</h1>
      <div className="flex items-center gap-4">
        <JobsPanel />
        <div className="flex items-center gap-2 text-sm text-[#888888]">
          <span
            className={`inline-block h-2.5 w-2.5 rounded-full ${
              isConnected ? "bg-[#00ff88]" : "bg-red-500"
            }`}
          />
          {isConnected ? "Connected" : "Disconnected"}
        </div>
      </div>
    </header>
  );
}
