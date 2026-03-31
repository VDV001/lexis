"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import AppHeader from "@/components/layout/AppHeader";
import AppSidebar from "@/components/layout/AppSidebar";
import SettingsModal from "@/components/layout/SettingsModal";
import { useTutorSessionStore } from "@/lib/stores/tutor-session";
import { useSessionStore } from "@/lib/stores/session";
import api from "@/lib/api";
import type { Goal } from "@/types";

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const [settingsOpen, setSettingsOpen] = useState(false);
  const { goals, feedback, words, setGoals } = useTutorSessionStore();
  const isAuthenticated = useSessionStore((s) => s.isAuthenticated);
  const router = useRouter();

  // Client-side auth check
  useEffect(() => {
    const token = sessionStorage.getItem("access_token");
    if (!token && !isAuthenticated) {
      router.replace("/login");
    }
  }, [isAuthenticated, router]);

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
