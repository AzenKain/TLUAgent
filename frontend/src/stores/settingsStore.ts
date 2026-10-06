import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export const SUPPORTED_LANGUAGES = ['vi', 'en'] as const;

export type Language = (typeof SUPPORTED_LANGUAGES)[number];
export type Theme = 'system' | 'light' | 'dark';

// Resolve a persisted language value against the supported set, falling back to Vietnamese.
export const resolveLanguage = (value: unknown): Language =>
  (SUPPORTED_LANGUAGES as readonly string[]).includes(value as string) ? (value as Language) : 'vi';

// SettingsState controls theme, active language, and header settings dropdowns.
export interface SettingsState {
  theme: Theme;
  language: Language;
  isLanguageDropdownOpen: boolean;

  setTheme: (theme: Theme) => void;
  setLanguage: (lang: Language) => void;
  setIsLanguageDropdownOpen: (open: boolean) => void;
  toggleLanguageDropdown: () => void;
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      theme: 'system',
      language: 'vi',
      isLanguageDropdownOpen: false,

      setTheme: (theme) => set({ theme }),
      setLanguage: (language) => set({ language }),
      setIsLanguageDropdownOpen: (isLanguageDropdownOpen) => set({ isLanguageDropdownOpen }),
      toggleLanguageDropdown: () =>
        set((state) => ({ isLanguageDropdownOpen: !state.isLanguageDropdownOpen })),
    }),
    {
      name: 'tluagent-settings',
      partialize: (state) => ({
        theme: state.theme,
        language: state.language,
      }),
    }
  )
);
