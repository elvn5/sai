import { apiErrorMessage, hasStatus } from '@/core/api/errors';
import type { PaymentGateway, Transaction } from '@/core/api/types';
import { formatDateTime, personName, som } from '@/core/lib/format';
import {
  gatewayAmount,
  gatewayName,
  orderStatusLabel,
  transactionStatusLabel,
  transactionStatusTone,
} from '@/pages/online-payments/transaction-status';
import { HistoryLog } from '@/ui/history-log';
import { HttpClient, httpResource } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  linkedSignal,
  signal,
} from '@angular/core';
import { RouterLink } from '@angular/router';
import { type Observable, firstValueFrom } from 'rxjs';

type Confirming = 'cancel' | 'refund' | null;

// One online payment: what GoPay reports, the order behind it, GoPay's
// webhook deliveries, and what the admin can do — check it at GoPay now,
// cancel it while unpaid, or mark a refund (GoPay's API has none: the money
// goes back from the GoPay dashboard or the bank).
@Component({
  selector: 'app-online-payment-page',
  imports: [HistoryLog, RouterLink],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-6">
      <a routerLink="/online-payments" class="text-sm text-text-muted hover:text-text">← Онлайн-платежи</a>

      @if (transaction.error()) {
        <p class="ui-error">{{ hasStatus(transaction.error(), 404) ? 'Платёж не найден' : 'Не удалось загрузить платёж' }}</p>
      }
      @if (current(); as t) {
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1 class="text-xl font-semibold text-text">{{ t.order.itemTitle }}</h1>
            <p class="text-sm text-text-muted">
              {{ som(gatewayAmount(t.amount)) }} · {{ t.order.quantity }} чел. · {{ gatewayName(t.provider) }}
            </p>
            <p class="mt-2 flex flex-wrap gap-2">
              <span class="ui-badge" [class]="statusTone(t.status)">{{ statusLabel(t.status) }}</span>
              @if (t.testing) {
                <span class="ui-badge bg-violet-100 text-violet-800">тестовый платёж</span>
              }
              <span class="ui-badge bg-gray-100 text-gray-700">заказ: {{ orderLabel(t.order.status) }}</span>
            </p>
          </div>
          <div class="flex flex-wrap gap-2">
            <button type="button" class="ui-btn-soft" [disabled]="busy()" (click)="check()">Проверить в GoPay</button>
            @if (canCancel(t)) {
              <button type="button" class="ui-btn-danger" [disabled]="busy()" (click)="confirming.set('cancel')">Отменить платёж</button>
            }
            @if (canRefund(t)) {
              <button type="button" class="ui-btn-danger" [disabled]="busy()" (click)="confirming.set('refund')">Отметить возврат</button>
            }
          </div>
        </div>

        @if (error()) {
          <p class="ui-error">{{ error() }}</p>
        }
        @if (notice()) {
          <p class="text-sm text-green-700">{{ notice() }}</p>
        }

        @if (confirming() === 'cancel') {
          <div class="ui-card flex flex-col gap-3 p-4">
            <p class="text-sm">
              Отменить неоплаченный платёж? QR перестанет работать, заказ закроется, места освободятся.
            </p>
            <div class="flex gap-2">
              <button type="button" class="ui-btn-danger" [disabled]="busy()" (click)="cancel()">Да, отменить</button>
              <button type="button" class="ui-btn-ghost" (click)="confirming.set(null)">Нет</button>
            </div>
          </div>
        }
        @if (confirming() === 'refund') {
          <div class="ui-card flex flex-col gap-3 p-4">
            <p class="text-sm">
              У GoPay нет возвратов через API: верните {{ som(t.order.amount) }} клиенту из кабинета GoPay или через банк,
              затем отметьте возврат здесь — заказ перестанет считаться к выплате партнёру.
            </p>
            <label class="ui-field">
              <span>Комментарий (например, номер операции)</span>
              <input class="ui-input" maxlength="500" [value]="refundNote()" (input)="refundNote.set($any($event.target).value)" />
            </label>
            <div class="flex gap-2">
              <button type="button" class="ui-btn-danger" [disabled]="busy()" (click)="refund()">Отметить возврат</button>
              <button type="button" class="ui-btn-ghost" (click)="confirming.set(null)">Отмена</button>
            </div>
          </div>
        }

        <div class="grid gap-6 lg:grid-cols-2">
          <section class="ui-card p-4">
            <h2 class="mb-3 font-semibold text-text">GoPay</h2>
            <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
              <dt class="text-text-muted">ID платежа</dt><dd class="break-all">{{ t.paymentId ?? '—' }}</dd>
              <dt class="text-text-muted">order_id</dt><dd class="break-all">{{ t.providerOrderId }}</dd>
              <dt class="text-text-muted">Сумма</dt><dd>{{ t.amount }} сом</dd>
              <dt class="text-text-muted">Создан</dt><dd>{{ when(t.gatewayCreatedAt) }}</dd>
              <dt class="text-text-muted">Действует до</dt><dd>{{ when(t.expiresAt) }}</dd>
              <dt class="text-text-muted">Проведён</dt><dd>{{ when(t.committedAt) }}</dd>
              <dt class="text-text-muted">Опер. день банка</dt><dd>{{ t.bankOpDate ?? '—' }}</dd>
              <dt class="text-text-muted">Проверен</dt><dd>{{ formatDateTime(t.checkedAt) }}</dd>
              @if (t.checkoutUrl) {
                <dt class="text-text-muted">Страница оплаты</dt>
                <dd class="break-all"><a class="underline" [href]="t.checkoutUrl" target="_blank" rel="noopener">{{ t.checkoutUrl }}</a></dd>
              }
              @if (t.lastError) {
                <dt class="text-text-muted">Ошибка запроса</dt><dd class="text-red-700">{{ t.lastError }}</dd>
              }
            </dl>
          </section>

          <section class="ui-card p-4">
            <h2 class="mb-3 font-semibold text-text">Заказ</h2>
            <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
              <dt class="text-text-muted">ID</dt><dd class="break-all">{{ t.order.id }}</dd>
              <dt class="text-text-muted">Дата тура</dt><dd>{{ t.order.itemDate ?? '—' }}</dd>
              <dt class="text-text-muted">Клиент</dt>
              <dd>{{ t.order.contactName }}, {{ t.order.contactPhone }}<p class="text-xs text-text-muted">{{ personName(t.order.payer) }}</p></dd>
              <dt class="text-text-muted">Партнёр</dt><dd>{{ personName(t.order.payee) }}</dd>
              <dt class="text-text-muted">Комиссия</dt><dd>{{ som(t.order.fee) }}</dd>
              <dt class="text-text-muted">Партнёру</dt>
              <dd>{{ som(t.order.payeeAmount) }}@if (t.order.payoutId) { · выплачено }</dd>
              <dt class="text-text-muted">Оплачен</dt><dd>{{ when(t.order.paidAt) }}</dd>
              @if (t.order.refundNote) {
                <dt class="text-text-muted">Возврат</dt><dd>{{ t.order.refundNote }}</dd>
              }
            </dl>
          </section>
        </div>

        <section>
          <h2 class="mb-3 text-lg font-semibold text-text">Уведомления GoPay</h2>
          <div class="ui-table-wrap">
            <table class="ui-table">
              <thead><tr><th>Получено</th><th>Событие</th><th>Попыток</th><th>Обработано</th></tr></thead>
              <tbody>
                @for (e of t.events ?? []; track e.id) {
                  <tr>
                    <td class="text-text-muted">{{ formatDateTime(e.receivedAt) }}</td>
                    <td>{{ e.event }}@if (e.testing) { <span class="ui-badge ml-1 bg-violet-100 text-violet-800">тест</span> }</td>
                    <td class="text-text-muted">{{ e.attempts }}</td>
                    <td>
                      {{ when(e.processedAt) }}
                      @if (e.error) {
                        <p class="text-xs text-red-700">{{ e.error }}</p>
                      }
                    </td>
                  </tr>
                } @empty {
                  <tr><td colspan="4" class="ui-empty">Уведомлений не было</td></tr>
                }
              </tbody>
            </table>
          </div>
        </section>
      }
    </div>

    <app-history-log class="mt-8" [modules]="['payments']" [entityId]="id()" />
  `,
})
export class OnlinePaymentPage {
  private readonly http = inject(HttpClient);
  // The order's id, from the route.
  readonly id = input.required<string>();

  protected readonly gateway = httpResource<PaymentGateway>(() => '/payments/admin/gateway');
  protected readonly transaction = httpResource<Transaction>(
    () => `/payments/admin/transactions/${this.id()}`,
  );
  // The loaded payment, replaced by what an action returns.
  protected readonly current = linkedSignal<Transaction | undefined>(() =>
    this.transaction.value(),
  );

  protected readonly busy = signal(false);
  protected readonly error = signal<string | null>(null);
  protected readonly notice = signal<string | null>(null);
  protected readonly confirming = signal<Confirming>(null);
  protected readonly refundNote = signal('');

  protected readonly som = som;
  protected readonly personName = personName;
  protected readonly formatDateTime = formatDateTime;
  protected readonly gatewayAmount = gatewayAmount;
  protected readonly gatewayName = gatewayName;
  protected readonly orderLabel = orderStatusLabel;
  protected readonly statusLabel = transactionStatusLabel;
  protected readonly statusTone = transactionStatusTone;
  protected readonly hasStatus = hasStatus;

  private readonly cancellable = computed(() => this.gateway.value()?.canCancel ?? false);

  protected when(value: string | null): string {
    return value ? formatDateTime(value) : '—';
  }

  protected canCancel(t: Transaction): boolean {
    return this.cancellable() && t.order.status === 'pending';
  }

  protected canRefund(t: Transaction): boolean {
    return t.order.status === 'paid' && !t.order.payoutId;
  }

  protected async check(): Promise<void> {
    await this.run(
      () => this.http.post<Transaction>(`/payments/admin/transactions/${this.id()}/check`, {}),
      'Статус обновлён',
      'GoPay сейчас недоступен, попробуйте позже',
    );
  }

  protected async cancel(): Promise<void> {
    await this.run(
      () => this.http.post<Transaction>(`/payments/admin/transactions/${this.id()}/cancel`, {}),
      'Платёж отменён',
      'Не удалось отменить платёж: GoPay недоступен, попробуйте позже',
      'Платёж уже нельзя отменить: он оплачен или закрыт',
    );
  }

  protected async refund(): Promise<void> {
    const note = this.refundNote().trim() || null;
    this.busy.set(true);
    this.error.set(null);
    this.notice.set(null);
    try {
      await firstValueFrom(this.http.post(`/orders/${this.id()}/refund`, { note }));
      this.confirming.set(null);
      this.notice.set('Возврат отмечен');
      this.transaction.reload();
    } catch (error) {
      this.error.set(
        hasStatus(error, 409)
          ? 'Возврат возможен только для оплаченных заказов до выплаты партнёру'
          : apiErrorMessage(error, 'Не удалось отметить возврат'),
      );
    } finally {
      this.busy.set(false);
    }
  }

  private async run(
    request: () => Observable<Transaction>,
    done: string,
    fallback: string,
    conflict = fallback,
  ): Promise<void> {
    this.busy.set(true);
    this.error.set(null);
    this.notice.set(null);
    try {
      this.current.set(await firstValueFrom(request()));
      this.confirming.set(null);
      this.notice.set(done);
    } catch (error) {
      this.error.set(hasStatus(error, 409) ? conflict : fallback);
      this.transaction.reload();
    } finally {
      this.busy.set(false);
    }
  }
}
