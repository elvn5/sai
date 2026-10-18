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
import { PersonAvatar } from '@/ui/person-avatar';
import { RatingStars } from '@/ui/rating-stars';
import { SiteLink } from '@/ui/site-link';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

const DAYS = 30;
// The periods the summary can cover.
const PERIODS = [7, 30, 90];
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
  imports: [DailyBarChart, PersonAvatar, RatingStars, SiteLink, StatTile, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-6">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 class="text-2xl font-extrabold tracking-tight text-text">{{ 'cabinet.nav.dashboard' | transloco }}</h1>
          <p class="text-sm text-text-muted">{{ 'cabinet.dashboard.subtitle' | transloco: { days: days() } }}</p>
        </div>
        <div class="ui-segment w-60 text-[0.8rem]" role="radiogroup" [attr.aria-label]="'cabinet.dashboard.period' | transloco">
          @for (option of periods; track option) {
            <button type="button" role="radio" [attr.aria-checked]="days() === option" (click)="days.set(option)">{{ 'cabinet.dashboard.periodDays' | transloco: { days: option } }}</button>
          }
        </div>
      </div>

      @if (report() && onboardingLeft() > 0) {
        <div class="rounded-card bg-sky-500/13 px-4 py-4">
          <p class="flex items-center gap-2 font-semibold text-text"><span class="icon-[heroicons--sparkles] size-5 text-info" aria-hidden="true"></span>{{ 'cabinet.onboarding.title' | transloco }}</p>
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
        <app-stat-tile [label]="'cabinet.dashboard.monthBookings' | transloco" icon="icon-[heroicons--ticket]" [value]="money() ? '' + money()!.month.orders : '—'" [hint]="money() ? ('cabinet.dashboard.guests' | transloco: { count: money()!.month.guests }) : null" />
        <app-stat-tile [label]="'cabinet.finances.owed' | transloco" icon="icon-[heroicons--banknotes]" [value]="money() ? som(money()!.owed) : '—'" [hint]="money() ? ('cabinet.finances.owedOrders' | transloco: { count: money()!.owedOrders }) : null" />
      </div>

      <div class="grid grid-cols-1 gap-3.5 lg:grid-cols-[1.4fr_1fr]">
        <div class="ui-card flex flex-col gap-4 p-4">
          <app-daily-bar-chart [title]="'cabinet.stats.viewsPerDay' | transloco" [points]="viewsPoints()" />
          <app-daily-bar-chart [title]="'cabinet.stats.contactsPerDay' | transloco" [points]="contactsPoints()" />
          <p class="text-xs text-text-muted">{{ 'cabinet.stats.howCounted' | transloco }}</p>
        </div>
        <section class="ui-card flex flex-col gap-3 p-4" aria-labelledby="departures">
          <div class="flex items-center justify-between">
            <h2 id="departures" class="font-semibold text-text">{{ 'cabinet.dashboard.departures' | transloco }}</h2>
            <a appSiteLink="/partner/bookings" class="text-sm text-brand-700 hover:underline dark:text-brand-300">{{ 'cabinet.dashboard.allBookings' | transloco }}</a>
          </div>
          @if (departures().length > 0) {
            <ul class="flex flex-col gap-3">
              @for (departure of departures(); track departure.key) {
                <li class="flex items-center gap-3 text-sm">
                  <span class="flex w-11 shrink-0 flex-col items-center rounded-lg border border-border py-0.5 leading-tight">
                    <b class="text-text">{{ dayNumber(departure.date) }}</b>
                    <span class="text-[0.68rem] text-text-muted">{{ monthShort(departure.date) }}</span>
                  </span>
                  <div class="min-w-0">
                    <p class="truncate font-semibold text-text">{{ departure.title }}</p>
                    <p class="text-xs text-text-muted">{{ 'cabinet.dashboard.bookingsCount' | transloco: { count: departure.bookings } }} · {{ 'cabinet.dashboard.guests' | transloco: { count: departure.guests } }}</p>
                  </div>
                </li>
              }
            </ul>
          } @else if (orders.hasValue()) {
            <p class="flex flex-1 flex-col items-center justify-center gap-2 py-4 text-center text-sm text-text-muted">
              <span class="icon-[heroicons--calendar-days] size-7" aria-hidden="true"></span>
              {{ (hasPaid() ? 'cabinet.dashboard.noDepartures' : 'booking.noReceived') | transloco }}
            </p>
          }
        </section>
      </div>

      <div>
        <section class="ui-card flex flex-col gap-3 p-4" aria-labelledby="new-reviews">
          <div class="flex items-center justify-between">
            <h2 id="new-reviews" class="font-semibold text-text">
              {{ 'cabinet.dashboard.newReviews' | transloco }}
              @if (unansweredTotal() > 0) { <span class="text-text-muted">({{ unansweredTotal() }})</span> }
            </h2>
            <a appSiteLink="/partner/reviews" class="text-sm text-brand-700 hover:underline dark:text-brand-300">{{ 'cabinet.dashboard.answer' | transloco }}</a>
          </div>
          @if (newReviews().length > 0) {
            <ul class="flex flex-col divide-y divide-border">
              @for (review of newReviews(); track review.id) {
                <li class="flex items-start gap-3 py-2.5 first:pt-0 last:pb-0">
                  <app-person-avatar [person]="review.author" [size]="34" />
                  <div class="min-w-0 flex-1 text-sm">
                    <p class="flex flex-wrap items-center gap-2"><b class="text-text">{{ review.author.firstName ?? review.author.username }}</b><app-rating-stars [value]="review.rating" /></p>
                    <p class="truncate text-xs text-text-muted">{{ review.listing.title }}</p>
                    <p class="line-clamp-2 text-text-muted">{{ review.text }}</p>
                  </div>
                  <a appSiteLink="/partner/reviews" class="ui-btn-soft shrink-0"><span class="icon-[heroicons--chat-bubble-left-ellipsis] size-4" aria-hidden="true"></span>{{ 'cabinet.dashboard.answer' | transloco }}</a>
                </li>
              }
            </ul>
          } @else if (reviews.hasValue()) {
            <p class="flex items-center justify-center gap-2 py-4 text-sm text-text-muted"><span class="icon-[heroicons--check-circle] size-5 text-success" aria-hidden="true"></span>{{ 'cabinet.reviews.allAnswered' | transloco }}</p>
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
  protected readonly periods = PERIODS;
  protected readonly days = signal<number>(DAYS);

  private readonly stats = httpResource<StatsReport>(() => ({
    url: '/listing-stats/mine',
    params: { days: this.days() },
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

  protected dayNumber(date: string): string {
    return String(Number(date.slice(8, 10)));
  }

  protected monthShort(date: string): string {
    return formatDate(this.currentLang(), date, { month: 'short' }).replace('.', '');
  }

  protected day(date: string): string {
    return formatDate(this.currentLang(), date, { day: 'numeric', month: 'long' });
  }
}
