import { create } from "zustand";

export type AppTheme = "archive" | "paper";

interface AppState {
  setTheme: (theme: AppTheme) => void;
  theme: AppTheme;
}

export const useAppStore = create<AppState>((set) => ({
  theme: "archive",
  setTheme: (theme) => set({ theme }),
}));
