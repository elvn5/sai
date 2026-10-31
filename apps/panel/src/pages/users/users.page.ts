import { apiErrorMessage } from '@/core/api/errors';
import type { AppUser } from '@/core/api/types';
import { formatDate, fullName } from '@/core/lib/format';
import { HistoryLog } from '@/ui/history-log';
import { Dialog, type DialogRef } from '@angular/cdk/dialog';
import { HttpClient, httpResource } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  type TemplateRef,
  computed,
  inject,
  signal,
  viewChild,
} from '@angular/core';
import { firstValueFrom } from 'rxjs';

// An account that signs in only through Telegram (no email, no password):
// the one kind that can be merged into another account of the same person.
function telegramOnly(user: AppUser): boolean {
  return !user.email && user.hasPassword === false;
}

function matches(user: AppUser, query: string): boolean {
  const q = query.trim().toLowerCase().replace(/^@/, '');
  if (!q) return true;
  return [fullName(user), user.username, user.email, user.phone, user.telegramId]
    .filter(Boolean)
    .some((value) => String(value).toLowerCase().includes(q));
}

// Every account (services/users). A duplicate made by signing in with
// Telegram can be merged into the person's other account: its Telegram,
// tours, favorites and the rest move there (POST /users/:id/merge).
@Component({
  selector: 'app-users-page',
  imports: [HistoryLog],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <h1 class="mb-4 text-xl font-semibold text-text">Пользователи</h1>
    <div class="ui-table-wrap">
      <table class="ui-table">
        <thead>
          <tr>
            <th>Имя</th>
            <th>Username</th>
            <th>Telegram ID</th>
            <th>Email</th>
            <th>Телефон</th>
            <th>Регистрация</th>
            <th><span class="sr-only">Действия</span></th>
          </tr>
        </thead>
        <tbody>
          @for (user of users.value() ?? []; track user.id) {
            <tr>
              <td>{{ fullName(user) }}</td>
              <td>{{ user.username ? '@' + user.username : '—' }}</td>
              <td>{{ user.telegramId ?? '—' }}</td>
              <td>{{ user.email ?? '—' }}</td>
              <td>{{ user.phone ?? '—' }}</td>
              <td>{{ formatDate(user.createdAt) }}</td>
              <td class="whitespace-nowrap text-right">
                @if (telegramOnly(user)) {
                  <button type="button" class="ui-btn-soft ui-btn-xs" (click)="openMerge(user)">Объединить…</button>
                }
              </td>
            </tr>
          } @empty {
            @if (!users.isLoading()) {
              <tr><td colspan="7" class="ui-empty">Пользователей нет</td></tr>
            }
          }
        </tbody>
      </table>
    </div>

    <app-history-log [names]="names()" class="mt-8" [modules]="['users', 'partners']" title="История изменений аккаунтов" />

    <ng-template #merger>
      <form class="ui-dialog flex flex-col gap-4" role="dialog" aria-labelledby="merge-title" (submit)="merge($event)" novalidate>
        <h2 id="merge-title" class="text-lg font-semibold text-text">Объединить · {{ merging() ? fullName(merging()!) : '' }}</h2>
        <p class="ui-hint">
          Этот аккаунт входит только через Telegram. Выберите другой аккаунт того же человека: Telegram, туры, избранное,
          достижения и остальное перейдут туда, а этот аккаунт удалится. Отменить нельзя.
        </p>
        <label class="ui-field">
          Основной аккаунт
          <input
            class="ui-input"
            type="search"
            placeholder="Имя, email, телефон, @username"
            [value]="query()"
            (input)="query.set($any($event.target).value)"
          />
        </label>
        <ul class="max-h-64 overflow-y-auto rounded-lg border border-border" aria-label="Пользователи">
          @for (user of candidates(); track user.id) {
            <li>
              <button
                type="button"
                class="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm hover:bg-text/5"
                [class.bg-brand-500/10]="into()?.id === user.id"
                [attr.aria-pressed]="into()?.id === user.id"
                (click)="into.set(user)"
              >
                <span>{{ fullName(user) }}{{ user.username ? ' · @' + user.username : '' }}</span>
                <span class="text-xs text-text-muted">{{ user.email ?? user.phone ?? '—' }} · {{ formatDate(user.createdAt) }}</span>
              </button>
            </li>
          } @empty {
            <li class="ui-empty">Никого не нашлось</li>
          }
        </ul>
        @if (formError()) {
          <p class="ui-error">{{ formError() }}</p>
        }
        <div class="flex justify-end gap-2">
          <button type="button" class="ui-btn-ghost" (click)="dialogRef?.close()">Отмена</button>
          <button type="submit" class="ui-btn-danger" [disabled]="!into() || busy()">Объединить</button>
        </div>
      </form>
    </ng-template>
  `,
})
export class UsersPage {
  private readonly http = inject(HttpClient);
  private readonly dialog = inject(Dialog);
  private readonly mergerTemplate = viewChild.required<TemplateRef<unknown>>('merger');

  protected readonly users = httpResource<AppUser[]>(() => '/users');
  protected readonly fullName = fullName;
  protected readonly formatDate = formatDate;
  protected readonly telegramOnly = telegramOnly;
  protected readonly names = computed(() =>
    Object.fromEntries((this.users.value() ?? []).map((u) => [u.id, fullName(u)])),
  );

  protected dialogRef: DialogRef | null = null;
  protected readonly merging = signal<AppUser | null>(null);
  protected readonly into = signal<AppUser | null>(null);
  protected readonly query = signal('');
  protected readonly busy = signal(false);
  protected readonly formError = signal<string | null>(null);
  // Accounts the duplicate can go into: any other, those with the same
  // name or username first.
  protected readonly candidates = computed(() => {
    const from = this.merging();
    if (!from) return [];
    const same = (u: AppUser): boolean =>
      (!!from.username && u.username === from.username) || fullName(u) === fullName(from);
    return (this.users.value() ?? [])
      .filter((u) => u.id !== from.id && matches(u, this.query()))
      .sort((a, b) => Number(same(b)) - Number(same(a)))
      .slice(0, 50);
  });

  protected openMerge(user: AppUser): void {
    this.merging.set(user);
    this.into.set(null);
    this.query.set('');
    this.formError.set(null);
    this.dialogRef = this.dialog.open(this.mergerTemplate());
  }

  protected async merge(event: Event): Promise<void> {
    event.preventDefault();
    const from = this.merging();
    const into = this.into();
    if (!from || !into) return;
    this.busy.set(true);
    this.formError.set(null);
    try {
      await firstValueFrom(this.http.post(`/users/${from.id}/merge`, { intoId: into.id }));
      this.dialogRef?.close();
      this.users.reload();
    } catch (error) {
      this.formError.set(apiErrorMessage(error, 'Не удалось объединить аккаунты'));
    } finally {
      this.busy.set(false);
    }
  }
}
