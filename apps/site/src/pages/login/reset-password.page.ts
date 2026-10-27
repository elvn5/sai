import { AuthStore } from '@/core/auth/auth.store';
import { passwordFitsBcrypt } from '@/core/auth/password';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { Notify } from '@/core/notify/notify';
import { bindPageMeta } from '@/core/seo/seo.service';
import { FieldErrors } from '@/ui/field-errors';
import { SiteLink } from '@/ui/site-link';
import { StateBlock } from '@/ui/state-block';
import { HttpErrorResponse } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, afterNextRender, inject, signal } from '@angular/core';
import { FormField, form, submit, validate, validateStandardSchema } from '@angular/forms/signals';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { z } from 'zod/mini';

// The token from a reset link's fragment ("#<hex>"), or null when there is
// none or it can't be one.
export function readResetToken(hash: string): string | null {
  const token = hash.replace(/^#/, '');
  return /^[A-Za-z0-9]{16,128}$/.test(token) ? token : null;
}

// /reset-password#<token>: the page a password reset link (email or the
// Telegram bot) opens. The token rides in the fragment, which never reaches
// a server or a Referer, and is dropped from the address bar once read.
// Saving the new password signs in here and out everywhere else.
@Component({
  selector: 'app-reset-password-page',
  imports: [FieldErrors, FormField, SiteLink, StateBlock, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="ui-container py-6 sm:py-10">
      <div class="mx-auto max-w-md sm:rounded-[1.25rem] sm:border sm:border-border sm:bg-surface sm:px-8 sm:py-10">
        @switch (state()) {
          @case ('reading') {
            <div class="ui-skeleton h-48 rounded-xl" aria-hidden="true"></div>
          }
          @case ('invalid') {
            <app-state-block icon="icon-[heroicons--link-slash]" tone="warning" [card]="false" title="login.reset.invalidTitle" text="login.reset.invalidText">
              <a appSiteLink="/login" class="ui-btn-primary">{{ 'login.reset.requestNew' | transloco }}</a>
            </app-state-block>
          }
          @default {
            <h1 class="text-[1.375rem] font-extrabold tracking-tight text-text">{{ 'login.reset.title' | transloco }}</h1>
            <p class="mt-1 text-sm text-text-muted">{{ 'login.reset.subtitle' | transloco }}</p>
            <form class="mt-5 flex flex-col gap-4" novalidate [class.motion-shake]="shake()" (animationend)="shake.set(false)" (submit)="save($event)">
              <!-- Lets password managers file the new password under the account. -->
              <input type="text" class="hidden" autocomplete="username" aria-hidden="true" tabindex="-1" />
              <label class="block">
                <span class="ui-label">{{ 'login.reset.passwordLabel' | transloco }}</span>
                <span class="ui-input-icon">
                  <span class="icon-[heroicons--lock-closed]" aria-hidden="true"></span>
                  <input class="ui-input pr-10" [type]="showPassword() ? 'text' : 'password'" autocomplete="new-password" [formField]="resetForm.password" />
                  <button type="button" class="absolute top-1/2 right-1.5 flex size-8 -translate-y-1/2 items-center justify-center rounded-md text-text-muted hover:text-text" [attr.aria-label]="(showPassword() ? 'login.hidePassword' : 'login.showPassword') | transloco" [attr.aria-pressed]="showPassword()" (click)="showPassword.set(!showPassword())">
                    <span class="icon-swap" aria-hidden="true">
                      <span class="icon-[heroicons--eye] size-[1.1rem]" [attr.data-off]="showPassword() ? '' : null"></span>
                      <span class="icon-[heroicons--eye-slash] size-[1.1rem]" [attr.data-off]="showPassword() ? null : ''"></span>
                    </span>
                  </button>
                </span>
                <app-field-errors [state]="resetForm.password()" />
              </label>
              <label class="block">
                <span class="ui-label">{{ 'login.reset.confirmLabel' | transloco }}</span>
                <span class="ui-input-icon">
                  <span class="icon-[heroicons--lock-closed]" aria-hidden="true"></span>
                  <input class="ui-input" [type]="showPassword() ? 'text' : 'password'" autocomplete="new-password" [formField]="resetForm.confirm" />
                </span>
                <app-field-errors [state]="resetForm.confirm()" />
              </label>
              @if (error(); as key) {
                <p class="ui-error" role="alert"><span class="icon-[heroicons--exclamation-circle] size-4 shrink-0" aria-hidden="true"></span>{{ key | transloco }}</p>
              }
              <button type="submit" class="ui-btn-primary ui-btn-lg" [disabled]="resetForm().submitting()" [attr.aria-busy]="resetForm().submitting()">
                @if (resetForm().submitting()) {
                  <span class="icon-[heroicons--arrow-path] size-4 animate-spin" aria-hidden="true"></span>
                }
                {{ 'login.reset.submit' | transloco }}
              </button>
            </form>
          }
        }
      </div>
    </section>
  `,
})
export class ResetPasswordPage {
  private readonly auth = inject(AuthStore);
  private readonly notify = inject(Notify);
  private readonly transloco = inject(TranslocoService);
  private readonly navigate = injectSiteNavigate();
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  // reading: until the fragment is read in the browser.
  protected readonly state = signal<'reading' | 'form' | 'invalid'>('reading');
  private token = '';
  // Translation key of the last failure.
  protected readonly error = signal<string | null>(null);
  protected readonly showPassword = signal(false);
  protected readonly shake = signal(false);

  private readonly resetModel = signal({ password: '', confirm: '' });
  protected readonly resetForm = form(this.resetModel, (path) => {
    const t = (key: string): string => this.transloco.translate(key);
    validateStandardSchema(
      path.password,
      z
        .string()
        .check(
          z.minLength(8, t('login.passwordTooShort')),
          z.refine(passwordFitsBcrypt, t('login.passwordTooLong')),
        ),
    );
    validate(path.confirm, (field) =>
      field.value() !== field.valueOf(path.password)
        ? { kind: 'mismatch', message: t('login.reset.mismatch') }
        : undefined,
    );
  });

  constructor() {
    bindPageMeta(() => ({
      title: this.transloco.translate('login.reset.title'),
      description: this.transloco.translate('login.reset.subtitle'),
      robots: 'noindex',
    }));
    afterNextRender(() => {
      const fragment = this.route.snapshot.fragment ?? '';
      const token = readResetToken(fragment);
      this.token = token ?? '';
      this.state.set(token ? 'form' : 'invalid');
      // Out of the address bar and the history entry.
      if (fragment) void this.router.navigate([], { relativeTo: this.route, replaceUrl: true });
    });
  }

  protected async save(event: Event): Promise<void> {
    event.preventDefault();
    this.error.set(null);
    const valid = await submit(this.resetForm, async () => {
      try {
        await this.auth.resetPassword({ token: this.token, password: this.resetModel().password });
        this.notify.success({ title: 'login.reset.done' });
        await this.navigate('/');
      } catch (error) {
        const status = error instanceof HttpErrorResponse ? error.status : 0;
        // 400 without field errors: the link is spent or expired.
        if (status === 400 && !(error as HttpErrorResponse).error?.fieldErrors) {
          this.state.set('invalid');
        } else {
          this.error.set('login.networkError');
          this.shake.set(true);
        }
      }
      return undefined;
    });
    if (!valid) this.shake.set(true);
  }
}
