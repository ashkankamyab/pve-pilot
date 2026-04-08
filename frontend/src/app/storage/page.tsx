"use client";

import { useCallback } from "react";
import { usePolling } from "@/hooks/usePolling";
import { apiFetch, formatBytes } from "@/lib/api";
import { ClusterResource, StorageInfo } from "@/lib/types";
import MetricBar from "@/components/shared/MetricBar";

interface StorageWithNode extends StorageInfo {
  node: string;
}

export default function StoragePage() {
  const fetchStorage = useCallback(async () => {
    const resources = await apiFetch<ClusterResource[]>("/cluster/resources");
    const nodeNames = [
      ...new Set(resources.filter((r) => r.type === "node").map((r) => r.node)),
    ];

    const results = await Promise.all(
      nodeNames.map(async (node) => {
        try {
          const storages = await apiFetch<StorageInfo[]>(
            `/nodes/${node}/storage`
          );
          return storages.map((s) => ({ ...s, node }));
        } catch {
          return [];
        }
      })
    );

    return results.flat();
  }, []);

  const { data: storages, isLoading } = usePolling(fetchStorage, 10000);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <span className="text-[#888888]">Loading...</span>
      </div>
    );
  }

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {(storages ?? []).map((s) => {
        const usedGB = s.used / 1073741824;
        const totalGB = s.total / 1073741824;

        return (
          <div
            key={`${s.node}-${s.storage}`}
            className="rounded-lg border border-[#222222] bg-[#161616] p-5"
          >
            <div className="mb-1 flex items-center justify-between">
              <h3 className="font-mono text-sm font-semibold text-[#e0e0e0]">
                {s.storage}
              </h3>
              <span
                className={`inline-block h-2 w-2 rounded-full ${
                  s.active ? "bg-[#00ff88]" : "bg-red-500"
                }`}
              />
            </div>
            <p className="mb-4 text-xs text-[#888888]">
              {s.node} &middot; {s.type} &middot; {s.content}
            </p>
            <MetricBar
              used={usedGB}
              total={totalGB}
              label="Usage"
              unit="GB"
            />
            <div className="mt-3 flex items-center justify-between text-xs text-[#888888]">
              <span>Available: {formatBytes(s.avail)}</span>
              <span>{s.shared ? "Shared" : "Local"}</span>
            </div>
          </div>
        );
      })}
      {(storages ?? []).length === 0 && (
        <div className="col-span-full py-8 text-center text-[#888888]">
          No storage pools found.
        </div>
      )}
    </div>
  );
}
