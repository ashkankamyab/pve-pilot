interface MetricBarProps {
  used: number;
  total: number;
  label: string;
  unit?: string;
}

function formatValue(value: number, unit?: string): string {
  if (!unit) return value.toFixed(1);
  return `${value.toFixed(1)} ${unit}`;
}

export default function MetricBar({ used, total, label, unit }: MetricBarProps) {
  const pct = total > 0 ? (used / total) * 100 : 0;

  let barColor = "bg-[#00ff88]";
  if (pct > 80) barColor = "bg-red-500";
  else if (pct > 60) barColor = "bg-yellow-500";

  return (
    <div className="w-full">
      <div className="mb-1 flex items-center justify-between text-xs">
        <span className="text-[#888888]">{label}</span>
        <span className="text-[#e0e0e0]">
          {formatValue(used, unit)} / {formatValue(total, unit)}
        </span>
      </div>
      <div className="h-2 w-full overflow-hidden rounded-full bg-[#222222]">
        <div
          className={`h-full rounded-full transition-all ${barColor}`}
          style={{ width: `${Math.min(pct, 100)}%` }}
        />
      </div>
    </div>
  );
}
