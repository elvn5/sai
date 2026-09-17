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

export type VerificationMethod = 'email' | 'telegram';

export interface PendingVerification {
  identifier: string;
  method: VerificationMethod;
  // Only set for the 'telegram' method — embed this in the bot deep link.
  token?: string;
}

export interface VerifyCodePayload {
  identifier: string;
  code: string;
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

  async function register(payload: RegisterPayload): Promise<PendingVerification> {
    const res = await api.post<{ verification: PendingVerification }>(
      '/auth/user/register',
      payload,
    );
    return res.verification;
  }

  // Returns the pending verification info if the account hasn't confirmed
  // its code yet, or null once the session is established.
  async function login(payload: LoginPayload): Promise<PendingVerification | null> {
    const res = await api.post<{ user: AppUser } | { verification: PendingVerification }>(
      '/auth/user/login',
      payload,
    );
    if ('user' in res) {
      setUser(res.user);
      return null;
    }
    return res.verification;
  }

  async function verifyCode(payload: VerifyCodePayload): Promise<void> {
    const res = await api.post<{ user: AppUser }>('/auth/user/verify', payload);
    setUser(res.user);
  }

  async function resendCode(identifier: string): Promise<PendingVerification> {
    const res = await api.post<{ verification: PendingVerification }>('/auth/user/resend-code', {
      identifier,
    });
    return res.verification;
  }

  // Polled while waiting for the Telegram bot to confirm a pending phone
  // verification (there's no code to type for that channel).
  async function checkVerificationStatus(identifier: string): Promise<boolean> {
    const res = await api.get<{ verified: boolean }>(
      `/auth/user/verification-status?identifier=${encodeURIComponent(identifier)}`,
    );
    return res.verified;
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

  // The single entry point every page/layout should use to establish a
  // session: inside a Telegram Mini App there's no login form to show, so
  // Telegram's own initData silently authenticates the user instead of
  // falling back to an existing (usually absent) cookie session. Takes the
  // Telegram state as params rather than calling useTelegramWebApp() itself,
  // since this is invoked from onMounted callbacks where registering a new
  // onMounted (as that composable does) is no longer valid.
  async function ensureSession(tma: { isTma: boolean; initData?: string }): Promise<boolean> {
    if (tma.isTma && tma.initData) {
      try {
        await loginWithTelegramMiniApp(tma.initData);
        return true;
      } catch {
        // Fall through to a normal session check below.
      }
    }
    return checkSession();
  }

  return {
    user: computed(() => query.data.value ?? null),
    checked: computed(() => query.isFetched.value),
    checkSession,
    ensureSession,
    register,
    login,
    verifyCode,
    resendCode,
    checkVerificationStatus,
    loginWithTelegramWidget,
    loginWithTelegramMiniApp,
    logout,
  };
}
