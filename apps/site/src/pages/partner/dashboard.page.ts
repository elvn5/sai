import type { CrmSummary, Paginated, PartnerReview, StatsReport } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, formatDate } from '@/core/lib/format';
import { contentLang, listingPath, percentChange, shareOf } from '@/core/lib/partner-cabinet';
import { loaded } from '@/core/lib/resource';
import { bindPageMeta } from '@/core/seo/seo.service';
import { DailyBarChart } from '@/pages/partner/daily-bar-chart';
import { StatTile } from '@/pages/partner/stat-tile';
import { TaskList } from '@/pages/partner/task-list';
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

// /partner: clients and money (the CRM), the next tours with their groups,
// the last 30 days at a glance, reviews waiting for an answer and the most
// viewed listings.
@Component({
  selector: 'app-partner-dashboard-page',
  imports: [DailyBarChart, PersonAvatar, RatingStars, SiteLink, StatTile, TaskList, TranslocoPipe],
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

      <section class="ui-card flex flex-col gap-2 p-4" aria-labelledby="dashboard-tasks">
        <div class="flex items-center justify-between">
          <h2 id="dashboard-tasks" class="font-semibold text-text">{{ 'crm.tasks.dashboard' | transloco }}</h2>
          <a appSiteLink="/partner/tasks" class="text-sm text-brand-700 hover:underline dark:text-brand-300">{{ 'crm.tasks.all' | transloco }}</a>
        </div>
        <app-task-list [limit]="5" />
      </section>

      @if (crm(); as summary) {
        <div class="grid grid-cols-3 gap-3">
          <app-stat-tile [label]="'cabinet.nav.clients' | transloco" icon="icon-[heroicons--users]" [value]="number(summary.clients)" />
          <app-stat-tile [label]="'crm.dashboard.paidThisMonth' | transloco" icon="icon-[heroicons--banknotes]" [value]="number(summary.paidThisMonth)" />
          <app-stat-tile [label]="'crm.dashboard.outstanding' | transloco" icon="icon-[heroicons--clock]" [value]="number(summary.outstanding)" />
        </div>
        @if (summary.upcoming.length > 0) {
          <section class="flex flex-col gap-3" aria-labelledby="upcoming-tours">
            <div class="flex items-center justify-between">
              <h2 id="upcoming-tours" class="font-semibold text-text">{{ 'crm.dashboard.upcoming' | transloco }}</h2>
              <a appSiteLink="/partner/clients" class="text-sm text-brand-700 hover:underline dark:text-brand-300">{{ 'cabinet.nav.clients' | transloco }}</a>
            </div>
            <ul class="ui-card flex flex-col divide-y divide-border">
              @for (tour of summary.upcoming; track tour.id) {
                <li>
                  <a [appSiteLink]="'/partner/tours/' + tour.id + '/participants'" class="flex flex-wrap items-center gap-x-4 gap-y-1 px-4 py-3 hover:bg-text/3">
                    <span class="min-w-0 flex-1">
                      <span class="block truncate font-medium text-text">{{ tour.title }}</span>
                      <span class="text-xs text-text-muted">{{ day(tour.startDate) }}</span>
                    </span>
                    <span class="flex items-center gap-1 text-sm text-text-muted tabular-nums"><span class="icon-[heroicons--user-group] size-4" aria-hidden="true"></span>{{ tour.taken }}@if (tour.capacity !== null) { / {{ tour.capacity }}}@if (tour.waitlist > 0) { <span class="ui-badge ui-badge-warning">+{{ tour.waitlist }}</span>}</span>
                    <span class="text-sm tabular-nums" [class]="tour.due > tour.paid ? 'text-amber-700 dark:text-amber-300' : 'text-text-muted'">{{ number(tour.paid) }} / {{ number(tour.due) }}</span>
                  </a>
                </li>
              }
            </ul>
          </section>
        }
      }

      <div class="grid grid-cols-2 gap-3">
        <app-stat-tile [label]="'cabinet.stats.views' | transloco" icon="icon-[heroicons--eye]" [value]="report() ? number(report()!.totals.views) : '—'" [delta]="report() ? change(report()!.totals.views, report()!.previous.views) : null" [hint]="'cabinet.dashboard.vsPrevious' | transloco" />
        <app-stat-tile [label]="'cabinet.stats.contacts' | transloco" icon="icon-[heroicons--chat-bubble-left-ellipsis]" [value]="report() ? number(report()!.totals.contacts) : '—'" [delta]="report() ? change(report()!.totals.contacts, report()!.previous.contacts) : null" [hint]="report() ? ('cabinet.stats.conversionHint' | transloco: { value: share(report()!.totals.contacts, report()!.totals.views) }) : null" />
      </div>

      <div class="ui-card flex flex-col gap-4 p-4">
        <app-daily-bar-chart [title]="'cabinet.stats.viewsPerDay' | transloco" [points]="viewsPoints()" />
        <app-daily-bar-chart [title]="'cabinet.stats.contactsPerDay' | transloco" [points]="contactsPoints()" />
        <p class="text-xs text-text-muted">{{ 'cabinet.stats.howCounted' | transloco }}</p>
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
  protected readonly reviews = httpResource<Paginated<PartnerReview>>(() => ({
    url: '/reviews/mine',
    params: { limit: 3, unanswered: true },
  }));

  private readonly crmSummary = httpResource<CrmSummary>(() => '/crm/summary');
  protected readonly crm = computed(() => loaded(this.crmSummary));
  protected readonly report = computed(() => loaded(this.stats));
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

  protected day(date: string): string {
    return formatDate(this.currentLang(), date, {
      weekday: 'short',
      day: 'numeric',
      month: 'short',
    });
  }
}
