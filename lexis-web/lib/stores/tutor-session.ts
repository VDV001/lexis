import { create } from "zustand";
import type { ChatFeedback, Goal } from "@/types";

interface TutorSessionState {
  goals: Goal[];
  feedback: ChatFeedback[];
  words: string[];
  setGoals: (goals: Goal[]) => void;
  addFeedback: (fb: ChatFeedback) => void;
  addWords: (words: string[]) => void;
}

export const useTutorSessionStore = create<TutorSessionState>((set) => ({
  goals: [],
  feedback: [],
  words: [],
  setGoals: (goals) => set({ goals }),
  addFeedback: (fb) =>
    set((state) => ({
      feedback: [fb, ...state.feedback].slice(0, 6),
    })),
  addWords: (newWords) =>
    set((state) => ({
      words: [...new Set([...state.words, ...newWords.map((w) => w.toLowerCase())])].slice(-16),
    })),
}));
