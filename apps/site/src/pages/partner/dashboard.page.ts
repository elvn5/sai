import type {
  Order,
  Paginated,
  PartnerFinances,
  PartnerReview,
  StatsReport,
} from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, formatDate, todayInBishkek } from '@/core/lib/format';
import { contentLang, listingPath, percentChange, shareOf } from '@/core/lib/partner-cabinet';
import { loaded } from '@/core/lib/resource';
import { bindPageMeta } from '@/core/seo/seo.service';
import { DailyBarChart } from '@/pages/partner/daily-bar-chart';
import { StatTile } from '@/pages/partner/stat-tile';
import { RatingStars } from '@/ui/rating-stars';
import { SiteLink } from '@/ui/site-link';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

const DAYS = 30;
const TOP = 5;

interface Departure {
  key: string;
  title: string;
  date: string;
  guests: number;
  bookings: number;
}

// /partner: the last 30 days at a glance, upcoming departures, reviews
// waiting for an answer and the most viewed listings.
@Component({
  selector: 'app-partner-dashboard-page',
  imports: [DailyBarChart, RatingStars, SiteLink, StatTile, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-6">
      <div>
        <h1 class="text-2xl font-extrabold text-text sm:text-3xl">{{ 'cabinet.title' | transloco }}</h1>
        <p class="mt-1 text-sm text-text-muted">{{ 'cabinet.dashboard.subtitle' | transloco: { days: days } }}</p>
      </div>

      @if (report() && onboardingLeft() > 0) {
        <div class="rounded-card border border-brand-500/30 bg-brand-500/5 px-4 py-4">
          <p class="font-semibold text-text">{{ 'cabinet.onboarding.title' | transloco }}</p>
          <ul class="mt-3 flex flex-col gap-2">
            @for (step of onboarding(); track step.key) {
              <li class="flex items-center gap-2 text-sm">
                <span [class]="(step.done ? 'icon-[heroicons--check-circle-solid] text-emerald-700 dark:text-emerald-300' : 'icon-[heroicons--minus-circle] text-text-muted') + ' size-5 shrink-0'" aria-hidden="true"></span>
                @if (step.done) {
                  <span class="text-text-muted line-through">{{ step.key | transloco }}</span>
                } @else {
                  <a [appSiteLink]="step.path" class="font-medium text-text hover:underline">{{ step.key | transloco }}</a>
                }
              </li>
            }
          </ul>
        </div>
      }

      <div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <app-stat-tile [label]="'cabinet.stats.views' | transloco" icon="icon-[heroicons--eye]" [value]="report() ? number(report()!.totals.views) : '—'" [delta]="report() ? change(report()!.totals.views, report()!.previous.views) : null" [hint]="'cabinet.dashboard.vsPrevious' | transloco" />
        <app-stat-tile [label]="'cabinet.stats.contacts' | transloco" icon="icon-[heroicons--chat-bubble-left-ellipsis]" [value]="report() ? number(report()!.totals.contacts) : '—'" [delta]="report() ? change(report()!.totals.contacts, report()!.previous.contacts) : null" [hint]="report() ? ('cabinet.stats.conversionHint' | transloco: { value: share(report()!.totals.contacts, report()!.totals.views) }) : null" />
        <app-stat-tile [label]="'cabinet.dashboard.monthBookings' | transloco" icon="icon-[mdi--ticket-confirmation-outline]" [value]="money() ? '' + money()!.month.orders : '—'" [hint]="money() ? ('cabinet.dashboard.guests' | transloco: { count: money()!.month.guests }) : null" />
        <app-stat-tile [label]="'cabinet.finances.owed' | transloco" icon="icon-[heroicons--banknotes]" [value]="money() ? som(money()!.owed) : '—'" [hint]="money() ? ('cabinet.finances.owedOrders' | transloco: { count: money()!.owedOrders }) : null" />
      </div>

      <div class="ui-card grid grid-cols-1 gap-6 p-4 sm:grid-cols-2">
        <app-daily-bar-chart [title]="'cabinet.stats.viewsPerDay' | transloco" [points]="viewsPoints()" />
        <app-daily-bar-chart [title]="'cabinet.stats.contactsPerDay' | transloco" [points]="contactsPoints()" />
        <p class="text-xs text-text-muted sm:col-span-2">{{ 'cabinet.stats.howCounted' | transloco }}</p>
      </div>

      <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <section class="flex flex-col gap-3" aria-labelledby="departures">
          <div class="flex items-center justify-between">
            <h2 id="departures" class="font-semibold text-text">{{ 'cabinet.dashboard.departures' | transloco }}</h2>
            <a appSiteLink="/partner/bookings" class="text-sm text-brand-700 hover:underline dark:text-brand-300">{{ 'cabinet.dashboard.allBookings' | transloco }}</a>
          </div>
          @if (departures().length > 0) {
            <ul class="flex flex-col gap-2">
              @for (departure of departures(); track departure.key) {
                <li class="ui-card flex items-center justify-between gap-3 px-4 py-3">
                  <div class="min-w-0">
                    <p class="truncate font-medium text-text">{{ departure.title }}</p>
                    <p class="text-xs text-text-muted">{{ day(departure.date) }}</p>
                  </div>
                  <div class="shrink-0 text-right">
                    <p class="font-bold text-text tabular-nums">{{ 'cabinet.dashboard.guests' | transloco: { count: departure.guests } }}</p>
                    <p class="text-xs text-text-muted">{{ 'cabinet.dashboard.bookingsCount' | transloco: { count: departure.bookings } }}</p>
                  </div>
                </li>
              }
            </ul>
          } @else if (orders.hasValue()) {
            <p class="rounded-card border border-dashed border-border px-4 py-6 text-center text-sm text-text-muted">
              {{ (hasPaid() ? 'cabinet.dashboard.noDepartures' : 'booking.noReceived') | transloco }}
            </p>
          }
        </section>

        <section class="flex flex-col gap-3" aria-labelledby="new-reviews">
          <div class="flex items-center justify-between">
            <h2 id="new-reviews" class="font-semibold text-text">
              {{ 'cabinet.dashboard.newReviews' | transloco }}
              @if (unansweredTotal() > 0) { <span class="text-text-muted">({{ unansweredTotal() }})</span> }
            </h2>
            <a appSiteLink="/partner/reviews" class="text-sm text-brand-700 hover:underline dark:text-brand-300">{{ 'cabinet.dashboard.answer' | transloco }}</a>
          </div>
          @if (newReviews().length > 0) {
            <ul class="flex flex-col gap-2">
              @for (review of newReviews(); track review.id) {
                <li>
                  <a appSiteLink="/partner/reviews" class="ui-card flex flex-col gap-1 px-4 py-3 hover:border-brand-500/40">
                    <span class="flex items-center justify-between gap-2">
                      <span class="truncate text-sm font-medium text-text">{{ review.listing.title }}</span>
                      <app-rating-stars [value]="review.rating" />
                    </span>
                    <span class="line-clamp-2 text-sm text-text-muted">{{ review.text }}</span>
                  </a>
                </li>
              }
            </ul>
          } @else if (reviews.hasValue()) {
            <p class="rounded-card border border-dashed border-border px-4 py-6 text-center text-sm text-text-muted">{{ 'cabinet.reviews.allAnswered' | transloco }}</p>
          }
        </section>
      </div>

      @if (topListings().length > 0) {
        <section class="flex flex-col gap-3" aria-labelledby="top-listings">
          <div class="flex items-center justify-between">
            <h2 id="top-listings" class="font-semibold text-text">{{ 'cabinet.dashboard.topListings' | transloco }}</h2>
            <a appSiteLink="/partner/stats" class="text-sm text-brand-700 hover:underline dark:text-brand-300">{{ 'cabinet.dashboard.allStats' | transloco }}</a>
          </div>
          <ul class="ui-card flex flex-col divide-y divide-border">
            @for (item of topListings(); track item.listing.id) {
              <li class="flex items-center justify-between gap-3 px-4 py-3">
                <a [appSiteLink]="path(item.listing)" [lang]="lang(item.listing.locale)" class="min-w-0 truncate font-medium text-text hover:underline">{{ item.listing.title }}</a>
                <span class="flex shrink-0 gap-4 text-sm text-text-muted tabular-nums">
                  <span class="flex items-center gap-1"><span class="icon-[heroicons--eye] size-4" aria-hidden="true"></span><span class="sr-only">{{ 'cabinet.stats.views' | transloco }}:</span>{{ item.views }}</span>
                  <span class="flex items-center gap-1"><span class="icon-[heroicons--chat-bubble-left-ellipsis] size-4" aria-hidden="true"></span><span class="sr-only">{{ 'cabinet.stats.contacts' | transloco }}:</span>{{ item.contacts }}</span>
                </span>
              </li>
            }
          </ul>
        </section>
      }
    </div>
  `,
})
export class PartnerDashboardPage {
  private readonly auth = inject(AuthStore);
  private readonly transloco = inject(TranslocoService);
  private readonly currentLang = injectLang();
  protected readonly days = DAYS;

  private readonly stats = httpResource<StatsReport>(() => ({
    url: '/listing-stats/mine',
    params: { days: DAYS },
  }));
  private readonly finances = httpResource<PartnerFinances>(() => '/payouts/mine');
  protected readonly orders = httpResource<Order[]>(() => '/orders/received');
  protected readonly reviews = httpResource<Paginated<PartnerReview>>(() => ({
    url: '/reviews/mine',
    params: { limit: 3, unanswered: true },
  }));

  protected readonly report = computed(() => loaded(this.stats));
  protected readonly money = computed(() => loaded(this.finances));
  protected readonly newReviews = computed(() => loaded(this.reviews)?.items ?? []);
  protected readonly unansweredTotal = computed(() => loaded(this.reviews)?.total ?? 0);
  protected readonly viewsPoints = computed(
    () => this.report()?.daily.map((d) => ({ day: d.day, value: d.views })) ?? [],
  );
  protected readonly contactsPoints = computed(
    () => this.report()?.daily.map((d) => ({ day: d.day, value: d.contacts })) ?? [],
  );
  protected readonly topListings = computed(() =>
    [...(this.report()?.listings ?? [])]
      .filter((item) => item.views > 0 || item.contacts > 0)
      .sort((a, b) => b.views - a.views)
      .slice(0, TOP),
  );
  protected readonly hasPaid = computed(() =>
    (loaded(this.orders) ?? []).some((order) => order.status === 'paid'),
  );
  // Paid bookings of trips still ahead, by tour and date.
  protected readonly departures = computed((): Departure[] => {
    const today = todayInBishkek();
    const groups = new Map<string, Departure>();
    for (const order of loaded(this.orders) ?? []) {
      if (order.status !== 'paid' || !order.itemDate || order.itemDate < today) continue;
      const key = `${order.itemId}|${order.itemDate}`;
      const group = groups.get(key) ?? {
        key,
        title: order.itemTitle,
        date: order.itemDate,
        guests: 0,
        bookings: 0,
      };
      group.guests += order.quantity;
      group.bookings += 1;
      groups.set(key, group);
    }
    return [...groups.values()].sort((a, b) => a.date.localeCompare(b.date)).slice(0, TOP);
  });
  protected readonly onboarding = computed(() => [
    {
      key: 'cabinet.onboarding.listing',
      path: '/partner/listings',
      done: (this.report()?.listings.length ?? 0) > 0,
    },
    {
      key: 'cabinet.onboarding.telegram',
      path: '/profile',
      done: Boolean(this.auth.user()?.telegramId),
    },
  ]);
  protected readonly onboardingLeft = computed(
    () => this.onboarding().filter((step) => !step.done).length,
  );

  protected readonly change = percentChange;
  protected readonly share = shareOf;
  protected readonly path = listingPath;
  protected readonly lang = contentLang;

  constructor() {
    bindPageMeta(() => ({
      title: this.transloco.translate('cabinet.title'),
      description: this.transloco.translate('cabinet.stats.subtitle'),
      robots: 'noindex',
    }));
  }

  protected number(value: number): string {
    return formatAmount(this.currentLang(), value);
  }

  protected som(value: number): string {
    return `${formatAmount(this.currentLang(), value)} ${this.transloco.translate('listing.currency')}`;
  }

  protected day(date: string): string {
    return formatDate(this.currentLang(), date, { day: 'numeric', month: 'long' });
  }
}
