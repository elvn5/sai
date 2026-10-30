import type { AppUser } from '@/core/api/types';
import { formatDate, fullName } from '@/core/lib/format';
import { HistoryLog } from '@/ui/history-log';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed } from '@angular/core';

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
            </tr>
          } @empty {
            @if (!users.isLoading()) {
              <tr><td colspan="6" class="ui-empty">Пользователей нет</td></tr>
            }
          }
        </tbody>
      </table>
    </div>

    <app-history-log [names]="names()" class="mt-8" [modules]="['users', 'partners']" title="История изменений аккаунтов" />
  `,
})
export class UsersPage {
  protected readonly users = httpResource<AppUser[]>(() => '/users');
  protected readonly fullName = fullName;
  protected readonly names = computed(() =>
    Object.fromEntries((this.users.value() ?? []).map((u) => [u.id, fullName(u)])),
  );
  protected readonly formatDate = formatDate;
}
