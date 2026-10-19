import type { Order } from '@/core/api/types';
import { todayInBishkek } from '@/core/lib/format';
import { downloadCsv } from '@/core/lib/partner-cabinet';
import { loaded } from '@/core/lib/resource';
import { bindPageMeta } from '@/core/seo/seo.service';
import { OrderCard } from '@/ui/order-card';
import { httpResource } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  DOCUMENT,
  computed,
  inject,
  signal,
} from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

type When = 'upcoming' | 'past' | 'all';

// /partner/bookings: paid (and refunded) bookings of the partner's tours,
// by trip date, searchable, with a CSV export.
@Component({
  selector: 'app-partner-bookings-page',
  imports: [OrderCard, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-6">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 class="text-2xl font-extrabold text-text sm:text-3xl">{{ 'cabinet.nav.bookings' | transloco }}</h1>
          <p class="mt-1 text-sm text-text-muted">{{ 'cabinet.bookings.subtitle' | transloco }}</p>
        </div>
        <button type="button" class="ui-btn-soft" [disabled]="list().length === 0" (click)="exportCsv()">
          <span class="icon-[heroicons--arrow-down-tray] size-4" aria-hidden="true"></span>CSV
        </button>
      </div>

      <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
        <select class="ui-input sm:w-48" [attr.aria-label]="'cabinet.bookings.date' | transloco" (change)="when.set($any($event.target).value)">
          @for (option of whenOptions; track option) {
            <option [value]="option" [selected]="when() === option">{{ 'cabinet.bookings.' + option | transloco }}</option>
          }
        </select>
        <div class="relative flex-1">
          <span class="icon-[heroicons--magnifying-glass] pointer-events-none absolute top-1/2 left-3 size-5 -translate-y-1/2 text-text-muted" aria-hidden="true"></span>
          <input type="search" class="ui-input pl-10" [placeholder]="'cabinet.bookings.search' | transloco" [attr.aria-label]="'cabinet.bookings.search' | transloco" (input)="search.set($any($event.target).value)" />
        </div>
        <label class="flex items-center gap-2 text-sm text-text">
          <input type="checkbox" class="size-4 accent-brand-600" [checked]="showRefunded()" (change)="showRefunded.set($any($event.target).checked)" />
          {{ 'cabinet.bookings.showRefunded' | transloco }}
        </label>
      </div>

      @if (list().length > 0) {
        <p class="text-sm text-text-muted" aria-live="polite">{{ 'cabinet.bookings.summary' | transloco: { bookings: list().length, guests: guests() } }}</p>
        <ul class="flex flex-col gap-2">
          @for (order of list(); track order.id) {
            <li><app-order-card [order]="order" [forPartner]="true" /></li>
          }
        </ul>
      } @else if (orders.hasValue()) {
        <div class="flex flex-col items-center gap-2 rounded-card border border-dashed border-border px-6 py-12 text-center">
          <span class="icon-[heroicons--ticket] size-10 text-text-muted" aria-hidden="true"></span>
          <p class="font-medium text-text">{{ 'booking.noReceived' | transloco }}</p>
          <p class="max-w-sm text-sm text-text-muted">{{ 'cabinet.bookings.emptyHint' | transloco }}</p>
        </div>
      }
    </div>
  `,
})
export class PartnerBookingsPage {
  private readonly transloco = inject(TranslocoService);
  private readonly document = inject(DOCUMENT);
  protected readonly whenOptions: When[] = ['upcoming', 'past', 'all'];
  protected readonly when = signal<When>('upcoming');
  protected readonly search = signal('');
  protected readonly showRefunded = signal(false);

  protected readonly orders = httpResource<Order[]>(() => '/orders/received');

  protected readonly list = computed(() => {
    const today = todayInBishkek();
    const needle = this.search().trim().toLowerCase();
    const when = this.when();
    const list = (loaded(this.orders) ?? []).filter((order) => {
      if (!this.showRefunded() && order.status === 'refunded') return false;
      const date = order.itemDate ?? '';
      if (when === 'upcoming' && date < today) return false;
      if (when === 'past' && date >= today) return false;
      return (
        !needle ||
        [order.contactName, order.contactPhone, order.itemTitle].some((field) =>
          field.toLowerCase().includes(needle),
        )
      );
    });
    // Upcoming trips soonest first; the rest newest first (the API's order).
    return when === 'upcoming'
      ? list.sort((a, b) => (a.itemDate ?? '').localeCompare(b.itemDate ?? ''))
      : list;
  });
  protected readonly guests = computed(() =>
    this.list()
      .filter((order) => order.status === 'paid')
      .reduce((sum, order) => sum + order.quantity, 0),
  );

  constructor() {
    bindPageMeta(() => ({
      title: this.transloco.translate('cabinet.nav.bookings'),
      description: this.transloco.translate('cabinet.bookings.subtitle'),
      robots: 'noindex',
    }));
  }

  protected exportCsv(): void {
    const t = (key: string): string => this.transloco.translate(key);
    downloadCsv(this.document, `ermekten-bookings-${todayInBishkek()}.csv`, [
      [
        t('cabinet.bookings.tour'),
        t('cabinet.bookings.date'),
        t('cabinet.bookings.guests'),
        t('cabinet.bookings.name'),
        t('cabinet.bookings.phone'),
        t('cabinet.bookings.comment'),
        t('cabinet.bookings.paid'),
        t('cabinet.bookings.earned'),
        t('cabinet.bookings.status'),
      ],
      ...this.list().map((order) => [
        order.itemTitle,
        order.itemDate,
        order.quantity,
        order.contactName,
        order.contactPhone,
        order.comment,
        order.amount,
        order.payeeAmount,
        t(`booking.status.${order.status}`),
      ]),
    ]);
  }
}
