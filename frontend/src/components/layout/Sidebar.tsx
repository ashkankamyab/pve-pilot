"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Server,
  Monitor,
  Box,
  Copy,
  HardDrive,
} from "lucide-react";

const navItems = [
  { href: "/", label: "Dashboard", icon: LayoutDashboard },
  { href: "/nodes", label: "Nodes", icon: Server },
  { href: "/vms", label: "VMs", icon: Monitor },
  { href: "/containers", label: "Containers", icon: Box },
  { href: "/templates", label: "Templates", icon: Copy },
  { href: "/storage", label: "Storage", icon: HardDrive },
];

export default function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="fixed left-0 top-0 z-40 flex h-full w-56 flex-col border-r border-[#222222] bg-[#111111]">
      <div className="flex items-center gap-2 px-5 py-5">
        <span className="font-mono text-lg font-bold text-[#00ff88]">&gt;_</span>
        <span className="text-lg font-semibold text-[#e0e0e0]">PVE Pilot</span>
      </div>

      <nav className="mt-2 flex flex-1 flex-col gap-1 px-3">
        {navItems.map(({ href, label, icon: Icon }) => {
          const isActive =
            href === "/" ? pathname === "/" : pathname.startsWith(href);
          return (
            <Link
              key={href}
              href={href}
              className={`flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
                isActive
                  ? "bg-[#00ff88]/15 text-[#00ff88]"
                  : "text-[#888888] hover:bg-[#1a1a1a] hover:text-[#e0e0e0]"
              }`}
            >
              <Icon size={18} />
              {label}
            </Link>
          );
        })}
      </nav>
    </aside>
  );
}
