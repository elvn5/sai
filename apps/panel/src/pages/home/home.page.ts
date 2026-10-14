import { AdminAuth } from '@/features/auth/auth';
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { Router } from '@angular/router';

// Placeholder of the migration's phase 0: proves the session round trip.
// The admin sections arrive in phase 1.
@Component({
  selector: 'app-home-page',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <main class="mx-auto flex min-h-screen max-w-3xl flex-col justify-center gap-4 px-4">
      <h1 class="flex items-center gap-2 text-2xl font-extrabold text-text">
        <span class="icon-[heroicons--shield-check] size-7 text-brand-600" aria-hidden="true"></span>
        Админка Ermekten
      </h1>
      <p class="text-text-muted">Вы вошли. Разделы админки появятся в фазе 1 миграции.</p>
      <button type="button" class="self-start rounded-lg border border-border px-4 py-2 text-sm" (click)="logout()">
        Выйти
      </button>
    </main>
  `,
})
export class HomePage {
  private readonly auth = inject(AdminAuth);
  private readonly router = inject(Router);

  protected async logout(): Promise<void> {
    await this.auth.logout();
    await this.router.navigateByUrl('/login');
  }
}
