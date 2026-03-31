"use client";

import NavTabs from "./NavTabs";
import ConfigStrip from "./ConfigStrip";

interface AppHeaderProps {
  onOpenSettings: () => void;
}

export default function AppHeader({ onOpenSettings }: AppHeaderProps) {
  return (
    <header
      className="flex items-center justify-between shrink-0 gap-4"
      style={{
        height: "52px",
        padding: "0 22px",
        background: "var(--bg2)",
        borderBottom: "1px solid var(--border)",
      }}
    >
      <div>
        <div className="text-[17px] font-bold text-[var(--green)] tracking-[-0.5px]">
          lang.tutor
          <span className="inline-block w-[9px] h-[16px] bg-[var(--green)] ml-[2px] align-middle animate-blink" />
        </div>
        <div className="text-[10.5px] text-[var(--text2)] mt-[1px]">
          {">"} AI-наставник для изучения языков
        </div>
      </div>

      <NavTabs />

      <div className="flex items-center gap-[14px] shrink-0">
        <ConfigStrip onOpenSettings={onOpenSettings} />
        <div className="flex items-center gap-[5px] text-[11px] text-[var(--green)]">
          <div
            className="w-[6px] h-[6px] rounded-full bg-[var(--green)]"
            style={{ animation: "pulse 2s ease-in-out infinite" }}
          />
          ONLINE
        </div>
      </div>
    </header>
  );
}
