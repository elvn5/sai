import type { Order, OrderStatus } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, formatDate, formatInBishkek } from '@/core/lib/format';
import { contentLang } from '@/core/lib/partner-cabinet';
import { SiteLink } from '@/ui/site-link';
import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

const STATUS_CLASS: Record<OrderStatus, string> = {
  pending: 'bg-amber-500/15 text-amber-800 dark:text-amber-200',
  paid: 'bg-emerald-500/15 text-emerald-800 dark:text-emerald-200',
  failed: 'bg-red-500/15 text-red-800 dark:text-red-200',
  expired: 'bg-text/10 text-text',
  refunded: 'bg-sky-500/15 text-sky-800 dark:text-sky-200',
};

// A booking: what, when, how many, how much, and its payment status.
// `forPartner` shows the customer and what the partner gets instead.
@Component({
  selector: 'app-order-card',
  imports: [SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <article class="ui-card flex flex-col gap-2 px-4 py-3">
      <div class="flex items-start justify-between gap-3">
        <div class="min-w-0">
          @if (order().itemSlug; as slug) {
            <a [appSiteLink]="'/tours/' + slug" [lang]="tourLang()" class="block truncate font-semibold text-text hover:underline">{{ order().itemTitle }}</a>
          } @else {
            <span class="block truncate font-semibold text-text">{{ order().itemTitle }}</span>
          }
          <p class="text-xs text-text-muted">
            @if (tripDate(); as date) { {{ date }} · }{{ 'booking.placesCount' | transloco: { count: order().quantity } }}
          </p>
        </div>
        <span class="shrink-0 rounded-full px-2.5 py-1 text-xs font-semibold" [class]="statusClass()">{{ 'booking.status.' + order().status | transloco }}</span>
      </div>
      <div class="flex flex-wrap items-center justify-between gap-2 text-sm">
        @if (forPartner()) {
          <span class="text-text">
            {{ order().contactName }} ·
            <a [href]="'tel:' + order().contactPhone" class="text-brand-700 hover:underline dark:text-brand-300">{{ order().contactPhone }}</a>
          </span>
        } @else {
          <span class="text-text-muted">{{ created() }}</span>
        }
        <span class="font-bold text-text">{{ amount() }} {{ 'listing.currency' | transloco }}</span>
      </div>
      @if (forPartner() && order().comment) {
        <p class="text-sm text-text-muted italic">«{{ order().comment }}»</p>
      }
      @if (!forPartner() && order().status === 'pending') {
        <div class="flex flex-wrap items-center gap-3">
          @if (order().paymentUrl; as url) {
            <a [href]="url" class="ui-btn-primary ui-btn-xs">{{ 'booking.continuePayment' | transloco }}</a>
          }
          <a [appSiteLink]="'/payments/return?order=' + order().id" class="text-xs text-text-muted underline">{{ 'booking.checkStatus' | transloco }}</a>
        </div>
      }
    </article>
  `,
})
export class OrderCard {
  readonly order = input.required<Order>();
  readonly forPartner = input(false);
  private readonly lang = injectLang();

  protected readonly tourLang = computed(() => contentLang(this.order().itemLocale));
  protected readonly statusClass = computed(() => STATUS_CLASS[this.order().status]);
  protected readonly tripDate = computed(() => {
    const date = this.order().itemDate;
    return date ? formatDate(this.lang(), date, { day: 'numeric', month: 'long' }) : null;
  });
  protected readonly created = computed(() =>
    formatInBishkek(this.lang(), this.order().createdAt, {
      day: 'numeric',
      month: 'long',
      year: 'numeric',
    }),
  );
  protected readonly amount = computed(() =>
    formatAmount(this.lang(), this.forPartner() ? this.order().payeeAmount : this.order().amount),
  );
}
