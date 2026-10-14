import { apiErrorMessage, hasStatus } from '@/shared/api/errors';
import type { AdminOrder, OrderStatus, PayeeBalance, PayoutRecord } from '@/shared/api/types';
import { formatDateTime, personName, som } from '@/shared/lib/format';
import { HttpClient, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

const STATUSES: { value: OrderStatus | ''; label: string }[] = [
  { value: '', label: 'Все' },
  { value: 'paid', label: 'Оплачено' },
  { value: 'pending', label: 'Ждёт оплаты' },
  { value: 'failed', label: 'Не оплачено' },
  { value: 'expired', label: 'Время вышло' },
  { value: 'refunded', label: 'Возврат' },
];

const STATUS_CLASS: Record<OrderStatus, string> = {
  pending: 'bg-amber-100 text-amber-800',
  paid: 'bg-green-100 text-green-800',
  failed: 'bg-red-100 text-red-800',
  expired: 'bg-gray-100 text-gray-600',
  refunded: 'bg-sky-100 text-sky-800',
};

// Online booking payments (services/payments): what's owed to partners,
// orders, refunds and payouts.
@Component({
  selector: 'app-payments-page',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-8">
      <div>
        <h1 class="text-xl font-semibold text-text">Платежи</h1>
        <p class="text-sm text-text-muted">
          Онлайн-оплаты броней через Bakai Bank. Деньги приходят на счёт Ermekten; партнёрам выплачивается их часть за вычетом комиссии.
        </p>
        @if (pageError()) {
          <p class="ui-error mt-2">{{ pageError() }}</p>
        }
      </div>

      <section>
        <h2 class="mb-3 text-lg font-semibold text-text">К выплате партнёрам</h2>
        <div class="ui-table-wrap">
          <table class="ui-table">
            <thead>
              <tr><th>Партнёр</th><th>Заказов</th><th>Сумма</th><th><span class="sr-only">Действия</span></th></tr>
            </thead>
            <tbody>
              @for (balance of balances.value() ?? []; track balance.payee.id) {
                <tr>
                  <td>{{ personName(balance.payee) }} <span class="text-xs text-text-muted">{{ balance.payee.id }}</span></td>
                  <td class="text-text-muted">{{ balance.orders }}</td>
                  <td class="font-semibold">{{ som(balance.amount) }}</td>
                  <td class="text-right">
                    <button type="button" class="ui-btn-primary ui-btn-xs" (click)="payOut(balance)">Отметить выплату</button>
                  </td>
                </tr>
              } @empty {
                @if (!balances.isLoading()) {
                  <tr><td colspan="4" class="ui-empty">Никому ничего не должны</td></tr>
                }
              }
            </tbody>
          </table>
        </div>
      </section>

      <section>
        <div class="mb-3 flex items-center justify-between">
          <h2 class="text-lg font-semibold text-text">Заказы</h2>
          <select class="ui-input w-44" aria-label="Статус заказов" [value]="status()" (change)="status.set($any($event.target).value)">
            @for (option of statuses; track option.value) {
              <option [value]="option.value">{{ option.label }}</option>
            }
          </select>
        </div>
        <div class="ui-table-wrap">
          <table class="ui-table">
            <thead>
              <tr>
                <th>Что</th><th>Клиент</th><th>Партнёр</th><th>Сумма</th><th>Статус</th><th>Создан</th>
                <th><span class="sr-only">Действия</span></th>
              </tr>
            </thead>
            <tbody>
              @for (order of orders.value() ?? []; track order.id) {
                <tr>
                  <td>
                    {{ order.itemTitle }}
                    <p class="text-xs text-text-muted">{{ order.itemDate ?? '' }} · {{ order.quantity }} чел.</p>
                  </td>
                  <td>{{ order.contactName }}<p class="text-xs text-text-muted">{{ order.contactPhone }}</p></td>
                  <td class="text-text-muted">{{ personName(order.payee) }}</td>
                  <td>
                    {{ som(order.amount) }}
                    @if (order.fee) {
                      <p class="text-xs text-text-muted">комиссия {{ som(order.fee) }}</p>
                    }
                  </td>
                  <td>
                    <span class="ui-badge" [class]="statusClass[order.status]">{{ statusLabel(order.status) }}</span>
                    @if (order.payoutId) {
                      <p class="text-xs text-text-muted">выплачено</p>
                    }
                    @if (order.refundNote) {
                      <p class="text-xs italic text-text-muted">{{ order.refundNote }}</p>
                    }
                  </td>
                  <td class="text-text-muted">{{ formatDateTime(order.createdAt) }}</td>
                  <td class="text-right">
                    @if (order.status === 'paid' && !order.payoutId) {
                      <button type="button" class="ui-btn-danger ui-btn-xs" (click)="refund(order)">Возврат</button>
                    }
                  </td>
                </tr>
              } @empty {
                @if (!orders.isLoading()) {
                  <tr><td colspan="7" class="ui-empty">Заказов нет</td></tr>
                }
              }
            </tbody>
          </table>
        </div>
      </section>

      <section>
        <h2 class="mb-3 text-lg font-semibold text-text">История выплат</h2>
        <div class="ui-table-wrap">
          <table class="ui-table">
            <thead>
              <tr><th>Когда</th><th>Партнёр</th><th>Заказов</th><th>Сумма</th><th>Комментарий</th></tr>
            </thead>
            <tbody>
              @for (payout of payouts.value() ?? []; track payout.id) {
                <tr>
                  <td class="text-text-muted">{{ formatDateTime(payout.createdAt) }}</td>
                  <td>{{ personName(payout.payee) }}</td>
                  <td class="text-text-muted">{{ payout.orders }}</td>
                  <td class="font-semibold">{{ som(payout.amount) }}</td>
                  <td class="text-text-muted">{{ payout.note ?? '' }}</td>
                </tr>
              } @empty {
                @if (!payouts.isLoading()) {
                  <tr><td colspan="5" class="ui-empty">Выплат ещё не было</td></tr>
                }
              }
            </tbody>
          </table>
        </div>
      </section>
    </div>
  `,
})
export class PaymentsPage {
  private readonly http = inject(HttpClient);
  protected readonly statuses = STATUSES;
  protected readonly statusClass = STATUS_CLASS;
  protected readonly status = signal<OrderStatus | ''>('');
  protected readonly pageError = signal<string | null>(null);

  protected readonly orders = httpResource<AdminOrder[]>(
    () => `/orders/admin${this.status() ? `?status=${this.status()}` : ''}`,
  );
  protected readonly balances = httpResource<PayeeBalance[]>(() => '/payouts/balances');
  protected readonly payouts = httpResource<PayoutRecord[]>(() => '/payouts');

  protected readonly som = som;
  protected readonly personName = personName;
  protected readonly formatDateTime = formatDateTime;

  protected statusLabel(status: OrderStatus): string {
    return STATUSES.find((s) => s.value === status)?.label ?? status;
  }

  private reload(): void {
    this.orders.reload();
    this.balances.reload();
    this.payouts.reload();
  }

  protected async payOut(balance: PayeeBalance): Promise<void> {
    const note = window.prompt(
      `Отметить выплату ${som(balance.amount)} партнёру ${personName(balance.payee)}?\nКомментарий (например, номер перевода):`,
      '',
    );
    if (note === null) return;
    this.pageError.set(null);
    try {
      await firstValueFrom(
        this.http.post('/payouts', { payeeId: balance.payee.id, note: note.trim() || null }),
      );
      this.reload();
    } catch (error) {
      this.pageError.set(apiErrorMessage(error, 'Не удалось записать выплату'));
    }
  }

  protected async refund(order: AdminOrder): Promise<void> {
    const note = window.prompt(
      `Отметить возврат ${som(order.amount)} клиенту ${order.contactName}?\nДеньги нужно вернуть вручную. Комментарий:`,
      '',
    );
    if (note === null) return;
    this.pageError.set(null);
    try {
      await firstValueFrom(
        this.http.post(`/orders/${order.id}/refund`, { note: note.trim() || null }),
      );
      this.reload();
    } catch (error) {
      this.pageError.set(
        hasStatus(error, 409)
          ? 'Возврат возможен только для оплаченных заказов до выплаты партнёру'
          : 'Не удалось отметить возврат',
      );
    }
  }
}
