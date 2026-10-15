import type { Order, PartnerFinances } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, formatInBishkek } from '@/core/lib/format';
import { loaded } from '@/core/lib/resource';
import { bindPageMeta } from '@/core/seo/seo.service';
import { StatTile } from '@/pages/partner/stat-tile';
import { OrderCard } from '@/ui/order-card';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

// /partner/finances: what Ermekten owes the partner for paid bookings,
// this month's earnings and the payouts made.
@Component({
  selector: 'app-partner-finances-page',
  imports: [OrderCard, StatTile, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-6">
      <div>
        <h1 class="text-2xl font-extrabold text-text sm:text-3xl">{{ 'cabinet.nav.finances' | transloco }}</h1>
        <p class="mt-1 text-sm text-text-muted">{{ 'cabinet.finances.subtitle' | transloco }}</p>
      </div>

      @if (summary(); as f) {
        <div class="grid grid-cols-2 gap-3 lg:grid-cols-3">
          <app-stat-tile [label]="'cabinet.finances.owed' | transloco" icon="icon-[heroicons--banknotes]" [value]="som(f.owed)" [hint]="'cabinet.finances.owedOrders' | transloco: { count: f.owedOrders }" />
          <app-stat-tile [label]="'cabinet.finances.paidOut' | transloco" icon="icon-[heroicons--check-badge]" [value]="som(f.paidOut)" [hint]="'cabinet.finances.payoutsCount' | transloco: { count: f.payouts.length }" />
          <app-stat-tile class="col-span-2 lg:col-span-1" [label]="'cabinet.finances.monthEarned' | transloco" icon="icon-[heroicons--arrow-trending-up]" [value]="som(f.month.earned)" [hint]="'cabinet.dashboard.guests' | transloco: { count: f.month.guests }" />
        </div>

        <section class="flex flex-col gap-3" aria-labelledby="this-month">
          <h2 id="this-month" class="font-semibold text-text">{{ 'cabinet.finances.monthHeading' | transloco }}</h2>
          <dl class="ui-card flex flex-col divide-y divide-border text-sm">
            <div class="flex justify-between gap-3 px-4 py-3">
              <dt class="text-text-muted">{{ 'cabinet.finances.revenue' | transloco }}</dt>
              <dd class="text-text tabular-nums">{{ som(f.month.revenue) }}</dd>
            </div>
            <div class="flex justify-between gap-3 px-4 py-3">
              <dt class="text-text-muted">{{ 'cabinet.finances.fee' | transloco }}</dt>
              <dd class="text-text tabular-nums">−{{ som(f.month.fee) }}</dd>
            </div>
            <div class="flex justify-between gap-3 px-4 py-3 font-semibold">
              <dt class="text-text">{{ 'cabinet.finances.earned' | transloco }}</dt>
              <dd class="text-text tabular-nums">{{ som(f.month.earned) }}</dd>
            </div>
          </dl>
          <p class="text-xs text-text-muted">{{ 'cabinet.finances.howPaid' | transloco }}</p>
        </section>
      }

      <section class="flex flex-col gap-3" aria-labelledby="awaiting">
        <h2 id="awaiting" class="font-semibold text-text">{{ 'cabinet.finances.awaiting' | transloco }}</h2>
        @if (owed().length > 0) {
          <ul class="flex flex-col gap-2">
            @for (order of owed(); track order.id) {
              <li><app-order-card [order]="order" [forPartner]="true" /></li>
            }
          </ul>
        } @else if (orders.hasValue()) {
          <p class="rounded-card border border-dashed border-border px-4 py-6 text-center text-sm text-text-muted">{{ 'cabinet.finances.nothingOwed' | transloco }}</p>
        }
      </section>

      <section class="flex flex-col gap-3" aria-labelledby="payouts">
        <h2 id="payouts" class="font-semibold text-text">{{ 'cabinet.finances.history' | transloco }}</h2>
        @if (summary()?.payouts?.length) {
          <ul class="ui-card flex flex-col divide-y divide-border">
            @for (payout of summary()!.payouts; track payout.id) {
              <li class="flex items-center justify-between gap-3 px-4 py-3">
                <div class="min-w-0">
                  <p class="font-medium text-text">{{ date(payout.createdAt) }}</p>
                  <p class="truncate text-xs text-text-muted">
                    {{ 'cabinet.dashboard.bookingsCount' | transloco: { count: payout.orders } }}@if (payout.note) { · {{ payout.note }} }
                  </p>
                </div>
                <span class="shrink-0 font-bold text-text tabular-nums">{{ som(payout.amount) }}</span>
              </li>
            }
          </ul>
        } @else if (finances.hasValue()) {
          <p class="rounded-card border border-dashed border-border px-4 py-6 text-center text-sm text-text-muted">{{ 'cabinet.finances.noPayouts' | transloco }}</p>
        }
      </section>
    </div>
  `,
})
export class PartnerFinancesPage {
  private readonly transloco = inject(TranslocoService);
  private readonly lang = injectLang();

  protected readonly finances = httpResource<PartnerFinances>(() => '/payouts/mine');
  protected readonly orders = httpResource<Order[]>(() => '/orders/received');
  protected readonly summary = computed(() => loaded(this.finances));
  // Paid bookings Ermekten hasn't paid out yet.
  protected readonly owed = computed(() =>
    (loaded(this.orders) ?? []).filter((order) => order.status === 'paid' && !order.payoutId),
  );

  constructor() {
    bindPageMeta(() => ({
      title: this.transloco.translate('cabinet.nav.finances'),
      description: this.transloco.translate('cabinet.finances.subtitle'),
      robots: 'noindex',
    }));
  }

  protected som(amount: number): string {
    return `${formatAmount(this.lang(), amount)} ${this.transloco.translate('listing.currency')}`;
  }

  protected date(iso: string): string {
    return formatInBishkek(this.lang(), iso, { day: 'numeric', month: 'long', year: 'numeric' });
  }
}
