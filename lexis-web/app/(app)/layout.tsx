"use client";

import { useEffect, useState } from "react";
import AppHeader from "@/components/layout/AppHeader";
import AppSidebar from "@/components/layout/AppSidebar";
import SettingsModal from "@/components/layout/SettingsModal";
import { useTutorSessionStore } from "@/lib/stores/tutor-session";
import api from "@/lib/api";
import type { Goal } from "@/types";

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const [settingsOpen, setSettingsOpen] = useState(false);
  const { goals, feedback, words, setGoals } = useTutorSessionStore();

  useEffect(() => {
    api.get<Goal[]>("/progress/goals").then(setGoals).catch(() => {});
  }, [setGoals]);

  return (
    <>
      <SettingsModal isOpen={settingsOpen} onClose={() => setSettingsOpen(false)} />
      <AppHeader onOpenSettings={() => setSettingsOpen(true)} />
      <div className="flex flex-1 overflow-hidden">
        <AppSidebar goals={goals} feedback={feedback} words={words} />
        <main className="flex-1 flex flex-col overflow-hidden">
          {children}
        </main>
      </div>
    </>
  );
}
