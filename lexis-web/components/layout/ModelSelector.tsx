"use client";

import { useSettingsStore } from "@/lib/stores/settings";

const MODEL_INFO: Record<string, { icon: string; color: string; label: string }> = {
  "claude-sonnet-4-20250514": { icon: "A", color: "var(--green)", label: "Sonnet" },
  "claude-haiku-4-20250514": { icon: "A", color: "var(--green)", label: "Haiku" },
  "qwen-plus": { icon: "Q", color: "var(--amber)", label: "Qwen" },
  "gpt-4o": { icon: "G", color: "var(--cyan)", label: "GPT-4o" },
  "gpt-4o-mini": { icon: "G", color: "var(--cyan)", label: "Mini" },
  "gemini-2.0-flash": { icon: "✦", color: "var(--purple)", label: "Gemini" },
};

interface ModelSelectorProps {
  onClick?: () => void;
}

export default function ModelSelector({ onClick }: ModelSelectorProps) {
  const aiModel = useSettingsStore((s) => s.ai_model);
  const info = MODEL_INFO[aiModel] || { icon: "?", color: "var(--text3)", label: aiModel };

  return (
    <span
      onClick={onClick}
      className="cursor-pointer transition-all flex items-center gap-[5px]"
      style={{
        fontSize: "10.5px",
        padding: "2px 7px",
        borderRadius: "2px",
        background: "rgba(63,185,80,0.1)",
        color: "var(--green)",
        border: "1px solid rgba(63,185,80,0.25)",
      }}
    >
      <span
        className="inline-flex items-center justify-center shrink-0"
        style={{
          width: "14px",
          height: "14px",
          borderRadius: "50%",
          fontSize: "8px",
          fontWeight: 700,
          color: info.color,
          border: `1px solid ${info.color}`,
        }}
      >
        {info.icon}
      </span>
      {info.label}
    </span>
  );
}
