import { AdminAuth } from '@/features/auth/auth';
import { HttpErrorResponse } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormField, form, submit, validateStandardSchema } from '@angular/forms/signals';
import { Router } from '@angular/router';
import * as z from 'zod/mini';

// Zod schemas plug into Signal Forms through Standard Schema. zod/mini keeps
// the browser bundle small (the full zod build adds ~70 kB gzipped).
const loginSchema = z.object({
  password: z.string().check(z.minLength(1, 'Введите пароль')),
});

@Component({
  selector: 'app-login-page',
  imports: [FormField],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <main class="flex min-h-screen items-center justify-center bg-bg px-4">
      <form
        class="flex w-full max-w-sm flex-col gap-4 rounded-card border border-border bg-surface p-6 shadow-sm"
        (submit)="onSubmit($event)"
        novalidate
      >
        <h1 class="text-lg font-semibold text-text">Вход в админку</h1>
        <label class="flex flex-col gap-1 text-sm text-text">
          Пароль
          <input
            type="password"
            placeholder="Пароль"
            class="rounded-lg border border-border bg-bg px-3 py-2"
            [formField]="loginForm.password"
          />
        </label>
        @if (loginForm.password().touched() && loginForm.password().invalid()) {
          @for (error of loginForm.password().errors(); track $index) {
            <p class="text-sm text-red-600">{{ error.message }}</p>
          }
        }
        @if (failure()) {
          <p class="text-sm text-red-600">{{ failure() }}</p>
        }
        <button
          type="submit"
          class="rounded-lg bg-brand-600 px-4 py-2 font-medium text-white disabled:opacity-60"
          [disabled]="loginForm().submitting()"
        >
          Войти
        </button>
      </form>
    </main>
  `,
})
export class LoginPage {
  private readonly auth = inject(AdminAuth);
  private readonly router = inject(Router);

  protected readonly model = signal({ password: '' });
  protected readonly loginForm = form(this.model, (path) => {
    validateStandardSchema(path, loginSchema);
  });
  protected readonly failure = signal<string | null>(null);

  protected async onSubmit(event: Event): Promise<void> {
    event.preventDefault();
    this.failure.set(null);
    await submit(this.loginForm, async () => {
      try {
        await this.auth.login(this.model().password);
        await this.router.navigateByUrl('/');
      } catch (error) {
        this.failure.set(
          error instanceof HttpErrorResponse && error.status === 401
            ? 'Неверный пароль'
            : 'Ошибка сети',
        );
      }
      return undefined;
    });
  }
}
