"use client";

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export default function SettingsModal({ isOpen, onClose }: SettingsModalProps) {
  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      style={{ background: "rgba(0,0,0,0.6)" }}
      onClick={onClose}
    >
      <div
        className="relative"
        style={{
          width: "500px",
          padding: "24px",
          background: "var(--bg2)",
          border: "1px solid var(--border)",
          borderRadius: "6px",
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="text-[13px] text-[var(--text2)]">
          // Настройки — Phase 2
        </div>
        <button
          className="absolute top-[12px] right-[12px] text-[var(--text3)] hover:text-[var(--text)] text-[14px] cursor-pointer"
          onClick={onClose}
        >
          x
        </button>
      </div>
    </div>
  );
}
