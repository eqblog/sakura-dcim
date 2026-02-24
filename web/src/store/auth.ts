import { create } from 'zustand';
import type { User } from '../types';
import { authAPI } from '../api';

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  fetchUser: () => Promise<void>;
  checkAuth: () => boolean;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isAuthenticated: !!localStorage.getItem('access_token'),
  loading: false,

  login: async (email: string, password: string) => {
    set({ loading: true });
    try {
      const { data: resp } = await authAPI.login({ email, password });
      if (resp.success && resp.data) {
        localStorage.setItem('access_token', resp.data.access_token);
        localStorage.setItem('refresh_token', resp.data.refresh_token);
        set({ user: resp.data.user, isAuthenticated: true, loading: false });
      } else {
        throw new Error(resp.error || 'Login failed');
      }
    } catch (error: any) {
      set({ loading: false });
      throw error?.response?.data?.error || error.message || 'Login failed';
    }
  },

  logout: async () => {
    try {
      await authAPI.logout();
    } finally {
      localStorage.removeItem('access_token');
      localStorage.removeItem('refresh_token');
      set({ user: null, isAuthenticated: false });
    }
  },

  fetchUser: async () => {
    try {
      const { data: resp } = await authAPI.me();
      if (resp.success && resp.data) {
        set({ user: resp.data, isAuthenticated: true });
      }
    } catch {
      set({ user: null, isAuthenticated: false });
      localStorage.removeItem('access_token');
      localStorage.removeItem('refresh_token');
    }
  },

  checkAuth: () => {
    return !!localStorage.getItem('access_token');
  },
}));
