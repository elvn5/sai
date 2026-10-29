import type { DriverExpense, DriverVehicle } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, formatDate, todayInBishkek } from '@/core/lib/format';
import { loaded } from '@/core/lib/resource';
import { Notify } from '@/core/notify/notify';
import { bindPageMeta } from '@/core/seo/seo.service';
import { DRIVER_EXPENSE_CATEGORIES } from '@/pages/partner/driver/driver';
import { injectConfirm } from '@/ui/confirm-dialog';
import { ListSkeleton } from '@/ui/list-skeleton';
import { LoadError } from '@/ui/load-error';
import { SelectField, injectOptions } from '@/ui/select-field';
import { SiteLink } from '@/ui/site-link';
import { StateBlock } from '@/ui/state-block';
import { HttpClient, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// The month a date falls in, and the first and last day of a month.
function monthOf(day: string): string {
  return day.slice(0, 7);
}

function monthRange(month: string): { from: string; to: string } {
  const [year, m] = month.split('-').map(Number);
  const last = new Date(Date.UTC(year ?? 2026, m ?? 1, 0)).getUTCDate();
  return { from: `${month}-01`, to: `${month}-${String(last).padStart(2, '0')}` };
}

function shiftMonth(month: string, by: number): string {
  const [year, m] = month.split('-').map(Number);
  const date = new Date(Date.UTC(year ?? 2026, (m ?? 1) - 1 + by, 1));
  return date.toISOString().slice(0, 7);
}

// /partner/costs: what the vehicles cost month by month — fuel, repairs,
// servicing, washing, parking, fines, insurance — by vehicle and trip.
@Component({
  selector: 'app-driver-costs-page',
  imports: [ListSkeleton, LoadError, SelectField, SiteLink, StateBlock, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-5">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 class="text-2xl font-extrabold tracking-tight text-text">{{ 'cabinet.nav.costs' | transloco }}</h1>
          <p class="text-sm text-text-muted">{{ 'driver.costs.subtitle' | transloco }}</p>
        </div>
        <div class="flex items-center gap-1">
          <button type="button" class="ui-icon-btn" [attr.aria-label]="'driver.costs.previous' | transloco" (click)="month.set(shift(month(), -1))"><span class="icon-[heroicons--chevron-left] size-4" aria-hidden="true"></span></button>
          <span class="min-w-32 text-center text-sm font-semibold text-text capitalize">{{ monthLabel() }}</span>
          <button type="button" class="ui-icon-btn" [attr.aria-label]="'driver.costs.next' | transloco" [disabled]="month() >= thisMonth" (click)="month.set(shift(month(), 1))"><span class="icon-[heroicons--chevron-right] size-4" aria-hidden="true"></span></button>
        </div>
      </div>

      <form class="ui-card grid gap-2 p-4 sm:grid-cols-[9rem_7rem_9rem_1fr] sm:items-end" novalidate (submit)="add($event, amount.value, day.value, note.value)">
        <div class="ui-field">
          <span class="ui-label" id="cost-category">{{ 'crm.finance.category' | transloco }}</span>
          <app-select-field [options]="categoryOptions()" labelledBy="cost-category" [value]="category()" (valueChange)="category.set($event)" />
        </div>
        <div class="ui-field">
          <label class="ui-label" for="cost-amount">{{ 'crm.payment.amount' | transloco }}</label>
          <input #amount id="cost-amount" class="ui-input" type="number" inputmode="numeric" />
        </div>
        <div class="ui-field">
          <label class="ui-label" for="cost-day">{{ 'crm.payment.date' | transloco }}</label>
          <input #day id="cost-day" class="ui-input" type="date" [value]="today" [max]="today" />
        </div>
        <div class="ui-field">
          <span class="ui-label" id="cost-vehicle">{{ 'driver.trip.vehicle' | transloco }}</span>
          <app-select-field [options]="vehicleOptions()" labelledBy="cost-vehicle" [value]="vehicleId()" (valueChange)="vehicleId.set($event)" />
        </div>
        <div class="ui-field sm:col-span-3">
          <label class="ui-label" for="cost-note">{{ 'crm.payment.note' | transloco }}</label>
          <input #note id="cost-note" class="ui-input" maxlength="500" [placeholder]="'driver.costs.notePlaceholder' | transloco" />
        </div>
        <button type="submit" class="ui-btn-primary" [disabled]="busy()"><span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>{{ 'crm.finance.add' | transloco }}</button>
      </form>

      @if (expenses.error()) {
        <app-load-error (retry)="expenses.reload()" />
      } @else if (expenses.isLoading() && !expenses.hasValue()) {
        <app-list-skeleton variant="rows" [count]="3" />
      } @else if (list().length === 0) {
        <app-state-block icon="icon-[heroicons--receipt-percent]" title="driver.costs.empty" text="driver.costs.emptyHint" />
      } @else {
        <div class="flex flex-wrap gap-2">
          <span class="ui-badge">{{ 'driver.costs.total' | transloco }}: {{ num(total()) }}</span>
          @for (row of byCategory(); track row.category) {
            <span class="ui-badge ui-badge-soft">{{ 'driver.expense.' + row.category | transloco }}: {{ num(row.amount) }}</span>
          }
        </div>
        <ul class="ui-card flex flex-col divide-y divide-border text-sm">
          @for (e of list(); track e.id) {
            <li class="flex flex-wrap items-center gap-x-3 gap-y-1 px-4 py-2.5">
              <span class="w-16 shrink-0 text-text-muted tabular-nums">{{ date(e.spentOn) }}</span>
              <span class="font-medium text-text">{{ 'driver.expense.' + e.category | transloco }}</span>
              @if (e.vehicle) {
                <span class="text-text-muted">{{ e.vehicle }}</span>
              }
              @if (e.tripId) {
                <a [appSiteLink]="'/partner/trips/' + e.tripId" class="text-brand-700 hover:underline dark:text-brand-300">{{ 'driver.costs.trip' | transloco }}</a>
              }
              @if (e.note) {
                <span class="min-w-0 truncate text-text-muted">— {{ e.note }}</span>
              }
              <span class="ml-auto font-medium text-text tabular-nums">{{ num(e.amount) }}</span>
              <button type="button" class="ui-icon-btn size-7" [attr.aria-label]="'crm.finance.delete' | transloco" [disabled]="busy()" (click)="remove(e)"><span class="icon-[heroicons--x-mark] size-4" aria-hidden="true"></span></button>
            </li>
          }
        </ul>
      }
    </div>
  `,
})
export class DriverCostsPage {
  private readonly http = inject(HttpClient);
  private readonly transloco = inject(TranslocoService);
  private readonly notify = inject(Notify);
  private readonly confirm = injectConfirm();
  private readonly lang = injectLang();
  protected readonly today = todayInBishkek();
  protected readonly thisMonth = monthOf(this.today);
  protected readonly month = signal(this.thisMonth);
  protected readonly category = signal<string>('fuel');
  protected readonly vehicleId = signal('');
  protected readonly busy = signal(false);
  protected readonly shift = shiftMonth;

  protected readonly expenses = httpResource<DriverExpense[]>(() => ({
    url: '/driver/expenses',
    params: monthRange(this.month()),
  }));
  private readonly vehicles = httpResource<DriverVehicle[]>(() => '/driver/vehicles');
  protected readonly list = computed(() => loaded(this.expenses) ?? []);
  protected readonly total = computed(() => this.list().reduce((sum, e) => sum + e.amount, 0));
  protected readonly byCategory = computed(() =>
    DRIVER_EXPENSE_CATEGORIES.map((category) => ({
      category,
      amount: this.list()
        .filter((e) => e.category === category)
        .reduce((sum, e) => sum + e.amount, 0),
    })).filter((row) => row.amount > 0),
  );
  protected readonly categoryOptions = injectOptions(() =>
    DRIVER_EXPENSE_CATEGORIES.map((c) => ({ value: c, key: `driver.expense.${c}` })),
  );
  protected readonly vehicleOptions = injectOptions(() => [
    { value: '', key: 'driver.costs.anyVehicle' },
    ...(loaded(this.vehicles) ?? []).map((v) => ({ value: v.id, label: v.model })),
  ]);
  protected readonly monthLabel = computed(() =>
    formatDate(this.lang(), `${this.month()}-01`, { month: 'long', year: 'numeric' }),
  );

  constructor() {
    bindPageMeta(() => ({
      title: this.transloco.translate('cabinet.nav.costs'),
      description: this.transloco.translate('driver.costs.subtitle'),
      robots: 'noindex',
    }));
  }

  protected async add(
    event: Event,
    rawAmount: string,
    spentOn: string,
    note: string,
  ): Promise<void> {
    event.preventDefault();
    const amount = Math.round(Number(rawAmount));
    if (!(amount > 0) || !spentOn) {
      this.notify.warning({ title: 'crm.errors.amount' });
      return;
    }
    this.busy.set(true);
    try {
      await firstValueFrom(
        this.http.post('/driver/expenses', {
          category: this.category(),
          amount,
          spentOn,
          vehicleId: this.vehicleId() || null,
          note: note.trim() || null,
        }),
      );
      (event.target as HTMLFormElement).reset();
      this.month.set(monthOf(spentOn));
      this.expenses.reload();
    } catch {
      this.notify.error({ title: 'crm.saveError' });
    } finally {
      this.busy.set(false);
    }
  }

  protected async remove(e: DriverExpense): Promise<void> {
    if (!(await this.confirm({ title: 'crm.finance.deleteConfirm', danger: true }))) return;
    this.busy.set(true);
    try {
      await firstValueFrom(this.http.delete(`/driver/expenses/${e.id}`));
      this.expenses.reload();
    } catch {
      this.notify.error({ title: 'crm.saveError' });
    } finally {
      this.busy.set(false);
    }
  }

  protected num(value: number): string {
    return formatAmount(this.lang(), value);
  }

  protected date(day: string): string {
    return formatDate(this.lang(), day, { day: 'numeric', month: 'short' });
  }
}
