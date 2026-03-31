import { create } from "zustand";
import type { User } from "@/types";

interface SessionState {
  user: User | null;
  accessToken: string | null;
  isAuthenticated: boolean;
  setAuth: (user: User, accessToken: string) => void;
  clearSession: () => void;
}

export const useSessionStore = create<SessionState>((set) => ({
  user: null,
  accessToken: null,
  isAuthenticated: false,
  setAuth: (user, accessToken) => {
    sessionStorage.setItem("access_token", accessToken);
    set({ user, accessToken, isAuthenticated: true });
  },
  clearSession: () => {
    sessionStorage.removeItem("access_token");
    set({ user: null, accessToken: null, isAuthenticated: false });
  },
}));
