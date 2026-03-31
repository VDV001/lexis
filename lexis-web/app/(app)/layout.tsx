"use client";

import { useState } from "react";
import AppHeader from "@/components/layout/AppHeader";
import AppSidebar from "@/components/layout/AppSidebar";
import SettingsModal from "@/components/layout/SettingsModal";

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const [settingsOpen, setSettingsOpen] = useState(false);

  return (
    <>
      <SettingsModal isOpen={settingsOpen} onClose={() => setSettingsOpen(false)} />
      <AppHeader onOpenSettings={() => setSettingsOpen(true)} />
      <div className="flex flex-1 overflow-hidden">
        <AppSidebar goals={[]} feedback={[]} words={[]} />
        <main className="flex-1 flex flex-col overflow-hidden">
          {children}
        </main>
      </div>
    </>
  );
}
