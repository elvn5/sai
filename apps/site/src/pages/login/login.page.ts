import type { TelegramWidgetPayload } from '@/core/api/types';
import { AuthStore, type PendingVerification } from '@/core/auth/auth.store';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import { localizedPath } from '@/core/i18n/langs';
import { Notify } from '@/core/notify/notify';
import { bindPageMeta } from '@/core/seo/seo.service';
import { TelegramWebAppStore } from '@/core/telegram/telegram-web-app';
import { AppLogo } from '@/ui/app-logo';
import { CodeInput } from '@/ui/code-input';
import { FieldErrors } from '@/ui/field-errors';
import { MountainScene } from '@/ui/mountain-scene';
import { SiteLink } from '@/ui/site-link';
import { TelegramLoginWidget } from '@/ui/telegram-login-widget';
import { NgTemplateOutlet } from '@angular/common';
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
  viewChild,
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
  imports: [
    AppLogo,
    CodeInput,
    FieldErrors,
    FormField,
    MountainScene,
    NgTemplateOutlet,
    SiteLink,
    TelegramLoginWidget,
    TranslocoPipe,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="ui-container py-6 sm:py-10">
      <div class="mx-auto grid max-w-5xl overflow-hidden rounded-[1.25rem] border-border bg-surface lg:min-h-[40rem] lg:grid-cols-2 lg:border lg:shadow-[0_30px_60px_-24px_rgb(15_23_42/0.25)]">
        <div class="relative isolate hidden overflow-hidden lg:block">
          <app-mountain-scene palette="meadow" fit="bottom" />
          <div class="absolute inset-0 -z-10 bg-linear-to-t from-black/65 to-transparent to-60%"></div>
          <div class="absolute inset-x-7 bottom-8 text-white">
            <span class="flex items-center gap-2 text-[1.06rem] font-extrabold">
              <span class="flex size-[1.875rem] items-center justify-center rounded-[0.5625rem] bg-brand-600"><app-logo class="size-[1.15rem]" /></span>
              {{ 'brand.name' | transloco }}
            </span>
            <p class="mt-4 text-xl leading-snug font-bold">{{ 'login.pitch' | transloco }}</p>
            <ul class="mt-3.5 flex flex-wrap gap-4 text-[0.8rem] text-white/90">
              <li class="flex items-center gap-1"><span class="icon-[heroicons--trophy] size-4" aria-hidden="true"></span>{{ 'login.perks.fame' | transloco }}</li>
              <li class="flex items-center gap-1"><span class="icon-[heroicons--ticket] size-4" aria-hidden="true"></span>{{ 'login.perks.bookings' | transloco }}</li>
              <li class="flex items-center gap-1"><span class="icon-[heroicons--bell] size-4" aria-hidden="true"></span>{{ 'login.perks.telegram' | transloco }}</li>
            </ul>
          </div>
        </div>

        <div class="flex flex-col px-1 py-2 sm:px-8 sm:py-10 lg:px-11 lg:py-12">
          @if (mode() === 'verify' && pending(); as verification) {
            <button type="button" class="ui-icon-btn -ml-2 self-start" [attr.aria-label]="'login.verify.back' | transloco" (click)="backToLogin()">
              <span class="icon-[heroicons--chevron-left] size-5" aria-hidden="true"></span>
            </button>
            <h1 class="mt-1.5 text-[1.375rem] font-extrabold tracking-tight text-text">{{ 'login.verify.title' | transloco }}</h1>
            @if (verification.method === 'email') {
              <p class="mt-1 text-sm text-text-muted">{{ 'login.verify.subtitleEmailLead' | transloco }} <b class="text-text">{{ verification.identifier }}</b></p>
              <form class="mt-5 flex flex-col gap-4" novalidate (submit)="verify($event)">
                <div>
                  <span class="ui-label">{{ 'login.verify.codeLabel' | transloco }}</span>
                  <app-code-input #code [formField]="verifyForm.code" [label]="'login.verify.codeLabel' | transloco" [refused]="error() === 'login.verify.invalidCode'" />
                  <p class="ui-hint mt-1.5">{{ 'login.verify.pasteHint' | transloco }}</p>
                  <app-field-errors [state]="verifyForm.code()" />
                </div>
                <button type="submit" class="ui-btn-primary ui-btn-lg" [disabled]="verifyForm().submitting() || verifyModel().code.length < 6" [attr.aria-busy]="verifyForm().submitting()">
                  @if (verifyForm().submitting()) {
                    <span class="icon-[heroicons--arrow-path] size-4 animate-spin" aria-hidden="true"></span>
                  }
                  {{ 'login.verify.submit' | transloco }}
                </button>
              </form>
              <p class="mt-4 flex items-center justify-center gap-1.5 text-center text-[0.8rem] text-text-muted">
                @if (cooldown() > 0) {
                  <span class="icon-[heroicons--clock] size-4" aria-hidden="true"></span>{{ 'login.verify.resendIn' | transloco }}
                  <b class="text-text tabular-nums">{{ cooldownLabel() }}</b>
                } @else {
                  <button type="button" class="font-medium text-brand-700 hover:underline dark:text-brand-300" (click)="resend()">{{ 'login.verify.resend' | transloco }}</button>
                }
              </p>
              <p class="ui-banner ui-banner-info mt-5 items-start text-[0.8rem]">
                <span class="icon-[heroicons--information-circle]" aria-hidden="true"></span>{{ 'login.verify.noEmail' | transloco }}
              </p>
            } @else {
              <p class="mt-1 text-sm text-text-muted">{{ 'login.verify.subtitleTelegram' | transloco }}</p>
              @if (telegramLink(); as url) {
                <a class="ui-btn-lg mt-6 inline-flex items-center justify-center gap-2 rounded-[0.625rem] bg-[#2aabee] font-semibold text-white hover:bg-[#229ed9]" [href]="url" target="_blank" rel="noopener noreferrer">
                  <span class="icon-[heroicons--paper-airplane] size-[1.1rem]" aria-hidden="true"></span>{{ 'login.verify.openTelegram' | transloco }}
                </a>
              }
              <p class="mt-4 flex items-center justify-center gap-2 text-sm text-text-muted" role="status">
                <span class="icon-[heroicons--arrow-path] size-4 animate-spin" aria-hidden="true"></span>{{ 'login.verify.waitingTelegram' | transloco }}
              </p>
              <p class="mt-3 flex items-center justify-center gap-1.5 text-center text-[0.8rem] text-text-muted">
                @if (cooldown() > 0) {
                  <span class="icon-[heroicons--clock] size-4" aria-hidden="true"></span>{{ 'login.verify.resendLinkIn' | transloco }}
                  <b class="text-text tabular-nums">{{ cooldownLabel() }}</b>
                } @else {
                  <button type="button" class="font-medium text-brand-700 hover:underline dark:text-brand-300" (click)="resend()">{{ 'login.verify.resendLink' | transloco }}</button>
                }
              </p>
            }
            @if (error(); as key) {
              <p class="ui-error mt-4 justify-center" role="alert"><span class="icon-[heroicons--exclamation-circle] size-4" aria-hidden="true"></span>{{ key | transloco }}</p>
            }
          } @else {
            <h1 class="text-2xl font-extrabold tracking-tight text-text sm:text-[1.625rem]">{{ 'login.title' | transloco }}</h1>
            <p class="mt-1 text-text-muted">{{ 'login.subtitle' | transloco }}</p>

            @if (telegram.isTma()) {
              <p class="mt-8 text-text-muted" role="status">{{ 'login.loading' | transloco }}</p>
            } @else {
              <div class="ui-segment mt-5" role="tablist">
                <button type="button" role="tab" [attr.aria-selected]="mode() === 'login'" (click)="switchMode('login')">{{ 'login.tabLogin' | transloco }}</button>
                <button type="button" role="tab" [attr.aria-selected]="mode() === 'register'" (click)="switchMode('register')">{{ 'login.tabRegister' | transloco }}</button>
              </div>

              @if (mode() === 'login') {
                <form class="mt-4 flex flex-col gap-3" novalidate [class.motion-shake]="shake()" (animationend)="shake.set(false)" (submit)="signIn($event)">
                  <label class="block">
                    <span class="ui-label">{{ 'login.identifierLabel' | transloco }}</span>
                    <span class="ui-input-icon">
                      <span class="icon-[heroicons--envelope]" aria-hidden="true"></span>
                      <input class="ui-input" autocomplete="username" placeholder="you@example.com" [formField]="loginForm.identifier" />
                    </span>
                    <app-field-errors [state]="loginForm.identifier()" />
                  </label>
                  <label class="block">
                    <span class="ui-label">{{ 'login.passwordLabel' | transloco }}</span>
                    <span class="ui-input-icon">
                      <span class="icon-[heroicons--lock-closed]" aria-hidden="true"></span>
                      <input class="ui-input pr-10" [type]="showPassword() ? 'text' : 'password'" autocomplete="current-password" [formField]="loginForm.password" [attr.aria-invalid]="error() === 'login.invalidCredentials' ? true : null" />
                      <ng-container *ngTemplateOutlet="eye" />
                    </span>
                    <app-field-errors [state]="loginForm.password()" />
                    @if (error(); as key) {
                      <p class="ui-error mt-1" role="alert"><span class="icon-[heroicons--exclamation-circle] size-4 shrink-0" aria-hidden="true"></span>{{ key | transloco }}</p>
                    }
                  </label>
                  <button type="submit" class="ui-btn-primary ui-btn-lg mt-1" [disabled]="loginForm().submitting()" [attr.aria-busy]="loginForm().submitting()">
                    @if (loginForm().submitting()) {
                      <span class="icon-[heroicons--arrow-path] size-4 animate-spin" aria-hidden="true"></span>{{ 'login.signingIn' | transloco }}
                    } @else {
                      {{ 'login.submitLogin' | transloco }}
                    }
                  </button>
                </form>
              } @else {
                <form class="mt-4 flex flex-col gap-3" novalidate animate.enter="motion-drop-in" [class.motion-shake]="shake()" (animationend)="shake.set(false)" (submit)="signUp($event)">
                  <div class="ui-segment text-xs" role="radiogroup" [attr.aria-label]="'login.identifierLabel' | transloco">
                    <button type="button" role="radio" [attr.aria-checked]="channel() === 'email'" (click)="channel.set('email')">{{ 'login.email' | transloco }}</button>
                    <button type="button" role="radio" [attr.aria-checked]="channel() === 'phone'" (click)="channel.set('phone')">{{ 'login.phone' | transloco }}</button>
                  </div>
                  <label class="block">
                    <span class="ui-label">{{ (channel() === 'email' ? 'login.email' : 'login.phone') | transloco }}</span>
                    <span class="ui-input-icon">
                      @if (channel() === 'email') {
                        <span class="icon-[heroicons--envelope]" aria-hidden="true"></span>
                      } @else {
                        <span class="icon-[heroicons--phone]" aria-hidden="true"></span>
                      }
                      <input class="ui-input" [type]="channel() === 'email' ? 'email' : 'tel'" [attr.autocomplete]="channel() === 'email' ? 'email' : 'tel'" [placeholder]="channel() === 'email' ? 'you@example.com' : '+996 700 000 000'" [formField]="registerForm.identifier" />
                    </span>
                    <span class="ui-hint mt-1 block">{{ (channel() === 'email' ? 'login.verify.hintEmail' : 'login.verify.hintTelegram') | transloco }}</span>
                    <app-field-errors [state]="registerForm.identifier()" />
                  </label>
                  <label class="block">
                    <span class="ui-label">{{ 'login.firstNameLabel' | transloco }}</span>
                    <span class="ui-input-icon">
                      <span class="icon-[heroicons--user]" aria-hidden="true"></span>
                      <input class="ui-input" autocomplete="given-name" [formField]="registerForm.firstName" />
                    </span>
                  </label>
                  <label class="block">
                    <span class="ui-label">{{ 'login.passwordLabel' | transloco }}</span>
                    <span class="ui-input-icon">
                      <span class="icon-[heroicons--lock-closed]" aria-hidden="true"></span>
                      <input class="ui-input pr-10" [type]="showPassword() ? 'text' : 'password'" autocomplete="new-password" [formField]="registerForm.password" />
                      <ng-container *ngTemplateOutlet="eye" />
                    </span>
                    <app-field-errors [state]="registerForm.password()" />
                  </label>
                  @if (error(); as key) {
                    <p class="ui-error" role="alert"><span class="icon-[heroicons--exclamation-circle] size-4 shrink-0" aria-hidden="true"></span>{{ key | transloco }}</p>
                  }
                  <button type="submit" class="ui-btn-primary ui-btn-lg mt-1" [disabled]="registerForm().submitting()" [attr.aria-busy]="registerForm().submitting()">
                    @if (registerForm().submitting()) {
                      <span class="icon-[heroicons--arrow-path] size-4 animate-spin" aria-hidden="true"></span>
                    }
                    {{ 'login.submitRegister' | transloco }}
                  </button>
                </form>
              }

              <div class="my-4 flex items-center gap-2 text-xs text-text-muted">
                <span class="h-px flex-1 bg-border"></span>{{ 'login.orTelegram' | transloco }}<span class="h-px flex-1 bg-border"></span>
              </div>
              @if (botUsername) {
                <div class="flex min-h-11 items-center justify-center rounded-[0.625rem] bg-[#2aabee]/10 p-1">
                  <app-telegram-login-widget (authorized)="withTelegram($event)" />
                </div>
              } @else {
                <button type="button" class="ui-btn-lg inline-flex items-center justify-center gap-2 rounded-[0.625rem] bg-[#2aabee] font-semibold text-white opacity-55" disabled>
                  <span class="icon-[heroicons--paper-airplane] size-[1.1rem]" aria-hidden="true"></span>{{ 'login.withTelegram' | transloco }}
                </button>
                <p class="ui-hint mt-1.5 text-center">{{ 'login.telegramUnavailable' | transloco }}</p>
              }
              <p class="mt-6 text-[0.72rem] text-text-muted lg:mt-auto lg:pt-6">
                {{ 'legal.consent.register' | transloco }}
                <a appSiteLink="/offer" class="text-brand-700 hover:underline dark:text-brand-300">{{ 'legal.consent.offer' | transloco }}</a>
                {{ 'legal.consent.and' | transloco }}
                <a appSiteLink="/privacy" class="text-brand-700 hover:underline dark:text-brand-300">{{ 'legal.consent.privacy' | transloco }}</a>.
              </p>
            }
          }
        </div>
      </div>
    </section>

    <ng-template #eye>
      <button type="button" class="absolute top-1/2 right-1.5 flex size-8 -translate-y-1/2 items-center justify-center rounded-md text-text-muted hover:text-text" [attr.aria-label]="(showPassword() ? 'login.hidePassword' : 'login.showPassword') | transloco" [attr.aria-pressed]="showPassword()" (click)="showPassword.set(!showPassword())">
        <span class="icon-swap" aria-hidden="true">
          <span class="icon-[heroicons--eye] size-[1.1rem]" [attr.data-off]="showPassword() ? '' : null"></span>
          <span class="icon-[heroicons--eye-slash] size-[1.1rem]" [attr.data-off]="showPassword() ? null : ''"></span>
        </span>
      </button>
    </ng-template>
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
  // "0:42" until the code can be sent again.
  protected readonly cooldownLabel = computed(() => {
    const seconds = this.cooldown();
    return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}`;
  });
  protected readonly showPassword = signal(false);
  // The form shakes once when the server refuses.
  protected readonly shake = signal(false);
  private readonly notify = inject(Notify);
  private readonly codeInput = viewChild<CodeInput>('code');
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

  protected readonly verifyModel = signal({ code: '' });
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
        this.shake.set(true);
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
        this.shake.set(true);
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
      this.notify.info({ title: 'notify.codeSent' });
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
    setTimeout(() => this.codeInput()?.focus());
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
    const name = this.auth.user()?.firstName;
    this.notify.success(
      name ? { title: 'notify.welcome', params: { name } } : { title: 'notify.welcomePlain' },
    );
    const target = safeRedirect(this.redirect()) ?? localizedPath(this.lang(), '/');
    await this.router.navigateByUrl(target);
  }
}
