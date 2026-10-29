import {
  type AppUser,
  type PartnerKind,
  PartnerStatus,
  type TelegramWidgetPayload,
} from '@/core/api/types';
import { TelegramWebAppStore } from '@/core/telegram/telegram-web-app';
import { HttpClient } from '@angular/common/http';
import { Injectable, computed, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

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
  // Only for 'telegram': goes into the bot's t.me deep link.
  token?: string;
}

export interface VerifyCodePayload {
  identifier: string;
  code: string;
}

// Where a password reset link goes: the account's email, or the Telegram
// chat linked to it.
export type ResetChannel = 'email' | 'telegram';

export interface ForgotPasswordPayload {
  identifier: string;
  channel: ResetChannel;
  // The site's language, for the email and the link.
  lang: string;
}

export interface ResetPasswordPayload {
  // From the link's #fragment.
  token: string;
  password: string;
}

// The signed-in visitor (cookie user_token, set by the API). Pages render
// on the server anonymously, so the session is only ever checked in the
// browser, after hydration.
@Injectable({ providedIn: 'root' })
export class AuthStore {
  private readonly http = inject(HttpClient);
  private readonly telegram = inject(TelegramWebAppStore);
  private readonly userSignal = signal<AppUser | null>(null);
  private readonly checkedSignal = signal(false);
  private pending: Promise<void> | null = null;

  readonly user = this.userSignal.asReadonly();
  // Whether the session check has finished (user() is then final).
  readonly checked = this.checkedSignal.asReadonly();
  // Only approved partners publish hotels, guides, photos and posts.
  readonly isPartner = computed(() => this.userSignal()?.partnerStatus === PartnerStatus.APPROVED);
  // An approved partner who carries tour groups: their cabinet is the
  // driver's CRM, and they don't publish tours.
  readonly isDriver = computed(
    () => this.isPartner() && this.userSignal()?.partnerKind === 'driver',
  );
  readonly displayName = computed(() => {
    const user = this.userSignal();
    return user?.firstName ?? user?.username ?? '';
  });

  // The one entry point for establishing the session, checked once per page
  // load: inside a Telegram Mini App, Telegram's initData signs the visitor
  // in (there is no login form there); otherwise the cookie decides.
  // Resolves to whether someone is signed in now: a later sign-in or
  // sign-out counts, not just the first check.
  async ensureSession(): Promise<boolean> {
    this.pending ??= this.establish();
    await this.pending;
    return this.userSignal() !== null;
  }

  async register(payload: RegisterPayload): Promise<PendingVerification> {
    const res = await this.post<{ verification: PendingVerification }>(
      '/auth/user/register',
      payload,
    );
    return res.verification;
  }

  // The pending verification when the account hasn't confirmed its code
  // yet, null once signed in.
  async login(payload: LoginPayload): Promise<PendingVerification | null> {
    const res = await this.post<{ user: AppUser } | { verification: PendingVerification }>(
      '/auth/user/login',
      payload,
    );
    if ('user' in res) {
      this.setUser(res.user);
      return null;
    }
    return res.verification;
  }

  async verifyCode(payload: VerifyCodePayload): Promise<void> {
    this.setUser((await this.post<{ user: AppUser }>('/auth/user/verify', payload)).user);
  }

  // Asks for a one-time link for setting a new password. The answer is the
  // same whether or not the account exists (or has that channel).
  async forgotPassword(payload: ForgotPasswordPayload): Promise<void> {
    await this.post('/auth/user/password/forgot', payload);
  }

  // Sets the new password with the link's token, and signs in; the
  // account's other sessions end.
  async resetPassword(payload: ResetPasswordPayload): Promise<void> {
    this.setUser((await this.post<{ user: AppUser }>('/auth/user/password/reset', payload)).user);
  }

  async resendCode(identifier: string): Promise<PendingVerification> {
    const res = await this.post<{ verification: PendingVerification }>('/auth/user/resend-code', {
      identifier,
    });
    return res.verification;
  }

  // Polled while the Telegram bot confirms a phone registration (no code
  // to type on that channel).
  async checkVerificationStatus(identifier: string): Promise<boolean> {
    const res = await firstValueFrom(
      this.http.get<{ verified: boolean }>('/auth/user/verification-status', {
        params: { identifier },
      }),
    );
    return res.verified;
  }

  async loginWithTelegramWidget(payload: TelegramWidgetPayload): Promise<void> {
    this.setUser((await this.post<{ user: AppUser }>('/auth/telegram/widget', payload)).user);
  }

  async loginWithTelegramMiniApp(initData: string): Promise<void> {
    this.setUser((await this.post<{ user: AppUser }>('/auth/telegram/miniapp', { initData })).user);
  }

  // Re-reads the account (e.g. after the bot linked Telegram to it).
  async refresh(): Promise<AppUser | null> {
    const user = await this.fetchUser();
    this.setUser(user);
    return user;
  }

  // A one-time token for the bot's t.me/<bot>?start=link_<token> deep link.
  startTelegramLink(): Promise<{ token: string; expiresAt: string }> {
    return this.post<{ token: string; expiresAt: string }>('/auth/user/telegram/link', {});
  }

  async unlinkTelegram(): Promise<void> {
    this.setUser(
      (await firstValueFrom(this.http.delete<{ user: AppUser }>('/auth/user/telegram'))).user,
    );
  }

  // Asks the admin to make this account a partner (status becomes pending).
  async applyAsPartner(kind: PartnerKind = 'organizer'): Promise<void> {
    this.setUser((await this.post<{ user: AppUser }>('/partners/apply', { kind })).user);
  }

  async logout(): Promise<void> {
    await this.post('/auth/user/logout', {});
    this.setUser(null);
  }

  private async establish(): Promise<void> {
    await this.telegram.ready;
    const initData = this.telegram.initData();
    if (initData) {
      try {
        await this.loginWithTelegramMiniApp(initData);
        return;
      } catch {
        // Fall back to the cookie.
      }
    }
    this.setUser(await this.fetchUser());
  }

  private async fetchUser(): Promise<AppUser | null> {
    try {
      return (await firstValueFrom(this.http.get<{ user: AppUser }>('/auth/user/me'))).user;
    } catch {
      return null;
    }
  }

  private setUser(user: AppUser | null): void {
    this.userSignal.set(user);
    this.checkedSignal.set(true);
  }

  private post<T>(path: string, body: unknown): Promise<T> {
    return firstValueFrom(this.http.post<T>(path, body));
  }
}
