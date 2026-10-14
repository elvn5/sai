import type { AppUser } from '@/shared/api/types';
import { formatDate, fullName } from '@/shared/lib/format';
import { HttpClient, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

@Component({
  selector: 'app-partners-page',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <h1 class="mb-4 text-xl font-semibold text-text">Заявки на партнёрство</h1>
    <div class="ui-table-wrap">
      <table class="ui-table">
        <thead>
          <tr>
            <th>Имя</th>
            <th>Контакты</th>
            <th>Подана</th>
            <th><span class="sr-only">Действия</span></th>
          </tr>
        </thead>
        <tbody>
          @for (user of requests.value() ?? []; track user.id) {
            <tr>
              <td>{{ fullName(user) }}</td>
              <td class="text-text-muted">{{ contact(user) }}</td>
              <td class="text-text-muted">{{ user.partnerRequestedAt ? formatDate(user.partnerRequestedAt) : '—' }}</td>
              <td class="whitespace-nowrap text-right">
                <button type="button" class="ui-btn-primary ui-btn-xs" [disabled]="busyId() === user.id" (click)="decide(user, 'approve')">
                  Одобрить
                </button>
                <button type="button" class="ui-btn-danger ui-btn-xs" [disabled]="busyId() === user.id" (click)="decide(user, 'reject')">
                  Отклонить
                </button>
              </td>
            </tr>
          } @empty {
            @if (!requests.isLoading()) {
              <tr><td colspan="4" class="ui-empty">Заявок нет</td></tr>
            }
          }
        </tbody>
      </table>
    </div>
  `,
})
export class PartnersPage {
  private readonly http = inject(HttpClient);
  protected readonly requests = httpResource<AppUser[]>(() => '/partners/requests');
  protected readonly busyId = signal<string | null>(null);
  protected readonly fullName = fullName;
  protected readonly formatDate = formatDate;

  protected contact(user: AppUser): string {
    return user.email ?? user.phone ?? (user.telegramId ? `tg:${user.telegramId}` : '—');
  }

  protected async decide(user: AppUser, decision: 'approve' | 'reject'): Promise<void> {
    this.busyId.set(user.id);
    try {
      await firstValueFrom(this.http.post(`/partners/${user.id}/${decision}`, {}));
      this.requests.reload();
    } finally {
      this.busyId.set(null);
    }
  }
}
