"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import AppHeader from "@/components/layout/AppHeader";
import AppSidebar from "@/components/layout/AppSidebar";
import SettingsModal from "@/components/layout/SettingsModal";
import { useTutorSessionStore } from "@/lib/stores/tutor-session";
import { useSessionStore } from "@/lib/stores/session";
import { useSettingsStore } from "@/lib/stores/settings";
import api from "@/lib/api";
import type { Goal } from "@/types";

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const [settingsOpen, setSettingsOpen] = useState(false);
  const { goals, feedback, words, setGoals } = useTutorSessionStore();
  const tryRestore = useSessionStore((s) => s.tryRestore);
  const isRestoring = useSessionStore((s) => s.isRestoring);
  const isAuthenticated = useSessionStore((s) => s.isAuthenticated);
  const hydrate = useSettingsStore((s) => s.hydrate);
  const router = useRouter();

  // Restore session on mount (handles page reload + new tab)
  useEffect(() => {
    tryRestore().then((ok) => {
      if (!ok) {
        router.replace("/login");
      } else {
        hydrate();
      }
    });
  }, [tryRestore, router, hydrate]);

  useEffect(() => {
    if (isAuthenticated) {
      api.get<Goal[]>("/progress/goals").then((g) => setGoals(g || [])).catch(() => {});
    }
  }, [isAuthenticated, setGoals]);

  // Show loading while restoring session
  if (isRestoring || !isAuthenticated) {
    return (
      <div style={{ flex: 1, display: "flex", alignItems: "center", justifyContent: "center" }}>
        <div
          style={{
            width: 20,
            height: 20,
            border: "2px solid var(--bg4)",
            borderTopColor: "var(--cyan)",
            borderRadius: "50%",
            animation: "spin 0.8s linear infinite",
          }}
        />
      </div>
    );
  }

  return (
    <>
      <SettingsModal isOpen={settingsOpen} onClose={() => setSettingsOpen(false)} />
      <AppHeader onOpenSettings={() => setSettingsOpen(true)} />
      <div className="flex flex-1 overflow-hidden">
        <AppSidebar goals={goals || []} feedback={feedback || []} words={words || []} />
        <main className="flex-1 flex flex-col overflow-hidden">
          {children}
        </main>
      </div>
    </>
  );
}
