interface StatusBadgeProps {
  status: string;
}

const statusStyles: Record<string, string> = {
  running: "bg-[#00ff88]/15 text-[#00ff88]",
  stopped: "bg-red-500/15 text-red-400",
  paused: "bg-yellow-500/15 text-yellow-400",
};

export default function StatusBadge({ status }: StatusBadgeProps) {
  const style = statusStyles[status] || "bg-[#222222] text-[#888888]";
  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${style}`}
    >
      {status}
    </span>
  );
}
