import type { TelegramWidgetPayload } from '@/core/api/types';
import { AuthStore, type PendingVerification } from '@/core/auth/auth.store';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import { localizedPath } from '@/core/i18n/langs';
import { bindPageMeta } from '@/core/seo/seo.service';
import { TelegramWebAppStore } from '@/core/telegram/telegram-web-app';
import { FieldErrors } from '@/ui/field-errors';
import { TelegramLoginWidget } from '@/ui/telegram-login-widget';
import { HttpErrorResponse } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  afterNextRender,
  computed,
  inject,
  input,
  signal,
} from '@angular/core';
import { FormField, form, maxLength, submit, validateStandardSchema } from '@angular/forms/signals';
import { Router } from '@angular/router';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { z } from 'zod/mini';

const RESEND_COOLDOWN_S = 60;
const VERIFY_POLL_MS = 3000;

type Mode = 'login' | 'register' | 'verify';

// A path on this site only, so ?redirect can't send people elsewhere.
export function safeRedirect(target: string | undefined): string | null {
  return target?.startsWith('/') && !target.startsWith('//') ? target : null;
}

// /login: sign in with a password or Telegram, or register (email with a
// code, phone confirmed through the bot). Inside the Telegram Mini App the
// visitor is signed in by Telegram and sent on. ?redirect brings them back.
@Component({
  selector: 'app-login-page',
  imports: [FieldErrors, FormField, TelegramLoginWidget, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="mx-auto flex max-w-sm flex-col px-4 py-16">
      @if (mode() === 'verify' && pending(); as verification) {
        <h1 class="text-center text-2xl font-extrabold text-text">{{ 'login.verify.title' | transloco }}</h1>
        @if (verification.method === 'email') {
          <p class="mt-2 text-center text-text-muted">{{ 'login.verify.subtitleEmail' | transloco: { identifier: verification.identifier } }}</p>
          <form class="mt-8 flex flex-col gap-4" novalidate (submit)="verify($event)">
            <label class="ui-field">
              {{ 'login.verify.codeLabel' | transloco }}
              <input class="ui-input text-center font-mono text-lg tracking-[0.4em]" inputmode="numeric" autocomplete="one-time-code" placeholder="000000" [formField]="verifyForm.code" />
              <app-field-errors [state]="verifyForm.code()" />
            </label>
            <button type="submit" class="ui-btn-primary py-2.5" [disabled]="verifyForm().submitting()">{{ 'login.verify.submit' | transloco }}</button>
          </form>
          <button type="button" class="mt-4 text-center text-sm text-brand-700 disabled:text-text-muted dark:text-brand-300" [disabled]="cooldown() > 0" (click)="resend()">
            @if (cooldown() > 0) { {{ 'login.verify.resendCooldown' | transloco: { seconds: cooldown() } }} } @else { {{ 'login.verify.resend' | transloco }} }
          </button>
        } @else {
          <p class="mt-2 text-center text-text-muted">{{ 'login.verify.subtitleTelegram' | transloco }}</p>
          @if (telegramLink(); as url) {
            <a class="ui-btn-primary mt-8 py-2.5" [href]="url" target="_blank" rel="noopener noreferrer">{{ 'login.verify.openTelegram' | transloco }}</a>
          }
          <p class="mt-4 flex items-center justify-center gap-2 text-sm text-text-muted" role="status">
            <span class="icon-[heroicons--arrow-path] size-4 animate-spin" aria-hidden="true"></span>{{ 'login.verify.waitingTelegram' | transloco }}
          </p>
          <button type="button" class="mt-4 text-center text-sm text-brand-700 disabled:text-text-muted dark:text-brand-300" [disabled]="cooldown() > 0" (click)="resend()">
            @if (cooldown() > 0) { {{ 'login.verify.resendLinkCooldown' | transloco: { seconds: cooldown() } }} } @else { {{ 'login.verify.resendLink' | transloco }} }
          </button>
        }
        <button type="button" class="mt-2 text-center text-sm text-text-muted hover:underline" (click)="backToLogin()">{{ 'login.verify.back' | transloco }}</button>
      } @else {
        <h1 class="text-center text-2xl font-extrabold text-text">{{ 'login.title' | transloco }}</h1>
        <p class="mt-2 text-center text-text-muted">{{ 'login.subtitle' | transloco }}</p>

        @if (telegram.isTma()) {
          <p class="mt-8 text-center text-text-muted" role="status">{{ 'login.loading' | transloco }}</p>
        } @else {
          <div class="mt-8 flex rounded-full border border-border bg-surface p-1" role="tablist">
            <button type="button" role="tab" class="flex-1 rounded-full py-2 text-sm font-medium transition-colors" [class]="mode() === 'login' ? 'bg-brand-600 text-white' : 'text-text-muted'" [attr.aria-selected]="mode() === 'login'" (click)="switchMode('login')">{{ 'login.tabLogin' | transloco }}</button>
            <button type="button" role="tab" class="flex-1 rounded-full py-2 text-sm font-medium transition-colors" [class]="mode() === 'register' ? 'bg-brand-600 text-white' : 'text-text-muted'" [attr.aria-selected]="mode() === 'register'" (click)="switchMode('register')">{{ 'login.tabRegister' | transloco }}</button>
          </div>

          @if (mode() === 'login') {
            <form class="mt-6 flex flex-col gap-4" novalidate (submit)="signIn($event)">
              <label class="ui-field">
                {{ 'login.identifierLabel' | transloco }}
                <input class="ui-input" autocomplete="username" placeholder="you@example.com" [formField]="loginForm.identifier" />
                <app-field-errors [state]="loginForm.identifier()" />
              </label>
              <label class="ui-field">
                {{ 'login.passwordLabel' | transloco }}
                <input class="ui-input" type="password" autocomplete="current-password" [formField]="loginForm.password" />
                <app-field-errors [state]="loginForm.password()" />
              </label>
              <button type="submit" class="ui-btn-primary py-2.5" [disabled]="loginForm().submitting()">{{ 'login.submitLogin' | transloco }}</button>
            </form>
          } @else {
            <form class="mt-6 flex flex-col gap-4" novalidate (submit)="signUp($event)">
              <div class="flex rounded-full border border-border p-1 text-sm" role="radiogroup" [attr.aria-label]="'login.identifierLabel' | transloco">
                <button type="button" role="radio" class="flex-1 rounded-full py-1.5 transition-colors" [class]="channel() === 'email' ? 'bg-text/10 font-medium text-text' : 'text-text-muted'" [attr.aria-checked]="channel() === 'email'" (click)="channel.set('email')">{{ 'login.email' | transloco }}</button>
                <button type="button" role="radio" class="flex-1 rounded-full py-1.5 transition-colors" [class]="channel() === 'phone' ? 'bg-text/10 font-medium text-text' : 'text-text-muted'" [attr.aria-checked]="channel() === 'phone'" (click)="channel.set('phone')">{{ 'login.phone' | transloco }}</button>
              </div>
              <p class="text-xs text-text-muted">{{ (channel() === 'email' ? 'login.verify.hintEmail' : 'login.verify.hintTelegram') | transloco }}</p>
              <label class="ui-field">
                {{ (channel() === 'email' ? 'login.email' : 'login.phone') | transloco }}
                <input class="ui-input" [type]="channel() === 'email' ? 'email' : 'tel'" [attr.autocomplete]="channel() === 'email' ? 'email' : 'tel'" [placeholder]="channel() === 'email' ? 'you@example.com' : '+996 700 000 000'" [formField]="registerForm.identifier" />
                <app-field-errors [state]="registerForm.identifier()" />
              </label>
              <label class="ui-field">
                {{ 'login.firstNameLabel' | transloco }}
                <input class="ui-input" autocomplete="given-name" [formField]="registerForm.firstName" />
              </label>
              <label class="ui-field">
                {{ 'login.passwordLabel' | transloco }}
                <input class="ui-input" type="password" autocomplete="new-password" [formField]="registerForm.password" />
                <app-field-errors [state]="registerForm.password()" />
              </label>
              <button type="submit" class="ui-btn-primary py-2.5" [disabled]="registerForm().submitting()">{{ 'login.submitRegister' | transloco }}</button>
            </form>
          }
        }
      }

      @if (error(); as key) {
        <p class="ui-error mt-4 text-center" role="alert">{{ key | transloco }}</p>
      }

      @if (mode() !== 'verify' && !telegram.isTma()) {
        <div class="my-6 flex items-center gap-3 text-xs text-text-muted">
          <span class="h-px flex-1 bg-border"></span>{{ 'login.orTelegram' | transloco }}<span class="h-px flex-1 bg-border"></span>
        </div>
        @if (botUsername) {
          <app-telegram-login-widget (authorized)="withTelegram($event)" />
        } @else {
          <p class="text-center text-sm text-text-muted">{{ 'login.telegramUnavailable' | transloco }}</p>
        }
      }
    </section>
  `,
})
export class LoginPage {
  readonly redirect = input<string | undefined>(undefined);

  private readonly auth = inject(AuthStore);
  protected readonly telegram = inject(TelegramWebAppStore);
  private readonly transloco = inject(TranslocoService);
  private readonly router = inject(Router);
  private readonly lang = injectLang();
  protected readonly botUsername = inject(APP_CONFIG).telegramBotUsername;

  protected readonly mode = signal<Mode>('login');
  protected readonly channel = signal<'email' | 'phone'>('email');
  // Translation key of the last failure.
  protected readonly error = signal<string | null>(null);
  protected readonly pending = signal<PendingVerification | null>(null);
  private pendingPassword = '';
  protected readonly cooldown = signal(0);
  private cooldownTimer: ReturnType<typeof setInterval> | undefined;
  private pollTimer: ReturnType<typeof setInterval> | undefined;

  protected readonly telegramLink = computed(() => {
    const token = this.pending()?.token;
    return token && this.botUsername
      ? `https://t.me/${this.botUsername}?start=verify_${token}`
      : null;
  });

  private readonly loginModel = signal({ identifier: '', password: '' });
  protected readonly loginForm = form(this.loginModel, (path) => {
    const required = this.transloco.translate('login.required');
    validateStandardSchema(
      path,
      z.object({
        identifier: z.string().check(z.trim(), z.minLength(1, required)),
        password: z.string().check(z.minLength(1, required)),
      }),
    );
  });

  private readonly registerModel = signal({ identifier: '', firstName: '', password: '' });
  protected readonly registerForm = form(this.registerModel, (path) => {
    const t = (key: string): string => this.transloco.translate(key);
    validateStandardSchema(
      path,
      z.object({
        identifier: z.string().check(z.trim(), z.minLength(1, t('login.required'))),
        password: z.string().check(z.minLength(8, t('login.passwordTooShort'))),
      }),
    );
  });

  private readonly verifyModel = signal({ code: '' });
  protected readonly verifyForm = form(this.verifyModel, (path) => {
    maxLength(path.code, 6);
    validateStandardSchema(
      path,
      z.object({
        code: z
          .string()
          .check(z.regex(/^\d{6}$/, this.transloco.translate('login.verify.invalidFormat'))),
      }),
    );
  });

  constructor() {
    afterNextRender(() => void this.signInInsideTelegram());
    inject(DestroyRef).onDestroy(() => this.stopTimers());
    bindPageMeta(() => ({
      title: this.transloco.translate('login.title'),
      description: this.transloco.translate('login.subtitle'),
      robots: 'noindex',
    }));
  }

  protected switchMode(mode: Mode): void {
    this.mode.set(mode);
    this.error.set(null);
  }

  protected async signIn(event: Event): Promise<void> {
    event.preventDefault();
    this.error.set(null);
    await submit(this.loginForm, async () => {
      const { identifier, password } = this.loginModel();
      try {
        const verification = await this.auth.login({ identifier: identifier.trim(), password });
        if (verification) {
          this.pendingPassword = password;
          this.startVerification(verification);
        } else {
          await this.leave();
        }
      } catch (error) {
        this.error.set(this.isRejected(error) ? 'login.invalidCredentials' : 'login.networkError');
      }
      return undefined;
    });
  }

  protected async signUp(event: Event): Promise<void> {
    event.preventDefault();
    this.error.set(null);
    await submit(this.registerForm, async () => {
      const { identifier, firstName, password } = this.registerModel();
      const email = this.channel() === 'email';
      try {
        const verification = await this.auth.register({
          email: email ? identifier.trim() : undefined,
          phone: email ? undefined : identifier.trim(),
          password,
          firstName: firstName.trim() || undefined,
        });
        this.pendingPassword = password;
        this.startVerification(verification);
      } catch (error) {
        // 500/503: the code couldn't be sent; anything else: the account exists.
        this.error.set(
          error instanceof HttpErrorResponse && error.status >= 400 && error.status < 500
            ? 'login.registerError'
            : 'login.verify.sendError',
        );
      }
      return undefined;
    });
  }

  protected async verify(event: Event): Promise<void> {
    event.preventDefault();
    const verification = this.pending();
    if (!verification) return;
    this.error.set(null);
    await submit(this.verifyForm, async () => {
      try {
        await this.auth.verifyCode({
          identifier: verification.identifier,
          code: this.verifyModel().code,
        });
        await this.leave();
      } catch (error) {
        this.error.set(this.isRejected(error) ? 'login.verify.invalidCode' : 'login.networkError');
      }
      return undefined;
    });
  }

  protected async resend(): Promise<void> {
    const verification = this.pending();
    if (!verification || this.cooldown() > 0) return;
    this.error.set(null);
    try {
      this.pending.set(await this.auth.resendCode(verification.identifier));
      this.startCooldown();
    } catch {
      this.error.set('login.verify.resendError');
    }
  }

  protected backToLogin(): void {
    this.stopTimers();
    this.pending.set(null);
    this.pendingPassword = '';
    this.error.set(null);
    this.mode.set('login');
  }

  protected async withTelegram(payload: TelegramWidgetPayload): Promise<void> {
    try {
      await this.auth.loginWithTelegramWidget(payload);
      await this.leave();
    } catch {
      this.error.set('login.error');
    }
  }

  // Inside the Mini App Telegram vouches for the visitor (initData).
  private async signInInsideTelegram(): Promise<void> {
    await this.telegram.ready;
    if (!this.telegram.isTma()) return;
    if (await this.auth.ensureSession()) await this.leave();
    else this.error.set('login.error');
  }

  private startVerification(verification: PendingVerification): void {
    this.pending.set(verification);
    this.verifyModel.set({ code: '' });
    this.error.set(null);
    this.mode.set('verify');
    this.startCooldown();
    // A phone is confirmed in the bot: wait for it, then sign in.
    if (verification.method === 'telegram') {
      clearInterval(this.pollTimer);
      this.pollTimer = setInterval(() => void this.checkTelegramConfirmation(), VERIFY_POLL_MS);
    }
  }

  private async checkTelegramConfirmation(): Promise<void> {
    const verification = this.pending();
    if (!verification) return;
    try {
      if (!(await this.auth.checkVerificationStatus(verification.identifier))) return;
      clearInterval(this.pollTimer);
      const again = await this.auth.login({
        identifier: verification.identifier,
        password: this.pendingPassword,
      });
      if (!again) await this.leave();
    } catch {
      // A network hiccup: keep waiting.
    }
  }

  private startCooldown(): void {
    this.cooldown.set(RESEND_COOLDOWN_S);
    clearInterval(this.cooldownTimer);
    this.cooldownTimer = setInterval(() => {
      this.cooldown.update((seconds) => seconds - 1);
      if (this.cooldown() <= 0) clearInterval(this.cooldownTimer);
    }, 1000);
  }

  private stopTimers(): void {
    clearInterval(this.cooldownTimer);
    clearInterval(this.pollTimer);
  }

  // The API answered (wrong password, bad code), as opposed to no answer.
  private isRejected(error: unknown): boolean {
    return error instanceof HttpErrorResponse && error.status > 0 && error.status < 500;
  }

  // Signed in: back to ?redirect (already in its language), else home.
  private async leave(): Promise<void> {
    const target = safeRedirect(this.redirect()) ?? localizedPath(this.lang(), '/');
    await this.router.navigateByUrl(target);
  }
}
