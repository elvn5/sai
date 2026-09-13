import { readonly, ref } from 'vue';
import { useApiClient } from '~/shared/api/client';
import type { TelegramUser, TelegramWidgetPayload } from '~/shared/api/types';

const user = ref<TelegramUser | null>(null);
const checked = ref(false);

export function useAuth() {
  const api = useApiClient();

  async function checkSession(): Promise<boolean> {
    try {
      const res = await api.get<{ user: TelegramUser }>('/auth/telegram/me');
      user.value = res.user;
    } catch {
      user.value = null;
    }
    checked.value = true;
    return user.value !== null;
  }

  async function loginWithWidget(payload: TelegramWidgetPayload): Promise<void> {
    const res = await api.post<{ user: TelegramUser }>('/auth/telegram/widget', payload);
    user.value = res.user;
    checked.value = true;
  }

  async function loginWithMiniApp(initData: string): Promise<void> {
    const res = await api.post<{ user: TelegramUser }>('/auth/telegram/miniapp', { initData });
    user.value = res.user;
    checked.value = true;
  }

  async function logout(): Promise<void> {
    await api.post('/auth/telegram/logout');
    user.value = null;
  }

  return {
    user: readonly(user),
    checked: readonly(checked),
    checkSession,
    loginWithWidget,
    loginWithMiniApp,
    logout,
  };
}
