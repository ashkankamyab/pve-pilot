"use client";

export type Distro = "ubuntu" | "debian" | "rocky" | "alma" | "centos" | "fedora" | "linux";

const DISTRO_COLORS: Record<Distro, string> = {
  ubuntu: "#E95420",
  debian: "#A80030",
  rocky: "#10B981",
  alma: "#0F4266",
  centos: "#932279",
  fedora: "#51A2DA",
  linux: "#FCC624",
};

const DISTRO_LABELS: Record<Distro, string> = {
  ubuntu: "Ubuntu",
  debian: "Debian",
  rocky: "Rocky",
  alma: "Alma",
  centos: "CentOS",
  fedora: "Fedora",
  linux: "Linux",
};

// Default cloud-init user for each distro
export const DISTRO_USERS: Record<Distro, string> = {
  ubuntu: "ubuntu",
  debian: "debian",
  rocky: "rocky",
  alma: "alma",
  centos: "centos",
  fedora: "fedora",
  linux: "root",
};

export function detectDistro(name: string): Distro {
  const n = name.toLowerCase();
  if (n.includes("ubuntu")) return "ubuntu";
  if (n.includes("debian")) return "debian";
  if (n.includes("rocky")) return "rocky";
  if (n.includes("alma")) return "alma";
  if (n.includes("centos")) return "centos";
  if (n.includes("fedora")) return "fedora";
  return "linux";
}

// Ubuntu circle of friends (simplified)
function UbuntuIcon({ size }: { size: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <circle cx="12" cy="12" r="11" fill="#E95420" />
      <circle cx="12" cy="5" r="2" fill="white" />
      <circle cx="5.5" cy="15.5" r="2" fill="white" />
      <circle cx="18.5" cy="15.5" r="2" fill="white" />
      <circle cx="12" cy="12" r="4" stroke="white" strokeWidth="1.5" fill="none" />
    </svg>
  );
}

function DebianIcon({ size }: { size: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <circle cx="12" cy="12" r="11" fill="#A80030" />
      <text x="12" y="16.5" textAnchor="middle" fill="white" fontSize="14" fontWeight="bold" fontFamily="serif">
        D
      </text>
    </svg>
  );
}

function RockyIcon({ size }: { size: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <circle cx="12" cy="12" r="11" fill="#10B981" />
      <path d="M7 16 L12 6 L17 16 Z" fill="white" opacity="0.9" />
    </svg>
  );
}

function AlmaIcon({ size }: { size: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <circle cx="12" cy="12" r="11" fill="#0F4266" />
      <text x="12" y="16.5" textAnchor="middle" fill="white" fontSize="14" fontWeight="bold" fontFamily="sans-serif">
        A
      </text>
    </svg>
  );
}

function LinuxIcon({ size }: { size: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <circle cx="12" cy="12" r="11" fill="#333" />
      <text x="12" y="16.5" textAnchor="middle" fill="#FCC624" fontSize="14" fontWeight="bold" fontFamily="sans-serif">
        🐧
      </text>
    </svg>
  );
}

const ICONS: Record<Distro, React.FC<{ size: number }>> = {
  ubuntu: UbuntuIcon,
  debian: DebianIcon,
  rocky: RockyIcon,
  alma: AlmaIcon,
  centos: ({ size }) => (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <circle cx="12" cy="12" r="11" fill="#932279" />
      <text x="12" y="16.5" textAnchor="middle" fill="white" fontSize="14" fontWeight="bold" fontFamily="sans-serif">C</text>
    </svg>
  ),
  fedora: ({ size }) => (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <circle cx="12" cy="12" r="11" fill="#51A2DA" />
      <text x="12" y="16.5" textAnchor="middle" fill="white" fontSize="14" fontWeight="bold" fontFamily="sans-serif">F</text>
    </svg>
  ),
  linux: LinuxIcon,
};

interface DistroIconProps {
  name: string;
  size?: number;
}

export default function DistroIcon({ name, size = 24 }: DistroIconProps) {
  const distro = detectDistro(name);
  const Icon = ICONS[distro];
  return <Icon size={size} />;
}

export function DistroLabel({ name }: { name: string }) {
  const distro = detectDistro(name);
  return <span>{DISTRO_LABELS[distro]}</span>;
}

export function getDistroColor(name: string): string {
  return DISTRO_COLORS[detectDistro(name)];
}
