import type { DriverSummary } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, formatDate, formatInBishkek, todayInBishkek } from '@/core/lib/format';
import { loaded } from '@/core/lib/resource';
import { bindPageMeta } from '@/core/seo/seo.service';
import { daysUntil, tripBadge, tripClientName } from '@/pages/partner/driver/driver';
import { StatTile } from '@/pages/partner/stat-tile';
import { TaskList } from '@/pages/partner/task-list';
import { SiteLink } from '@/ui/site-link';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

// /partner for a driver: requests from organizers waiting for an answer,
// the next trips, money this month and what customers still owe, vehicle
// documents running out, and the first steps (the card, a vehicle).
@Component({
  selector: 'app-driver-dashboard-page',
  imports: [SiteLink, StatTile, TaskList, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-6">
      <div>
        <h1 class="text-2xl font-extrabold tracking-tight text-text">{{ 'cabinet.nav.dashboard' | transloco }}</h1>
        <p class="text-sm text-text-muted">{{ 'driver.dashboard.subtitle' | transloco }}</p>
      </div>

      @if (data(); as s) {
        @if (!s.published || s.vehicles === 0) {
          <div class="rounded-card bg-sky-500/13 px-4 py-4">
            <p class="flex items-center gap-2 font-semibold text-text"><span class="icon-[heroicons--sparkles] size-5 text-info" aria-hidden="true"></span>{{ 'driver.onboarding.title' | transloco }}</p>
            <ul class="mt-3 flex flex-col gap-2">
              @for (step of [{ key: 'driver.onboarding.profile', path: '/partner/driver-profile', done: s.published }, { key: 'driver.onboarding.vehicle', path: '/partner/fleet', done: s.vehicles > 0 }]; track step.key) {
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

        @if (s.requests > 0) {
          <a appSiteLink="/partner/trips?status=requested" class="ui-banner ui-banner-warning hover:underline">
            <span class="icon-[heroicons--bell-alert]" aria-hidden="true"></span>
            <span>{{ 'driver.dashboard.requests' | transloco: { count: s.requests } }}</span>
          </a>
        }

        @for (doc of s.documents; track doc.vehicleId + doc.kind) {
          <a appSiteLink="/partner/fleet" class="ui-banner hover:underline" [class]="left(doc.until) < 0 ? 'ui-banner-danger' : 'ui-banner-warning'">
            <span class="icon-[heroicons--document-text]" aria-hidden="true"></span>
            <span>{{ (left(doc.until) < 0 ? 'driver.documents.expired' : 'driver.documents.soon') | transloco: { vehicle: doc.vehicle, document: ('driver.documents.' + doc.kind | transloco), date: date(doc.until), days: left(doc.until) } }}</span>
          </a>
        }

        <div class="grid grid-cols-3 gap-3">
          <app-stat-tile [label]="'driver.dashboard.income' | transloco" icon="icon-[heroicons--banknotes]" [value]="num(s.monthIncome)" />
          <app-stat-tile [label]="'driver.dashboard.expenses' | transloco" icon="icon-[heroicons--receipt-percent]" [value]="num(s.monthExpenses)" />
          <app-stat-tile [label]="'crm.dashboard.outstanding' | transloco" icon="icon-[heroicons--clock]" [value]="num(s.due)" />
        </div>

        <section class="flex flex-col gap-3" aria-labelledby="upcoming-trips">
          <div class="flex items-center justify-between">
            <h2 id="upcoming-trips" class="font-semibold text-text">{{ 'driver.dashboard.upcoming' | transloco }}</h2>
            <a appSiteLink="/partner/trips" class="text-sm text-brand-700 hover:underline dark:text-brand-300">{{ 'cabinet.nav.trips' | transloco }}</a>
          </div>
          @if (s.upcoming.length > 0) {
            <ul class="ui-card flex flex-col divide-y divide-border">
              @for (trip of s.upcoming; track trip.id) {
                <li>
                  <a [appSiteLink]="'/partner/trips/' + trip.id" class="flex flex-wrap items-center gap-x-4 gap-y-1 px-4 py-3 hover:bg-text/3">
                    <span class="min-w-0 flex-1">
                      <span class="block truncate font-medium text-text">{{ trip.routeFrom }} → {{ trip.routeTo }}</span>
                      <span class="text-xs text-text-muted">{{ when(trip.startsAt) }}@if (clientName(trip); as name) { · {{ name }}}@if (trip.tour) { · {{ trip.tour.title }}}</span>
                    </span>
                    <span [class]="badge(trip.status)">{{ 'driver.status.' + trip.status | transloco }}</span>
                  </a>
                </li>
              }
            </ul>
          } @else {
            <p class="ui-card px-4 py-6 text-center text-sm text-text-muted">{{ 'driver.dashboard.noUpcoming' | transloco }}</p>
          }
        </section>
      }

      <section class="ui-card flex flex-col gap-2 p-4" aria-labelledby="dashboard-tasks">
        <div class="flex items-center justify-between">
          <h2 id="dashboard-tasks" class="font-semibold text-text">{{ 'crm.tasks.dashboard' | transloco }}</h2>
          <a appSiteLink="/partner/tasks" class="text-sm text-brand-700 hover:underline dark:text-brand-300">{{ 'crm.tasks.all' | transloco }}</a>
        </div>
        <app-task-list [limit]="5" />
      </section>
    </div>
  `,
})
export class DriverDashboardPage {
  private readonly transloco = inject(TranslocoService);
  private readonly lang = injectLang();
  private readonly today = todayInBishkek();
  private readonly summary = httpResource<DriverSummary>(() => '/driver/summary');
  protected readonly data = computed(() => loaded(this.summary));
  protected readonly badge = tripBadge;
  protected readonly clientName = tripClientName;

  constructor() {
    bindPageMeta(() => ({
      title: this.transloco.translate('cabinet.title'),
      description: this.transloco.translate('driver.dashboard.subtitle'),
      robots: 'noindex',
    }));
  }

  protected num(value: number): string {
    return formatAmount(this.lang(), value);
  }

  protected when(iso: string): string {
    return formatInBishkek(this.lang(), iso, {
      weekday: 'short',
      day: 'numeric',
      month: 'short',
      hour: '2-digit',
      minute: '2-digit',
    });
  }

  protected date(day: string): string {
    return formatDate(this.lang(), day, { day: 'numeric', month: 'short', year: 'numeric' });
  }

  protected left(day: string): number {
    return daysUntil(this.today, day);
  }
}
