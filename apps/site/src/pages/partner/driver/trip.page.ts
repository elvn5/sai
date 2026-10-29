import type { DriverTrip, DriverVehicle, PaymentMethod, TripStatus } from '@/core/api/types';
import { openChat } from '@/core/chat/chats';
import { injectLang } from '@/core/i18n/current-lang';
import {
  formatAmount,
  formatDate,
  formatDateRange,
  formatInBishkek,
  todayInBishkek,
} from '@/core/lib/format';
import { loaded } from '@/core/lib/resource';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { Notify } from '@/core/notify/notify';
import { bindPageMeta } from '@/core/seo/seo.service';
import {
  DRIVER_EXPENSE_CATEGORIES,
  tripBadge,
  tripClientName,
  vehicleLabel,
} from '@/pages/partner/driver/driver';
import { TripForm } from '@/pages/partner/driver/trip-form';
import { injectConfirm } from '@/ui/confirm-dialog';
import { ListSkeleton } from '@/ui/list-skeleton';
import { LoadError } from '@/ui/load-error';
import { SelectField, injectOptions } from '@/ui/select-field';
import { SiteLink } from '@/ui/site-link';
import { HttpClient, httpResource } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  linkedSignal,
  signal,
} from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { type Observable, firstValueFrom } from 'rxjs';

const METHODS: PaymentMethod[] = ['cash', 'transfer', 'card', 'other'];

// /partner/trips/:id: one trip — an organizer's request answered here
// (with the vehicle), the group's meeting point and size, the customer and
// a chat with them, what they paid, and the fuel and other costs of the
// trip.
@Component({
  selector: 'app-driver-trip-page',
  imports: [ListSkeleton, LoadError, SelectField, SiteLink, TranslocoPipe, TripForm],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-5">
      <a appSiteLink="/partner/trips" class="flex items-center gap-1 text-sm text-text-muted hover:text-text"><span class="icon-[heroicons--arrow-left] size-4" aria-hidden="true"></span>{{ 'cabinet.nav.trips' | transloco }}</a>

      @if (trip.error()) {
        <app-load-error (retry)="trip.reload()" />
      } @else if (data(); as t) {
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="min-w-0">
            <h1 class="text-2xl font-extrabold tracking-tight text-text">{{ t.routeFrom }} → {{ t.routeTo }}</h1>
            <p class="flex flex-wrap items-center gap-2 text-sm text-text-muted">
              <span [class]="badge(t.status)">{{ 'driver.status.' + t.status | transloco }}</span>
              {{ when(t.startsAt) }}@if (t.endsAt) { – {{ when(t.endsAt) }}}
            </p>
          </div>
          @if (!editing()) {
            <div class="flex gap-2">
              <button type="button" class="ui-btn-soft" (click)="editing.set(true)"><span class="icon-[heroicons--pencil-square] size-4" aria-hidden="true"></span>{{ 'listing.edit' | transloco }}</button>
              @if (!t.tour) {
                <button type="button" class="ui-btn-ghost text-danger" (click)="remove()"><span class="icon-[heroicons--trash] size-4" aria-hidden="true"></span><span class="sr-only">{{ 'driver.trip.delete' | transloco }}</span></button>
              }
            </div>
          }
        </div>

        @if (editing()) {
          <section class="ui-card p-4" aria-labelledby="edit-trip">
            <h2 id="edit-trip" class="mb-3 font-semibold text-text">{{ 'driver.trip.edit' | transloco }}</h2>
            <app-trip-form [trip]="t" (saved)="update($event)" (cancelled)="editing.set(false)" />
          </section>
        }

        <section class="ui-card flex flex-col gap-3 p-4" aria-labelledby="trip-status">
          <h2 id="trip-status" class="font-semibold text-text">{{ 'driver.trip.status' | transloco }}</h2>
          @if (t.status === 'requested') {
            <p class="text-sm text-text-muted">{{ 'driver.trip.requestedHint' | transloco }}</p>
          }
          @if (t.declineReason && (t.status === 'declined' || t.status === 'cancelled')) {
            <p class="text-sm text-text-muted">{{ 'driver.trip.reason' | transloco }}: {{ t.declineReason }}</p>
          }
          @if (t.status === 'requested' || t.status === 'accepted') {
            <div class="grid gap-2 sm:grid-cols-[1fr_auto] sm:items-end">
              <div class="ui-field">
                <span class="ui-label" id="trip-vehicle">{{ 'driver.trip.vehicle' | transloco }}</span>
                <app-select-field [options]="vehicleOptions()" labelledBy="trip-vehicle" [value]="vehicleId()" (valueChange)="vehicleId.set($event)" />
              </div>
              @if (t.status === 'requested') {
                <button type="button" class="ui-btn-primary" [disabled]="busy()" (click)="setStatus('accepted')"><span class="icon-[heroicons--check] size-4" aria-hidden="true"></span>{{ 'driver.trip.accept' | transloco }}</button>
              } @else {
                <button type="button" class="ui-btn-primary" [disabled]="busy()" (click)="setStatus('done')"><span class="icon-[heroicons--flag] size-4" aria-hidden="true"></span>{{ 'driver.trip.markDone' | transloco }}</button>
              }
            </div>
            <div class="grid gap-2 sm:grid-cols-[1fr_auto] sm:items-end">
              <div class="ui-field">
                <label class="ui-label" for="trip-reason">{{ 'driver.trip.reason' | transloco }}</label>
                <input #reason id="trip-reason" class="ui-input" maxlength="300" [placeholder]="'driver.trip.reasonPlaceholder' | transloco" />
              </div>
              <button type="button" class="ui-btn-soft text-danger" [disabled]="busy()" (click)="setStatus(t.status === 'requested' ? 'declined' : 'cancelled', reason.value)">
                <span class="icon-[heroicons--x-mark] size-4" aria-hidden="true"></span>{{ (t.status === 'requested' ? 'driver.trip.decline' : 'driver.trip.cancel') | transloco }}
              </button>
            </div>
          } @else if (t.status === 'done' || t.status === 'cancelled') {
            <div><button type="button" class="ui-btn-soft" [disabled]="busy()" (click)="setStatus('accepted')"><span class="icon-[heroicons--arrow-uturn-left] size-4" aria-hidden="true"></span>{{ 'driver.trip.reopen' | transloco }}</button></div>
          }
        </section>

        <div class="grid gap-4 lg:grid-cols-2">
          @if (t.tour; as tour) {
            <section class="ui-card flex flex-col gap-2 p-4 text-sm" aria-labelledby="trip-tour">
              <h2 id="trip-tour" class="font-semibold text-text">{{ 'driver.trip.tour' | transloco }}</h2>
              <p class="font-medium text-text">{{ tour.title }}</p>
              <dl class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1">
                <dt class="text-text-muted">{{ 'driver.trip.dates' | transloco }}</dt><dd class="text-text">{{ range(tour.startDate, tour.endDate) }}</dd>
                @if (tour.meetingPoint) {
                  <dt class="text-text-muted">{{ 'driver.trip.meetingPoint' | transloco }}</dt><dd class="text-text">{{ tour.meetingPoint }}@if (tour.meetingTime) {, {{ tour.meetingTime }}}</dd>
                }
                <dt class="text-text-muted">{{ 'driver.trip.group' | transloco }}</dt><dd class="text-text">{{ 'driver.trip.people' | transloco: { count: tour.participants } }}</dd>
              </dl>
            </section>
          }
          <section class="ui-card flex flex-col gap-2 p-4 text-sm" aria-labelledby="trip-client">
            <h2 id="trip-client" class="font-semibold text-text">{{ 'driver.trip.client' | transloco }}</h2>
            @if (t.client) {
              <p class="font-medium text-text"><a [appSiteLink]="'/partner/clients/' + t.client.id" class="hover:underline">{{ clientName(t) ?? '—' }}</a></p>
              @if (t.client.phone) {
                <a class="text-text-muted hover:underline" [href]="'tel:' + t.client.phone">{{ t.client.phone }}</a>
              }
              @if (t.client.userId) {
                <div><button type="button" class="ui-btn-soft" (click)="write(t)"><span class="icon-[heroicons--chat-bubble-left-right] size-4" aria-hidden="true"></span>{{ 'chat.write' | transloco }}</button></div>
              }
            } @else {
              <p class="text-text-muted">{{ 'driver.trip.noClient' | transloco }}</p>
            }
            <dl class="mt-1 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1">
              @if (t.passengers) {
                <dt class="text-text-muted">{{ 'driver.trip.passengers' | transloco }}</dt><dd class="text-text tabular-nums">{{ t.passengers }}</dd>
              }
              @if (t.note) {
                <dt class="text-text-muted">{{ 'crm.client.note' | transloco }}</dt><dd class="whitespace-pre-line text-text">{{ t.note }}</dd>
              }
            </dl>
          </section>
        </div>

        <div class="grid gap-4 lg:grid-cols-2">
          <section class="ui-card flex flex-col gap-3 p-4" aria-labelledby="trip-payments">
            <h2 id="trip-payments" class="font-semibold text-text">{{ 'crm.payments' | transloco }}</h2>
            <dl class="grid grid-cols-3 gap-2 text-sm">
              <div><dt class="text-xs text-text-muted">{{ 'driver.trip.price' | transloco }}</dt><dd class="font-bold text-text tabular-nums">{{ t.price === null ? '—' : num(t.price) }}</dd></div>
              <div><dt class="text-xs text-text-muted">{{ 'crm.paid' | transloco }}</dt><dd class="font-bold text-text tabular-nums">{{ num(t.paid) }}</dd></div>
              <div><dt class="text-xs text-text-muted">{{ 'crm.debt' | transloco }}</dt><dd class="font-bold tabular-nums" [class]="t.due > 0 ? 'text-amber-700 dark:text-amber-300' : 'text-text'">{{ num(t.due) }}</dd></div>
            </dl>
            @if (t.payments?.length) {
              <ul class="flex flex-col divide-y divide-border text-sm">
                @for (p of t.payments; track p.id) {
                  <li class="flex items-center gap-2 py-1.5">
                    <span class="text-text">{{ day(p.paidOn) }}</span>
                    <span class="text-text-muted">{{ 'crm.method.' + p.method | transloco }}</span>
                    @if (p.note) {
                      <span class="truncate text-text-muted">— {{ p.note }}</span>
                    }
                    <span class="ml-auto font-medium text-text tabular-nums">{{ num(p.amount) }}</span>
                    <button type="button" class="ui-icon-btn size-7" [attr.aria-label]="'crm.finance.delete' | transloco" [disabled]="busy()" (click)="deletePayment(p.id)"><span class="icon-[heroicons--x-mark] size-4" aria-hidden="true"></span></button>
                  </li>
                }
              </ul>
            }
            <form class="grid gap-2 sm:grid-cols-[7rem_8rem_1fr_auto] sm:items-end" novalidate (submit)="addPayment($event, payAmount.value, method(), payDay.value)">
              <div class="ui-field">
                <label class="ui-label" for="pay-amount">{{ 'crm.payment.amount' | transloco }}</label>
                <input #payAmount id="pay-amount" class="ui-input" type="number" inputmode="numeric" [value]="t.due || ''" />
              </div>
              <div class="ui-field">
                <span class="ui-label" id="pay-method">{{ 'crm.payment.method' | transloco }}</span>
                <app-select-field [options]="methodOptions()" labelledBy="pay-method" [value]="method()" (valueChange)="method.set($event)" />
              </div>
              <div class="ui-field">
                <label class="ui-label" for="pay-day">{{ 'crm.payment.date' | transloco }}</label>
                <input #payDay id="pay-day" class="ui-input" type="date" [value]="today" />
              </div>
              <button type="submit" class="ui-btn-soft" [disabled]="busy()">{{ 'crm.payment.add' | transloco }}</button>
            </form>
          </section>

          <section class="ui-card flex flex-col gap-3 p-4" aria-labelledby="trip-expenses">
            <h2 id="trip-expenses" class="font-semibold text-text">{{ 'driver.trip.expenses' | transloco }}</h2>
            @if (t.expenseList?.length) {
              <ul class="flex flex-col divide-y divide-border text-sm">
                @for (e of t.expenseList; track e.id) {
                  <li class="flex items-center gap-2 py-1.5">
                    <span class="text-text">{{ 'driver.expense.' + e.category | transloco }}</span>
                    <span class="text-text-muted">{{ day(e.spentOn) }}</span>
                    <span class="ml-auto font-medium text-text tabular-nums">{{ num(e.amount) }}</span>
                    <button type="button" class="ui-icon-btn size-7" [attr.aria-label]="'crm.finance.delete' | transloco" [disabled]="busy()" (click)="deleteExpense(e.id)"><span class="icon-[heroicons--x-mark] size-4" aria-hidden="true"></span></button>
                  </li>
                }
              </ul>
              <p class="text-sm text-text-muted">{{ 'driver.trip.margin' | transloco }}: <b class="text-text tabular-nums">{{ num(t.paid - t.expenses) }}</b></p>
            }
            <form class="grid gap-2 sm:grid-cols-[9rem_7rem_auto] sm:items-end" novalidate (submit)="addExpense($event, category(), costAmount.value)">
              <div class="ui-field">
                <span class="ui-label" id="cost-category">{{ 'crm.finance.category' | transloco }}</span>
                <app-select-field [options]="categoryOptions()" labelledBy="cost-category" [value]="category()" (valueChange)="category.set($event)" />
              </div>
              <div class="ui-field">
                <label class="ui-label" for="cost-amount">{{ 'crm.payment.amount' | transloco }}</label>
                <input #costAmount id="cost-amount" class="ui-input" type="number" inputmode="numeric" />
              </div>
              <button type="submit" class="ui-btn-soft" [disabled]="busy()">{{ 'crm.finance.add' | transloco }}</button>
            </form>
          </section>
        </div>
      } @else {
        <app-list-skeleton variant="rows" [count]="3" />
      }
    </div>
  `,
})
export class DriverTripPage {
  readonly id = input.required<string>();

  private readonly http = inject(HttpClient);
  private readonly transloco = inject(TranslocoService);
  private readonly notify = inject(Notify);
  private readonly confirm = injectConfirm();
  private readonly navigate = injectSiteNavigate();
  private readonly lang = injectLang();
  protected readonly today = todayInBishkek();

  protected readonly trip = httpResource<DriverTrip>(() => `/driver/trips/${this.id()}`);
  private readonly fresh = signal<DriverTrip | null>(null);
  // The latest answer: a change's, else the loaded one.
  protected readonly data = computed(() => {
    const fresh = this.fresh();
    return fresh && fresh.id === this.id() ? fresh : loaded(this.trip);
  });
  private readonly vehicles = httpResource<DriverVehicle[]>(() => '/driver/vehicles');
  protected readonly vehicleOptions = injectOptions(() => [
    { value: '', key: 'driver.trip.noVehicle' },
    ...(loaded(this.vehicles) ?? [])
      .filter((v) => v.active)
      .map((v) => ({ value: v.id, label: vehicleLabel(v) })),
  ]);
  protected readonly vehicleId = linkedSignal(() => {
    const t = this.data();
    if (t?.vehicle) return t.vehicle.id;
    // A request: the first vehicle big enough for the group.
    const need = t?.passengers ?? t?.tour?.participants ?? 1;
    return (loaded(this.vehicles) ?? []).find((v) => v.active && v.seats >= need)?.id ?? '';
  });
  protected readonly methodOptions = injectOptions(() =>
    METHODS.map((m) => ({ value: m, key: `crm.method.${m}` })),
  );
  protected readonly categoryOptions = injectOptions(() =>
    DRIVER_EXPENSE_CATEGORIES.map((c) => ({ value: c, key: `driver.expense.${c}` })),
  );
  protected readonly method = signal<string>('cash');
  protected readonly category = signal<string>('fuel');
  protected readonly editing = signal(false);
  protected readonly busy = signal(false);
  protected readonly badge = tripBadge;
  protected readonly clientName = tripClientName;

  constructor() {
    bindPageMeta(() => ({
      title: this.transloco.translate('cabinet.nav.trips'),
      description: this.transloco.translate('driver.trips.subtitle'),
      robots: 'noindex',
    }));
  }

  protected update(trip: DriverTrip): void {
    this.fresh.set(trip);
    this.editing.set(false);
  }

  protected async setStatus(status: TripStatus, reason = ''): Promise<void> {
    const body: Record<string, unknown> = { status };
    if (status === 'accepted') body['vehicleId'] = this.vehicleId() || null;
    if (status === 'declined' || status === 'cancelled') {
      if (!(await this.confirm({ title: `driver.trip.${status}Confirm`, danger: true }))) return;
      body['declineReason'] = reason.trim() || null;
    }
    await this.run(this.http.put<DriverTrip>(`/driver/trips/${this.id()}/status`, body));
  }

  protected addPayment(event: Event, rawAmount: string, method: string, paidOn: string): void {
    event.preventDefault();
    const amount = Math.round(Number(rawAmount));
    if (!(amount > 0) || !paidOn) {
      this.notify.warning({ title: 'crm.errors.amount' });
      return;
    }
    void this.run(
      this.http.post<DriverTrip>(`/driver/trips/${this.id()}/payments`, { amount, method, paidOn }),
      false,
      event,
    );
  }

  protected addExpense(event: Event, category: string, rawAmount: string): void {
    event.preventDefault();
    const amount = Math.round(Number(rawAmount));
    if (!(amount > 0)) {
      this.notify.warning({ title: 'crm.errors.amount' });
      return;
    }
    const t = this.data();
    void this.run(
      this.http.post('/driver/expenses', {
        category,
        amount,
        spentOn: t ? t.startsAt.slice(0, 10) : this.today,
        tripId: this.id(),
        vehicleId: t?.vehicle?.id ?? null,
      }),
      true,
      event,
    );
  }

  protected deletePayment(id: string): void {
    void this.run(this.http.delete<DriverTrip>(`/driver/payments/${id}`));
  }

  protected async deleteExpense(id: string): Promise<void> {
    if (!(await this.confirm({ title: 'crm.finance.deleteConfirm', danger: true }))) return;
    await this.run(this.http.delete(`/driver/expenses/${id}`), true);
  }

  // Sends a change; an answer that isn't the trip reloads it.
  protected async run(request: Observable<unknown>, reload = false, event?: Event): Promise<void> {
    this.busy.set(true);
    try {
      const answer = await firstValueFrom(request);
      if (reload) {
        this.fresh.set(null);
        this.trip.reload();
      } else {
        this.fresh.set(answer as DriverTrip);
      }
      (event?.target as HTMLFormElement | undefined)?.reset();
    } catch {
      this.notify.error({ title: 'crm.saveError' });
    } finally {
      this.busy.set(false);
    }
  }

  protected async remove(): Promise<void> {
    if (!(await this.confirm({ title: 'driver.trip.deleteConfirm', danger: true }))) return;
    try {
      await firstValueFrom(this.http.delete(`/driver/trips/${this.id()}`));
      await this.navigate('/partner/trips');
    } catch {
      this.notify.error({ title: 'crm.saveError' });
    }
  }

  protected async write(t: DriverTrip): Promise<void> {
    const userId = t.client?.userId;
    if (!userId) return;
    try {
      const chat = t.chatId ? { id: t.chatId } : await openChat(this.http, { userId });
      await this.navigate(`/partner/messages/${chat.id}`);
    } catch {
      this.notify.error({ title: 'chat.openError' });
    }
  }

  protected num(value: number): string {
    return formatAmount(this.lang(), value);
  }

  protected day(date: string): string {
    return formatDate(this.lang(), date, { day: 'numeric', month: 'short' });
  }

  protected range(start: string, end: string): string {
    return formatDateRange(this.lang(), start, end);
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
}
