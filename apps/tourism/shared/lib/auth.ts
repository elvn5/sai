import { readonly, ref } from 'vue';
import { useApiClient } from '~/shared/api/client';
import type { AppUser, TelegramWidgetPayload } from '~/shared/api/types';

export interface RegisterPayload {
  email?: string;
  phone?: string;
  password: string;
  firstName?: string;
}

export interface LoginPayload {
  identifier: string;
  password: string;
}

const user = ref<AppUser | null>(null);
const checked = ref(false);

export function useAuth() {
  const api = useApiClient();

  async function checkSession(): Promise<boolean> {
    try {
      const res = await api.get<{ user: AppUser }>('/auth/user/me');
      user.value = res.user;
    } catch {
      user.value = null;
    }
    checked.value = true;
    return user.value !== null;
  }

  async function register(payload: RegisterPayload): Promise<void> {
    const res = await api.post<{ user: AppUser }>('/auth/user/register', payload);
    user.value = res.user;
    checked.value = true;
  }

  async function login(payload: LoginPayload): Promise<void> {
    const res = await api.post<{ user: AppUser }>('/auth/user/login', payload);
    user.value = res.user;
    checked.value = true;
  }

  async function loginWithTelegramWidget(payload: TelegramWidgetPayload): Promise<void> {
    const res = await api.post<{ user: AppUser }>('/auth/telegram/widget', payload);
    user.value = res.user;
    checked.value = true;
  }

  async function loginWithTelegramMiniApp(initData: string): Promise<void> {
    const res = await api.post<{ user: AppUser }>('/auth/telegram/miniapp', { initData });
    user.value = res.user;
    checked.value = true;
  }

  async function logout(): Promise<void> {
    await api.post('/auth/user/logout');
    user.value = null;
  }

  return {
    user: readonly(user),
    checked: readonly(checked),
    checkSession,
    register,
    login,
    loginWithTelegramWidget,
    loginWithTelegramMiniApp,
    logout,
  };
}
