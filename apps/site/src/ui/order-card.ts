import type { Order, OrderStatus } from '@/core/api/types';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, formatDate, formatInBishkek } from '@/core/lib/format';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { contentLang } from '@/core/lib/partner-cabinet';
import { SiteLink } from '@/ui/site-link';
import { NgOptimizedImage } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// Each status: its badge (color, icon and words together).
const STATUS: Record<OrderStatus, { badge: string; icon: string | null }> = {
  pending: { badge: 'ui-badge-warning', icon: 'icon-[heroicons--clock]' },
  paid: { badge: 'ui-badge-success', icon: 'icon-[heroicons--check-circle]' },
  failed: { badge: 'ui-badge-danger', icon: 'icon-[heroicons--x-circle]' },
  expired: { badge: 'ui-badge-muted', icon: null },
  refunded: { badge: 'ui-badge-info', icon: 'icon-[heroicons--arrow-uturn-left]' },
};

// A booking: its status, what, when, how many, how much, and the next step
// (pay, open the tour, book again), with the tour's cover. Waiting for
// payment, it stands out; expired or failed, it fades (the cover in grey). `forPartner` shows the customer and what the
// partner gets instead.
@Component({
  selector: 'app-order-card',
  imports: [NgOptimizedImage, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <article
      class="ui-card flex flex-col gap-3 p-3.5 sm:flex-row sm:items-center sm:gap-4"
      [class.!border-amber-500/50]="!forPartner() && order().status === 'pending'"
      [class.opacity-75]="!forPartner() && closed()"
    >
      @if (!forPartner()) {
        <div class="relative aspect-video w-full shrink-0 overflow-hidden rounded-[0.625rem] bg-slate-200 sm:aspect-[4/3] sm:w-28 dark:bg-slate-800" [class.grayscale]="closed()">
          @if (cover(); as url) {
            <img [ngSrc]="url" fill sizes="(min-width: 640px) 10vw, 100vw" alt="" class="object-cover" />
          } @else {
            <span class="icon-[heroicons--map] absolute inset-0 m-auto size-8 text-text-muted" aria-hidden="true"></span>
          }
        </div>
      }
      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-1.5">
          <span class="ui-badge" [class]="status().badge">
            @if (status().icon; as icon) {
              <span [class]="icon" aria-hidden="true"></span>
            }
            {{ 'booking.status.' + order().status | transloco }}
          </span>
          @if (!forPartner() && order().status === 'pending' && holdUntil(); as time) {
            <span class="text-xs text-text-muted">{{ 'booking.holdUntil' | transloco: { time: time } }}</span>
          }
        </div>
        @if (order().itemSlug; as slug) {
          <a [appSiteLink]="'/tours/' + slug" [lang]="tourLang()" class="mt-1.5 block truncate text-base font-semibold text-text hover:underline">{{ order().itemTitle }}</a>
        } @else {
          <span class="mt-1.5 block truncate text-base font-semibold text-text">{{ order().itemTitle }}</span>
        }
        <p class="text-[0.8125rem] text-text-muted">
          @if (tripDate(); as date) { {{ date }} · }{{ 'booking.placesCount' | transloco: { count: order().quantity } }} ·
          <span class="tabular-nums">{{ amount() }} {{ 'listing.currency' | transloco }}</span>
        </p>
        @if (forPartner()) {
          <p class="mt-1 text-sm text-text">
            {{ order().contactName }} ·
            <a [href]="'tel:' + order().contactPhone" class="text-brand-700 hover:underline dark:text-brand-300">{{ order().contactPhone }}</a>
          </p>
          @if (order().comment) {
            <p class="text-sm text-text-muted italic">«{{ order().comment }}»</p>
          }
        }
      </div>
      @if (!forPartner()) {
        <div class="flex shrink-0 flex-wrap items-center gap-2">
          @switch (order().status) {
            @case ('pending') {
              @if (order().paymentUrl; as url) {
                <a [href]="url" class="ui-btn-primary">
                  <span class="icon-[heroicons--credit-card] size-4" aria-hidden="true"></span>{{ 'booking.continuePayment' | transloco }}
                </a>
              }
              <a [appSiteLink]="'/payments/return?order=' + order().id" class="ui-btn-ghost">{{ 'booking.checkStatus' | transloco }}</a>
            }
            @case ('paid') {
              @if (order().itemSlug; as slug) {
                <a [appSiteLink]="'/tours/' + slug" [lang]="tourLang()" class="ui-btn-soft">
                  <span class="icon-[heroicons--map] size-4" aria-hidden="true"></span>{{ 'booking.openTour' | transloco }}
                </a>
              }
            }
            @default {
              @if (order().itemSlug; as slug) {
                <a [appSiteLink]="'/tours/' + slug" [lang]="tourLang()" class="ui-btn-ghost">{{ 'booking.bookAgain' | transloco }}</a>
              }
            }
          }
        </div>
      } @else {
        <span class="text-xs text-text-muted sm:self-start">{{ created() }}</span>
      }
    </article>
  `,
})
export class OrderCard {
  readonly order = input.required<Order>();
  readonly forPartner = input(false);
  private readonly lang = injectLang();
  private readonly apiBaseUrl = inject(APP_CONFIG).apiBaseUrl;

  protected readonly cover = computed(() => {
    const url = this.order().itemImageUrl;
    return url ? resolveMediaUrl(this.apiBaseUrl, url) : null;
  });

  protected readonly tourLang = computed(() => contentLang(this.order().itemLocale));
  protected readonly status = computed(() => STATUS[this.order().status]);
  protected readonly closed = computed(() =>
    ['failed', 'expired', 'refunded'].includes(this.order().status),
  );
  protected readonly tripDate = computed(() => {
    const date = this.order().itemDate;
    return date ? formatDate(this.lang(), date, { day: 'numeric', month: 'long' }) : null;
  });
  // Until when the places are held for payment ("14:35").
  protected readonly holdUntil = computed(() =>
    formatInBishkek(this.lang(), this.order().expiresAt, { hour: '2-digit', minute: '2-digit' }),
  );
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
