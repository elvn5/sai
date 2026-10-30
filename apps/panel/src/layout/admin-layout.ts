import { AdminAuth } from '@/core/auth/auth';
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';

// The panel's frame: section menu on the left, the section on the right.
@Component({
  selector: 'app-admin-layout',
  imports: [RouterLink, RouterLinkActive, RouterOutlet],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex min-h-screen bg-bg">
      <aside class="w-64 shrink-0 border-r border-border bg-surface p-4">
        <p class="mb-6 flex items-center gap-2 text-lg font-semibold text-text">
          <img src="logo.svg" alt="" class="size-6" />
          Ermekten · админка
        </p>
        <nav class="flex flex-col gap-1" aria-label="Разделы">
          @for (link of links; track link.path) {
            <a
              [routerLink]="link.path"
              routerLinkActive="bg-text/5 font-medium"
              class="rounded px-3 py-2 text-sm text-text hover:bg-text/5"
            >
              {{ link.label }}
            </a>
          }
        </nav>
        <button type="button" class="ui-btn-soft mt-6 w-full" (click)="logout()">Выйти</button>
      </aside>
      <main class="min-w-0 flex-1 p-6">
        <router-outlet />
      </main>
    </div>
  `,
})
export class AdminLayout {
  private readonly auth = inject(AdminAuth);
  private readonly router = inject(Router);

  protected readonly links = [
    { label: 'Поддержка', path: '/support' },
    { label: 'Пользователи', path: '/users' },
    { label: 'Партнёры', path: '/partners' },
    { label: 'Достижения', path: '/achievements' },
    { label: 'Маршруты', path: '/trails' },
    { label: 'Платежи', path: '/payments' },
    { label: 'Онлайн-платежи', path: '/online-payments' },
    { label: 'Блог', path: '/blog' },
    { label: 'SEO', path: '/seo' },
    { label: 'История', path: '/history' },
  ];

  protected async logout(): Promise<void> {
    await this.auth.logout();
    await this.router.navigateByUrl('/login');
  }
}
