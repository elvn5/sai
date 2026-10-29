import type { DriverExpenseCategory, DriverReport } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, formatDate, todayInBishkek } from '@/core/lib/format';
import { loaded } from '@/core/lib/resource';
import { bindPageMeta } from '@/core/seo/seo.service';
import { StatTile } from '@/pages/partner/stat-tile';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

// The periods offered, in months back (this one included).
const PERIODS = [1, 3, 6, 12];

function monthsBack(today: string, months: number): string {
  const date = new Date(`${today.slice(0, 7)}-01T00:00:00Z`);
  date.setUTCMonth(date.getUTCMonth() - months + 1);
  return date.toISOString().slice(0, 10);
}

// /partner/earnings: trips done and what they brought in, what the vehicles
// cost and what's left — by month, by vehicle and by kind of expense.
@Component({
  selector: 'app-driver-earnings-page',
  imports: [StatTile, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-5">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 class="text-2xl font-extrabold tracking-tight text-text">{{ 'cabinet.nav.earnings' | transloco }}</h1>
          <p class="text-sm text-text-muted">{{ 'driver.earnings.subtitle' | transloco }}</p>
        </div>
        <div class="ui-segment w-72 text-[0.8rem]" role="radiogroup" [attr.aria-label]="'cabinet.dashboard.period' | transloco">
          @for (option of periods; track option) {
            <button type="button" role="radio" [attr.aria-checked]="months() === option" (click)="months.set(option)">{{ 'driver.earnings.months' | transloco: { count: option } }}</button>
          }
        </div>
      </div>

      @if (data(); as r) {
        <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <app-stat-tile [label]="'driver.earnings.trips' | transloco" icon="icon-[heroicons--truck]" [value]="num(r.trips)" [hint]="'driver.earnings.cancelled' | transloco: { count: r.cancelled }" />
          <app-stat-tile [label]="'driver.trip.passengers' | transloco" icon="icon-[heroicons--user-group]" [value]="num(r.passengers)" />
          <app-stat-tile [label]="'driver.earnings.income' | transloco" icon="icon-[heroicons--banknotes]" [value]="num(r.income)" />
          <app-stat-tile [label]="'driver.earnings.profit' | transloco" icon="icon-[heroicons--scale]" [value]="num(r.profit)" [hint]="('crm.finance.expenses' | transloco) + ': ' + num(r.expenses)" />
        </div>

        <section class="flex flex-col gap-2" aria-labelledby="earnings-months">
          <h2 id="earnings-months" class="font-semibold text-text">{{ 'driver.earnings.byMonth' | transloco }}</h2>
          <div class="ui-table-wrap">
            <table class="ui-table">
              <thead><tr><th>{{ 'driver.earnings.month' | transloco }}</th><th class="text-right">{{ 'driver.earnings.trips' | transloco }}</th><th class="text-right">{{ 'driver.earnings.income' | transloco }}</th><th class="text-right">{{ 'crm.finance.expenses' | transloco }}</th><th class="text-right">{{ 'driver.earnings.profit' | transloco }}</th></tr></thead>
              <tbody>
                @for (m of r.byMonth; track m.month) {
                  <tr>
                    <td class="capitalize">{{ month(m.month) }}</td>
                    <td class="text-right tabular-nums">{{ m.trips }}</td>
                    <td class="text-right tabular-nums">{{ num(m.income) }}</td>
                    <td class="text-right tabular-nums">{{ num(m.expenses) }}</td>
                    <td class="text-right tabular-nums" [class.text-danger]="m.income < m.expenses">{{ num(m.income - m.expenses) }}</td>
                  </tr>
                }
              </tbody>
            </table>
          </div>
        </section>

        <div class="grid gap-4 lg:grid-cols-2">
          <section class="flex flex-col gap-2" aria-labelledby="earnings-vehicles">
            <h2 id="earnings-vehicles" class="font-semibold text-text">{{ 'driver.earnings.byVehicle' | transloco }}</h2>
            @if (r.byVehicle.length > 0) {
              <div class="ui-table-wrap">
                <table class="ui-table">
                  <thead><tr><th>{{ 'driver.trip.vehicle' | transloco }}</th><th class="text-right">{{ 'driver.earnings.trips' | transloco }}</th><th class="text-right">{{ 'driver.earnings.income' | transloco }}</th><th class="text-right">{{ 'crm.finance.expenses' | transloco }}</th></tr></thead>
                  <tbody>
                    @for (v of r.byVehicle; track v.id) {
                      <tr>
                        <td>{{ v.model ?? ('driver.earnings.noVehicle' | transloco) }}</td>
                        <td class="text-right tabular-nums">{{ v.trips }}</td>
                        <td class="text-right tabular-nums">{{ num(v.income) }}</td>
                        <td class="text-right tabular-nums">{{ num(v.expenses) }}</td>
                      </tr>
                    }
                  </tbody>
                </table>
              </div>
            } @else {
              <p class="text-sm text-text-muted">{{ 'driver.earnings.empty' | transloco }}</p>
            }
          </section>
          <section class="flex flex-col gap-2" aria-labelledby="earnings-costs">
            <h2 id="earnings-costs" class="font-semibold text-text">{{ 'driver.earnings.byCategory' | transloco }}</h2>
            @if (categories().length > 0) {
              <ul class="ui-card flex flex-col gap-2 p-4 text-sm">
                @for (row of categories(); track row.category) {
                  <li class="flex flex-col gap-1">
                    <span class="flex justify-between"><span class="text-text">{{ 'driver.expense.' + row.category | transloco }}</span><span class="font-medium text-text tabular-nums">{{ num(row.amount) }}</span></span>
                    <span class="h-1.5 overflow-hidden rounded-full bg-text/8"><span class="block h-full rounded-full bg-brand-500" [style.width.%]="(row.amount / r.expenses) * 100"></span></span>
                  </li>
                }
              </ul>
            } @else {
              <p class="text-sm text-text-muted">{{ 'driver.costs.empty' | transloco }}</p>
            }
          </section>
        </div>
        <p class="text-xs text-text-muted">{{ 'driver.earnings.howCounted' | transloco }}</p>
      }
    </div>
  `,
})
export class DriverEarningsPage {
  private readonly transloco = inject(TranslocoService);
  private readonly lang = injectLang();
  private readonly today = todayInBishkek();
  protected readonly periods = PERIODS;
  protected readonly months = signal(6);

  private readonly report = httpResource<DriverReport>(() => ({
    url: '/driver/reports',
    params: { from: monthsBack(this.today, this.months()), to: this.today },
  }));
  protected readonly data = computed(() => loaded(this.report));
  protected readonly categories = computed(() =>
    Object.entries(this.data()?.byCategory ?? {})
      .map(([category, amount]) => ({
        category: category as DriverExpenseCategory,
        amount: amount ?? 0,
      }))
      .sort((a, b) => b.amount - a.amount),
  );

  constructor() {
    bindPageMeta(() => ({
      title: this.transloco.translate('cabinet.nav.earnings'),
      description: this.transloco.translate('driver.earnings.subtitle'),
      robots: 'noindex',
    }));
  }

  protected num(value: number): string {
    return formatAmount(this.lang(), value);
  }

  protected month(value: string): string {
    return formatDate(this.lang(), `${value}-01`, { month: 'long', year: 'numeric' });
  }
}
