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
      <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke="#222222"
          strokeWidth={strokeWidth}
          transform={`rotate(-90 ${size / 2} ${size / 2})`}
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
          transform={`rotate(-90 ${size / 2} ${size / 2})`}
          className="transition-all duration-500"
        />
        <text
          x="50%"
          y="50%"
          dominantBaseline="central"
          textAnchor="middle"
          fill="#e0e0e0"
          fontSize={size * 0.22}
          fontWeight="bold"
          fontFamily="system-ui, sans-serif"
        >
          {value}%
        </text>
      </svg>
      <span className="text-xs text-[#888888]">{label}</span>
    </div>
  );
}
