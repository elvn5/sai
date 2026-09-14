import { useQuery, useQueryClient } from '@tanstack/vue-query';
import { computed } from 'vue';
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

const AUTH_QUERY_KEY = ['auth', 'me'] as const;
const AUTH_STALE_TIME_MS = 60 * 1000;

async function fetchCurrentUser(api: ReturnType<typeof useApiClient>): Promise<AppUser | null> {
  try {
    const res = await api.get<{ user: AppUser }>('/auth/user/me');
    return res.user;
  } catch {
    return null;
  }
}

export function useAuth() {
  const api = useApiClient();
  const queryClient = useQueryClient();

  // Reactive read of the shared ['auth', 'me'] cache entry — kept disabled so
  // mounting useAuth() never triggers its own fetch; checkSession() below
  // (via ensureQueryData) is the single dedup'd entry point that populates it.
  const query = useQuery({
    queryKey: AUTH_QUERY_KEY,
    queryFn: () => fetchCurrentUser(api),
    enabled: false,
    staleTime: AUTH_STALE_TIME_MS,
  });

  function setUser(nextUser: AppUser | null): void {
    queryClient.setQueryData(AUTH_QUERY_KEY, nextUser);
  }

  async function checkSession(): Promise<boolean> {
    const data = await queryClient.ensureQueryData({
      queryKey: AUTH_QUERY_KEY,
      queryFn: () => fetchCurrentUser(api),
      staleTime: AUTH_STALE_TIME_MS,
    });
    return data !== null;
  }

  async function register(payload: RegisterPayload): Promise<void> {
    const res = await api.post<{ user: AppUser }>('/auth/user/register', payload);
    setUser(res.user);
  }

  async function login(payload: LoginPayload): Promise<void> {
    const res = await api.post<{ user: AppUser }>('/auth/user/login', payload);
    setUser(res.user);
  }

  async function loginWithTelegramWidget(payload: TelegramWidgetPayload): Promise<void> {
    const res = await api.post<{ user: AppUser }>('/auth/telegram/widget', payload);
    setUser(res.user);
  }

  async function loginWithTelegramMiniApp(initData: string): Promise<void> {
    const res = await api.post<{ user: AppUser }>('/auth/telegram/miniapp', { initData });
    setUser(res.user);
  }

  async function logout(): Promise<void> {
    await api.post('/auth/user/logout');
    setUser(null);
  }

  return {
    user: computed(() => query.data.value ?? null),
    checked: computed(() => query.isFetched.value),
    checkSession,
    register,
    login,
    loginWithTelegramWidget,
    loginWithTelegramMiniApp,
    logout,
  };
}
