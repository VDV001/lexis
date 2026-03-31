"use client";

import { useSettingsStore } from "@/lib/stores/settings";
import ModelSelector from "./ModelSelector";

const LEVEL_LABELS: Record<string, string> = {
  a2: "A2", b1: "B1", b2: "B2", c1: "C1",
};

const TYPE_LABELS: Record<string, string> = {
  tech: "⚙️ tech", literary: "📖 literary", business: "💼 business",
};

interface ConfigStripProps {
  onOpenSettings?: () => void;
}

export default function ConfigStrip({ onOpenSettings }: ConfigStripProps) {
  const { target_language, proficiency_level, vocabulary_type } = useSettingsStore();

  return (
    <div
      className="flex items-center gap-[6px]"
      style={{
        padding: "4px 8px",
        background: "var(--bg3)",
        border: "1px solid var(--border)",
        borderRadius: "3px",
      }}
    >
      <span className="text-[10px] text-[var(--text3)] mr-[2px]">язык:</span>
      <span
        className="cursor-pointer transition-all duration-100 whitespace-nowrap"
        style={{
          fontSize: "10.5px",
          padding: "2px 7px",
          borderRadius: "2px",
          background: "rgba(63,185,80,0.1)",
          color: "var(--green)",
          border: "1px solid rgba(63,185,80,0.25)",
        }}
        onClick={onOpenSettings}
      >
        {target_language === "en" ? "🇬🇧 English" : target_language}
      </span>

      <div className="w-[1px] h-[14px] bg-[var(--border)] shrink-0" />

      <span className="text-[10px] text-[var(--text3)] mr-[2px]">уровень:</span>
      <span
        className="cursor-pointer transition-all duration-100"
        style={{
          fontSize: "10.5px",
          padding: "2px 7px",
          borderRadius: "2px",
          background: "rgba(63,185,80,0.1)",
          color: "var(--green)",
          border: "1px solid rgba(63,185,80,0.25)",
        }}
        onClick={onOpenSettings}
      >
        {LEVEL_LABELS[proficiency_level] || proficiency_level}
      </span>

      <div className="w-[1px] h-[14px] bg-[var(--border)] shrink-0" />

      <span className="text-[10px] text-[var(--text3)] mr-[2px]">тип:</span>
      <span
        className="cursor-pointer transition-all duration-100"
        style={{
          fontSize: "10.5px",
          padding: "2px 7px",
          borderRadius: "2px",
          background: "rgba(63,185,80,0.1)",
          color: "var(--green)",
          border: "1px solid rgba(63,185,80,0.25)",
        }}
        onClick={onOpenSettings}
      >
        {TYPE_LABELS[vocabulary_type] || vocabulary_type}
      </span>

      <div className="w-[1px] h-[14px] bg-[var(--border)] shrink-0" />

      <ModelSelector onClick={onOpenSettings} />

      <div className="w-[1px] h-[14px] bg-[var(--border)] shrink-0" />

      <span
        className="cursor-pointer transition-all duration-100"
        style={{ fontSize: "10px", color: "var(--text3)" }}
        onClick={onOpenSettings}
      >
        [ настроить ]
      </span>
    </div>
  );
}
