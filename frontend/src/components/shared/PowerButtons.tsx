"use client";

import { useState } from "react";
import { Play, Square, RotateCcw } from "lucide-react";
import ConfirmDialog from "./ConfirmDialog";

interface PowerButtonsProps {
  status: string;
  onStart: () => void;
  onStop: () => void;
  onReboot: () => void;
  isLoading?: boolean;
}

export default function PowerButtons({
  status,
  onStart,
  onStop,
  onReboot,
  isLoading,
}: PowerButtonsProps) {
  const [confirmAction, setConfirmAction] = useState<"stop" | "reboot" | null>(
    null
  );

  const isRunning = status === "running";

  return (
    <>
      <div className="flex items-center gap-1">
        <button
          disabled={isRunning || isLoading}
          onClick={onStart}
          title="Start"
          className="rounded p-1.5 text-[#888888] transition-colors hover:bg-[#222222] hover:text-[#00ff88] disabled:cursor-not-allowed disabled:opacity-30"
        >
          <Play size={15} />
        </button>
        <button
          disabled={!isRunning || isLoading}
          onClick={() => setConfirmAction("stop")}
          title="Stop"
          className="rounded p-1.5 text-[#888888] transition-colors hover:bg-[#222222] hover:text-red-400 disabled:cursor-not-allowed disabled:opacity-30"
        >
          <Square size={15} />
        </button>
        <button
          disabled={!isRunning || isLoading}
          onClick={() => setConfirmAction("reboot")}
          title="Reboot"
          className="rounded p-1.5 text-[#888888] transition-colors hover:bg-[#222222] hover:text-yellow-400 disabled:cursor-not-allowed disabled:opacity-30"
        >
          <RotateCcw size={15} />
        </button>
      </div>

      <ConfirmDialog
        isOpen={confirmAction === "stop"}
        onClose={() => setConfirmAction(null)}
        onConfirm={onStop}
        title="Confirm Stop"
        message="Are you sure you want to stop this instance? Any unsaved data may be lost."
      />
      <ConfirmDialog
        isOpen={confirmAction === "reboot"}
        onClose={() => setConfirmAction(null)}
        onConfirm={onReboot}
        title="Confirm Reboot"
        message="Are you sure you want to reboot this instance?"
      />
    </>
  );
}
