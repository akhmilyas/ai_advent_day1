import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { AuthService } from '../services/auth';

interface User {
  id: string;
  username: string;
}

interface AuthState {
  user: User | null;
  token: string | null;

  setAuth: (user: User, token: string) => void;
  logout: () => void;
  isAuthenticated: () => boolean;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      token: null,

      setAuth: (user, token) => {
        set({ user, token });
        AuthService.setToken(token);
      },

      logout: () => {
        set({ user: null, token: null });
        AuthService.logout();
      },

      isAuthenticated: () => !!get().token
    }),
    { name: 'auth-storage' }
  )
);
