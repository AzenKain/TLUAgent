import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { User } from '@/types/auth';

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  setAuth: (user: User) => void;
  setUser: (user: User) => void;
  logout: () => void;
}

interface PersistedAuthState {
  user?: User | null;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      isAuthenticated: false,

      setAuth: (user) =>
        set({
          user,
          isAuthenticated: true,
        }),

      setUser: (user) =>
        set({
          user,
        }),

      logout: () =>
        set({
          user: null,
          isAuthenticated: false,
        }),
    }),
    {
      name: 'tluagent-auth',
      partialize: (state) => ({ user: state.user }),
      merge: (persisted, current) => {
        const saved = (persisted ?? {}) as PersistedAuthState;
        return {
          ...current,
          user: saved.user ?? null,
          isAuthenticated: Boolean(saved.user),
        };
      },
    }
  )
);
