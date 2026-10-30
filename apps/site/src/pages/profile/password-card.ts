import type { AppUser } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { passwordFitsBcrypt } from '@/core/auth/password';
import { Notify } from '@/core/notify/notify';
import { FieldErrors } from '@/ui/field-errors';
import { HttpErrorResponse } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, input, signal } from '@angular/core';
import { FormField, form, submit, validate, validateStandardSchema } from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { z } from 'zod/mini';

const EMPTY = { current: '', password: '', confirm: '' };

// Changing the password in the profile (or setting the first one for an
// account made in Telegram that has an email or a phone): the account's
// other sessions end, this one goes on.
@Component({
  selector: 'app-password-card',
  imports: [FieldErrors, FormField, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <section class="ui-card flex flex-col gap-3 p-4" aria-labelledby="password-heading">
      <div>
        <h2 id="password-heading" class="font-semibold text-text">{{ 'profile.password.title' | transloco }}</h2>
        <p class="text-sm text-text-muted">{{ (hasPassword() ? 'profile.password.hint' : 'profile.password.setHint') | transloco }}</p>
      </div>
      @if (!hasPassword() && !user().email && !user().phone) {
        <p class="text-sm text-text-muted">{{ 'profile.password.needContact' | transloco }}</p>
      } @else if (!open()) {
        <button type="button" class="ui-btn-soft self-start" (click)="open.set(true)">
          <span class="icon-[heroicons--key] size-4" aria-hidden="true"></span>{{ (hasPassword() ? 'profile.password.change' : 'profile.password.set') | transloco }}
        </button>
      } @else {
        <form class="flex max-w-md flex-col gap-3" novalidate (submit)="save($event)">
          <!-- Lets password managers file the new password under the account. -->
          <input type="text" class="hidden" autocomplete="username" aria-hidden="true" tabindex="-1" [value]="user().email ?? user().phone ?? ''" />
          @if (hasPassword()) {
            <label class="block">
              <span class="ui-label">{{ 'profile.password.current' | transloco }}</span>
              <input class="ui-input" [type]="shown() ? 'text' : 'password'" autocomplete="current-password" [formField]="pf.current" />
              <app-field-errors [state]="pf.current()" />
            </label>
          }
          <label class="block">
            <span class="ui-label">{{ 'login.reset.passwordLabel' | transloco }}</span>
            <input class="ui-input" [type]="shown() ? 'text' : 'password'" autocomplete="new-password" [formField]="pf.password" />
            <app-field-errors [state]="pf.password()" />
          </label>
          <label class="block">
            <span class="ui-label">{{ 'login.reset.confirmLabel' | transloco }}</span>
            <input class="ui-input" [type]="shown() ? 'text' : 'password'" autocomplete="new-password" [formField]="pf.confirm" />
            <app-field-errors [state]="pf.confirm()" />
          </label>
          <label class="flex cursor-pointer items-center gap-2 text-sm text-text-muted">
            <input type="checkbox" [checked]="shown()" (change)="shown.set($any($event.target).checked)" />{{ 'login.showPassword' | transloco }}
          </label>
          @if (error(); as key) {
            <p class="ui-error" role="alert">{{ key | transloco }}</p>
          }
          <div class="flex flex-wrap gap-2">
            <button type="submit" class="ui-btn-primary" [disabled]="pf().submitting()" [attr.aria-busy]="pf().submitting()">
              {{ (hasPassword() ? 'profile.password.change' : 'profile.password.set') | transloco }}
            </button>
            <button type="button" class="ui-btn-ghost" (click)="close()">{{ 'crm.cancel' | transloco }}</button>
          </div>
        </form>
      }
    </section>
  `,
})
export class PasswordCard {
  readonly user = input.required<AppUser>();

  private readonly auth = inject(AuthStore);
  private readonly notify = inject(Notify);
  private readonly transloco = inject(TranslocoService);

  // Older sessions don't say: take it that there is one.
  protected readonly hasPassword = computed(() => this.user().hasPassword !== false);
  protected readonly open = signal(false);
  protected readonly shown = signal(false);
  // Translation key of a failure the fields don't show.
  protected readonly error = signal<string | null>(null);

  private readonly model = signal({ ...EMPTY });
  protected readonly pf = form(this.model, (path) => {
    const t = (key: string): string => this.transloco.translate(key);
    validate(path.current, (field) =>
      this.hasPassword() && !field.value()
        ? { kind: 'required', message: t('profile.password.currentRequired') }
        : undefined,
    );
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

  protected close(): void {
    this.open.set(false);
    this.error.set(null);
    this.model.set({ ...EMPTY });
    this.pf().reset();
  }

  protected async save(event: Event): Promise<void> {
    event.preventDefault();
    this.error.set(null);
    await submit(this.pf, async () => {
      const { current, password } = this.model();
      try {
        await this.auth.changePassword(
          this.hasPassword() ? { currentPassword: current, password } : { password },
        );
        this.notify.success({ title: 'profile.password.done' });
        this.close();
      } catch (error) {
        const status = error instanceof HttpErrorResponse ? error.status : 0;
        const fieldErrors = (error as HttpErrorResponse | undefined)?.error?.fieldErrors;
        this.error.set(
          status === 400 && !fieldErrors
            ? 'profile.password.wrongCurrent'
            : 'profile.password.error',
        );
      }
      return undefined;
    });
  }
}
