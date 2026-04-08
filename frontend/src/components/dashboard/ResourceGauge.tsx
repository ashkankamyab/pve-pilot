interface ResourceGaugeProps {
  value: number;
  label: string;
  size?: number;
}

export default function ResourceGauge({
  value,
  label,
  size = 100,
}: ResourceGaugeProps) {
  const strokeWidth = 8;
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  const offset = circumference - (Math.min(value, 100) / 100) * circumference;

  let strokeColor = "#00ff88";
  if (value > 80) strokeColor = "#ef4444";
  else if (value > 60) strokeColor = "#eab308";

  return (
    <div className="flex flex-col items-center gap-1">
      <svg width={size} height={size} className="-rotate-90">
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke="#222222"
          strokeWidth={strokeWidth}
        />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke={strokeColor}
          strokeWidth={strokeWidth}
          strokeDasharray={circumference}
          strokeDashoffset={offset}
          strokeLinecap="round"
          className="transition-all duration-500"
        />
      </svg>
      <div className="-mt-[calc(50%+10px)] flex flex-col items-center justify-center" style={{ marginTop: `-${size / 2 + 10}px`, height: `${size}px` }}>
        <span className="text-lg font-bold text-[#e0e0e0]">{value}%</span>
      </div>
      <span className="text-xs text-[#888888]">{label}</span>
    </div>
  );
}
