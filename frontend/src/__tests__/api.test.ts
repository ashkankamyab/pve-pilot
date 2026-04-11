import { describe, it, expect, vi, beforeEach } from "vitest";
import { formatBytes, formatUptime, cpuPercent, memPercent } from "@/lib/api";

describe("formatBytes", () => {
  it("returns '0 B' for zero", () => {
    expect(formatBytes(0)).toBe("0 B");
  });

  it("formats bytes", () => {
    expect(formatBytes(512)).toBe("512.0 B");
  });

  it("formats kilobytes", () => {
    expect(formatBytes(1024)).toBe("1.0 KB");
    expect(formatBytes(1536)).toBe("1.5 KB");
  });

  it("formats megabytes", () => {
    expect(formatBytes(1048576)).toBe("1.0 MB");
    expect(formatBytes(500 * 1024 * 1024)).toBe("500.0 MB");
  });

  it("formats gigabytes", () => {
    expect(formatBytes(1073741824)).toBe("1.0 GB");
    expect(formatBytes(8 * 1024 * 1024 * 1024)).toBe("8.0 GB");
  });

  it("formats terabytes", () => {
    expect(formatBytes(1099511627776)).toBe("1.0 TB");
  });
});

describe("formatUptime", () => {
  it("formats minutes only", () => {
    expect(formatUptime(300)).toBe("5m");
    expect(formatUptime(0)).toBe("0m");
  });

  it("formats hours and minutes", () => {
    expect(formatUptime(3660)).toBe("1h 1m");
    expect(formatUptime(7200)).toBe("2h 0m");
  });

  it("formats days and hours", () => {
    expect(formatUptime(86400)).toBe("1d 0h");
    expect(formatUptime(90000)).toBe("1d 1h");
    expect(formatUptime(172800 + 7200)).toBe("2d 2h");
  });
});

describe("cpuPercent", () => {
  it("converts decimal to percentage", () => {
    expect(cpuPercent(0)).toBe(0);
    expect(cpuPercent(0.5)).toBe(50);
    expect(cpuPercent(1.0)).toBe(100);
    expect(cpuPercent(0.123)).toBe(12);
  });
});

describe("memPercent", () => {
  it("returns 0 when total is 0", () => {
    expect(memPercent(100, 0)).toBe(0);
  });

  it("calculates percentage correctly", () => {
    expect(memPercent(512, 1024)).toBe(50);
    expect(memPercent(768, 1024)).toBe(75);
    expect(memPercent(1024, 1024)).toBe(100);
  });
});

describe("apiFetch", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("fetches data successfully", async () => {
    const mockData = { nodes: 2, vms_running: 5 };
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockData),
    });

    const { apiFetch } = await import("@/lib/api");
    const result = await apiFetch("/cluster/summary");

    expect(result).toEqual(mockData);
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining("/api/cluster/summary"),
      expect.objectContaining({ headers: { "Content-Type": "application/json" } })
    );
  });

  it("throws on API error", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 502,
      json: () => Promise.resolve({ error: "proxmox unreachable" }),
    });

    const { apiFetch } = await import("@/lib/api");
    await expect(apiFetch("/cluster/summary")).rejects.toThrow("proxmox unreachable");
  });

  it("throws with status text when error JSON fails", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
      statusText: "Internal Server Error",
      json: () => Promise.reject(new Error("parse error")),
    });

    const { apiFetch } = await import("@/lib/api");
    await expect(apiFetch("/health")).rejects.toThrow("Internal Server Error");
  });
});

describe("apiPost", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("sends POST with JSON body", async () => {
    const responseData = { upid: "UPID:pve:123" };
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(responseData),
    });

    const { apiPost } = await import("@/lib/api");
    const result = await apiPost("/nodes/pve/vms/100/start", { timeout: 30 });

    expect(result).toEqual(responseData);
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining("/api/nodes/pve/vms/100/start"),
      expect.objectContaining({
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ timeout: 30 }),
      })
    );
  });

  it("sends POST without body", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ upid: "UPID:pve:456" }),
    });

    const { apiPost } = await import("@/lib/api");
    await apiPost("/nodes/pve/vms/100/start");

    expect(fetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({ body: undefined })
    );
  });
});

describe("apiDelete", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("sends DELETE request", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ status: "deleted" }),
    });

    const { apiDelete } = await import("@/lib/api");
    const result = await apiDelete("/backups?node=pve&volid=local:backup/123");

    expect(result).toEqual({ status: "deleted" });
    expect(fetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({ method: "DELETE" })
    );
  });
});
