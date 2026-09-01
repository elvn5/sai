import { defineStore } from 'pinia';
import { useApiClient } from '~/shared/api/client';

export const useAuthStore = defineStore('auth', {
  state: () => ({
    authenticated: false,
    checked: false,
  }),
  actions: {
    async login(password: string): Promise<void> {
      const api = useApiClient();
      await api.post('/auth/login', { password });
      this.authenticated = true;
      this.checked = true;
    },
    async logout(): Promise<void> {
      const api = useApiClient();
      await api.post('/auth/logout');
      this.authenticated = false;
    },
    async checkSession(): Promise<boolean> {
      const api = useApiClient();
      try {
        await api.get('/auth/me');
        this.authenticated = true;
      } catch {
        this.authenticated = false;
      }
      this.checked = true;
      return this.authenticated;
    },
  },
});
