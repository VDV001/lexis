import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { UserSettings } from "@/types";
import api from "@/lib/api";

interface SettingsState extends UserSettings {
  isLoaded: boolean;
  hydrate: () => Promise<void>;
  updateSettings: (partial: Partial<UserSettings>) => Promise<void>;
}

const defaultSettings: UserSettings = {
  target_language: "en",
  proficiency_level: "b1",
  vocabulary_type: "tech",
  ai_model: "claude-sonnet-4-20250514",
  vocab_goal: 3000,
  ui_language: "ru",
};

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set, get) => ({
      ...defaultSettings,
      isLoaded: false,

      hydrate: async () => {
        try {
          const settings = await api.get<UserSettings>("/users/me/settings");
          set({ ...settings, isLoaded: true });
        } catch {
          set({ isLoaded: true });
        }
      },

      updateSettings: async (partial) => {
        const current = get();
        const updated = { ...current, ...partial };
        await api.put<UserSettings>("/users/me/settings", {
          target_language: updated.target_language,
          proficiency_level: updated.proficiency_level,
          vocabulary_type: updated.vocabulary_type,
          ai_model: updated.ai_model,
          vocab_goal: updated.vocab_goal,
          ui_language: updated.ui_language,
        });
        set(partial);
      },
    }),
    { name: "lexis-settings" }
  )
);
