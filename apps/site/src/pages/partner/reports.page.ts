import type { CrmReport } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, formatDate, todayInBishkek } from '@/core/lib/format';
import { loaded } from '@/core/lib/resource';
import { bindPageMeta } from '@/core/seo/seo.service';
import { StatTile } from '@/pages/partner/stat-tile';
import { SiteLink } from '@/ui/site-link';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

// The periods offered, in days back from today.
const PERIODS = [30, 90, 365];

function daysAgo(today: string, days: number): string {
  const date = new Date(`${today}T00:00:00Z`);
  date.setUTCDate(date.getUTCDate() - days);
  return date.toISOString().slice(0, 10);
}

function percent(part: number, whole: number): string {
  return whole > 0 ? `${Math.round((part / whole) * 100)}%` : '—';
}

// /partner/reports: departures that started in a period — fill rate,
// no-shows, cancellations, repeat clients, revenue, expenses and margin,
// each tour's numbers, and where leads came from and how many were won.
@Component({
  selector: 'app-partner-reports-page',
  imports: [SiteLink, StatTile, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-5">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 class="text-2xl font-extrabold tracking-tight text-text">{{ 'cabinet.nav.reports' | transloco }}</h1>
          <p class="text-sm text-text-muted">{{ 'crm.reports.subtitle' | transloco }}</p>
        </div>
        <div class="ui-segment w-72 text-[0.8rem]" role="radiogroup" [attr.aria-label]="'cabinet.dashboard.period' | transloco">
          @for (option of periods; track option) {
            <button type="button" role="radio" [attr.aria-checked]="days() === option" (click)="days.set(option)">{{ 'cabinet.dashboard.periodDays' | transloco: { days: option } }}</button>
          }
        </div>
      </div>

      @if (data(); as r) {
        <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <app-stat-tile [label]="'crm.reports.tours' | transloco" icon="icon-[heroicons--map]" [value]="num(r.tours)" />
          <app-stat-tile [label]="'crm.reports.fill' | transloco" icon="icon-[heroicons--user-group]" [value]="pct(r.taken, r.places)" [hint]="'crm.reports.fillHint' | transloco: { taken: r.taken, places: r.places }" />
          <app-stat-tile [label]="'crm.reports.noShows' | transloco" icon="icon-[heroicons--user-minus]" [value]="pct(r.noShows, r.arrived + r.noShows)" [hint]="'crm.reports.count' | transloco: { count: r.noShows }" />
          <app-stat-tile [label]="'crm.reports.repeat' | transloco" icon="icon-[heroicons--arrow-path]" [value]="pct(r.repeatClients, r.clients)" [hint]="'crm.reports.repeatHint' | transloco: { repeat: r.repeatClients, clients: r.clients }" />
          <app-stat-tile [label]="'crm.reports.revenue' | transloco" icon="icon-[heroicons--banknotes]" [value]="num(r.revenue)" />
          <app-stat-tile [label]="'crm.finance.expenses' | transloco" icon="icon-[heroicons--receipt-percent]" [value]="num(r.expenses)" />
          <app-stat-tile [label]="'crm.finance.margin' | transloco" icon="icon-[heroicons--scale]" [value]="num(r.revenue - r.expenses)" [hint]="pct(r.revenue - r.expenses, r.revenue)" />
          <app-stat-tile [label]="'crm.reports.cancelled' | transloco" icon="icon-[heroicons--x-circle]" [value]="num(r.cancelled)" [hint]="'crm.reports.participants' | transloco: { count: r.participants }" />
        </div>

        <section class="flex flex-col gap-2" aria-labelledby="report-leads">
          <h2 id="report-leads" class="font-semibold text-text">{{ 'crm.reports.leads' | transloco }}</h2>
          @if (r.leads.length > 0) {
            <div class="ui-table-wrap">
              <table class="ui-table">
                <thead><tr><th>{{ 'crm.client.source' | transloco }}</th><th class="text-right">{{ 'crm.reports.leadsTotal' | transloco }}</th><th class="text-right">{{ 'crm.leads.stages.won' | transloco }}</th><th class="text-right">{{ 'crm.leads.stages.lost' | transloco }}</th><th class="text-right">{{ 'crm.reports.conversion' | transloco }}</th></tr></thead>
                <tbody>
                  @for (row of r.leads; track row.source) {
                    <tr>
                      <td>{{ 'crm.source.' + (row.source ?? 'unknown') | transloco }}</td>
                      <td class="text-right tabular-nums">{{ row.total }}</td>
                      <td class="text-right tabular-nums">{{ row.won }}</td>
                      <td class="text-right tabular-nums">{{ row.lost }}</td>
                      <td class="text-right tabular-nums">{{ pct(row.won, row.total) }}</td>
                    </tr>
                  }
                </tbody>
              </table>
            </div>
          } @else {
            <p class="text-sm text-text-muted">{{ 'crm.reports.noLeads' | transloco }}</p>
          }
        </section>

        <section class="flex flex-col gap-2" aria-labelledby="report-tours">
          <h2 id="report-tours" class="font-semibold text-text">{{ 'crm.reports.byTour' | transloco }}</h2>
          @if (r.byTour.length > 0) {
            <div class="ui-table-wrap">
              <table class="ui-table">
                <thead><tr><th>{{ 'crm.reports.tour' | transloco }}</th><th class="text-right">{{ 'crm.group.places' | transloco }}</th><th class="text-right">{{ 'crm.group.arrived' | transloco }}</th><th class="text-right">{{ 'crm.reports.revenue' | transloco }}</th><th class="text-right">{{ 'crm.finance.expenses' | transloco }}</th><th class="text-right">{{ 'crm.finance.margin' | transloco }}</th></tr></thead>
                <tbody>
                  @for (t of r.byTour; track t.id) {
                    <tr>
                      <td><a [appSiteLink]="'/partner/tours/' + t.id + '/participants'" class="font-medium text-text hover:underline">{{ t.title }}</a><span class="block text-xs text-text-muted">{{ date(t.startDate) }}</span></td>
                      <td class="text-right tabular-nums">{{ t.taken }}@if (t.capacity !== null) { / {{ t.capacity }}}</td>
                      <td class="text-right tabular-nums">{{ t.arrived }}</td>
                      <td class="text-right tabular-nums">{{ num(t.revenue) }}</td>
                      <td class="text-right tabular-nums">{{ num(t.expenses) }}</td>
                      <td class="text-right tabular-nums" [class.text-danger]="t.revenue < t.expenses">{{ num(t.revenue - t.expenses) }}</td>
                    </tr>
                  }
                </tbody>
              </table>
            </div>
          } @else {
            <p class="text-sm text-text-muted">{{ 'crm.reports.noTours' | transloco }}</p>
          }
        </section>
        <p class="text-xs text-text-muted">{{ 'crm.reports.howCounted' | transloco }}</p>
      }
    </div>
  `,
})
export class PartnerReportsPage {
  private readonly transloco = inject(TranslocoService);
  private readonly lang = injectLang();
  protected readonly periods = PERIODS;
  protected readonly days = signal(90);
  private readonly today = todayInBishkek();

  private readonly report = httpResource<CrmReport>(() => ({
    url: '/crm/reports',
    params: { from: daysAgo(this.today, this.days()), to: this.today },
  }));
  protected readonly data = computed(() => loaded(this.report));

  constructor() {
    bindPageMeta(() => ({
      title: this.transloco.translate('cabinet.nav.reports'),
      description: this.transloco.translate('crm.reports.subtitle'),
      robots: 'noindex',
    }));
  }

  protected num(value: number): string {
    return formatAmount(this.lang(), value);
  }

  protected pct(part: number, whole: number): string {
    return percent(part, whole);
  }

  protected date(day: string): string {
    return formatDate(this.lang(), day, { day: 'numeric', month: 'short', year: 'numeric' });
  }
}
