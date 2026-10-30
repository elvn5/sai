import type { PaymentGateway, Transaction, TransactionStatus } from '@/core/api/types';
import { formatDateTime, personName, som } from '@/core/lib/format';
import {
  TRANSACTION_STATUSES,
  gatewayAmount,
  gatewayName,
  orderStatusLabel,
  transactionStatusLabel,
  transactionStatusTone,
} from '@/pages/online-payments/transaction-status';
import { HistoryLog } from '@/ui/history-log';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

// Online payments through the gateway (GoPay): every payment as GoPay
// reports it, filtered by its status and the day it was created.
@Component({
  selector: 'app-online-payments-page',
  imports: [HistoryLog, RouterLink],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-6">
      <div>
        <h1 class="text-xl font-semibold text-text">Онлайн-платежи</h1>
        <p class="text-sm text-text-muted">
          Платежи через GoPay (ELQR): статус каждого — как его видит GoPay. Заказы, выплаты партнёрам и возвраты — в разделе «Платежи».
        </p>
        @if (gateway.value(); as g) {
          <p class="mt-2 text-sm">
            Шлюз: <span class="font-medium">{{ gatewayName(g.provider) }}</span>
            @if (!g.online) {
              <span class="ui-badge ml-2 bg-gray-100 text-gray-600">онлайн-оплата выключена</span>
            }
          </p>
        }
        @if (transactions.error()) {
          <p class="ui-error mt-2">Не удалось загрузить платежи</p>
        }
      </div>

      <div class="flex flex-wrap items-end gap-3">
        <label class="ui-field">
          <span>Статус</span>
          <select class="ui-input w-44" [value]="status()" (change)="status.set($any($event.target).value)">
            <option value="">Все</option>
            @for (option of statuses; track option.value) {
              <option [value]="option.value">{{ option.label }}</option>
            }
          </select>
        </label>
        <label class="ui-field">
          <span>С</span>
          <input class="ui-input w-40" type="date" [value]="from()" (change)="from.set($any($event.target).value)" />
        </label>
        <label class="ui-field">
          <span>По</span>
          <input class="ui-input w-40" type="date" [value]="to()" (change)="to.set($any($event.target).value)" />
        </label>
        @if (status() || from() || to()) {
          <button type="button" class="ui-btn-ghost" (click)="clear()">Сбросить</button>
        }
      </div>

      @if (rows().length) {
        <p class="text-sm text-text-muted">
          {{ rows().length }} платежей · оплачено {{ som(paidTotal()) }}
        </p>
      }

      <div class="ui-table-wrap">
        <table class="ui-table">
          <thead>
            <tr><th>Создан</th><th>Заказ</th><th>Клиент</th><th>Сумма</th><th>Статус в GoPay</th><th>Заказ у нас</th></tr>
          </thead>
          <tbody>
            @for (t of rows(); track t.orderId) {
              <tr>
                <td class="text-text-muted">{{ formatDateTime(t.createdAt) }}</td>
                <td>
                  <a class="font-medium text-text underline-offset-2 hover:underline" [routerLink]="['/online-payments', t.orderId]">
                    {{ t.order.itemTitle }}
                  </a>
                  <p class="text-xs text-text-muted">{{ t.order.quantity }} чел. · {{ t.paymentId ?? t.providerOrderId }}</p>
                </td>
                <td>{{ t.order.contactName }}<p class="text-xs text-text-muted">{{ personName(t.order.payer) }}</p></td>
                <td class="font-semibold">{{ som(gatewayAmount(t.amount)) }}</td>
                <td>
                  <span class="ui-badge" [class]="statusTone(t.status)">{{ statusLabel(t.status) }}</span>
                  @if (t.testing) {
                    <span class="ui-badge ml-1 bg-violet-100 text-violet-800">тест</span>
                  }
                </td>
                <td class="text-text-muted">{{ orderLabel(t.order.status) }}</td>
              </tr>
            } @empty {
              @if (!transactions.isLoading()) {
                <tr><td colspan="6" class="ui-empty">Платежей нет</td></tr>
              }
            }
          </tbody>
        </table>
      </div>
    </div>

    <app-history-log class="mt-8" [modules]="['payments']" />
  `,
})
export class OnlinePaymentsPage {
  protected readonly statuses = TRANSACTION_STATUSES;
  protected readonly status = signal<TransactionStatus | ''>('');
  protected readonly from = signal('');
  protected readonly to = signal('');

  protected readonly gateway = httpResource<PaymentGateway>(() => '/payments/admin/gateway');
  protected readonly transactions = httpResource<Transaction[]>(() => {
    const query = new URLSearchParams();
    if (this.status()) query.set('status', this.status());
    if (this.from()) query.set('from', this.from());
    if (this.to()) query.set('to', this.to());
    const qs = query.toString();
    return `/payments/admin/transactions${qs ? `?${qs}` : ''}`;
  });
  protected readonly rows = computed(() => this.transactions.value() ?? []);
  protected readonly paidTotal = computed(() =>
    this.rows()
      .filter((t) => t.status === 'COMMITTED')
      .reduce((sum, t) => sum + gatewayAmount(t.amount), 0),
  );

  protected readonly som = som;
  protected readonly personName = personName;
  protected readonly formatDateTime = formatDateTime;
  protected readonly gatewayAmount = gatewayAmount;
  protected readonly statusLabel = transactionStatusLabel;
  protected readonly statusTone = transactionStatusTone;

  protected readonly gatewayName = gatewayName;
  protected readonly orderLabel = orderStatusLabel;

  protected clear(): void {
    this.status.set('');
    this.from.set('');
    this.to.set('');
  }
}
