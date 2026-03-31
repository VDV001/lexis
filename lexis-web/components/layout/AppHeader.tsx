"use client";

import { useRouter } from "next/navigation";
import NavTabs from "./NavTabs";
import ConfigStrip from "./ConfigStrip";
import { useSessionStore } from "@/lib/stores/session";
import api from "@/lib/api";

interface AppHeaderProps {
  onOpenSettings: () => void;
}

export default function AppHeader({ onOpenSettings }: AppHeaderProps) {
  const router = useRouter();
  const clearSession = useSessionStore((s) => s.clearSession);

  function handleLogout() {
    api.post("/auth/logout", {}).catch(() => {});
    clearSession();
    router.push("/login");
  }
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
        <button
          onClick={handleLogout}
          className="cursor-pointer transition-all font-[family-name:var(--font-mono)]"
          style={{
            background: "none",
            border: "1px solid var(--border)",
            borderRadius: "2px",
            padding: "2px 8px",
            fontSize: "10px",
            color: "var(--text3)",
          }}
        >
          выйти
        </button>
      </div>
    </header>
  );
}
