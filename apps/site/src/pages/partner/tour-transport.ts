import type { CatalogDriver, CrmGroup, TransportRequest } from '@/core/api/types';
import { openChat } from '@/core/chat/chats';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, formatInBishkek } from '@/core/lib/format';
import { loaded } from '@/core/lib/resource';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { Notify } from '@/core/notify/notify';
import { fromLocalInput, tripBadge, vehicleLabel } from '@/pages/partner/driver/driver';
import { injectConfirm } from '@/ui/confirm-dialog';
import { SelectField } from '@/ui/select-field';
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
import { firstValueFrom } from 'rxjs';

// Transport for a tour, on its group page: the trips asked of drivers and
// their answers, and a request to a driver from the catalog (those with a
// vehicle big enough for the group; busy ones that day are marked).
@Component({
  selector: 'app-tour-transport',
  imports: [SelectField, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <section class="ui-card flex flex-col gap-3 p-4" aria-labelledby="tour-transport">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <h2 id="tour-transport" class="flex items-center gap-2 font-semibold text-text"><span class="icon-[heroicons--truck] size-5 text-text-muted" aria-hidden="true"></span>{{ 'transport.title' | transloco }}</h2>
        @if (!asking()) {
          <button type="button" class="ui-btn-soft" (click)="asking.set(true)"><span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>{{ 'transport.ask' | transloco }}</button>
        }
      </div>

      @if (requests().length > 0) {
        <ul class="flex flex-col divide-y divide-border text-sm">
          @for (r of requests(); track r.id) {
            <li class="flex flex-wrap items-center gap-x-3 gap-y-1 py-2">
              <span class="min-w-0 flex-1">
                <span class="block font-medium text-text">{{ r.driver.name }}@if (r.vehicle) { <span class="font-normal text-text-muted">· {{ vehicle(r) }}</span>}</span>
                <span class="block text-xs text-text-muted">{{ when(r.startsAt) }} · {{ r.routeFrom }} → {{ r.routeTo }}@if (r.price !== null) { · {{ num(r.price) }}}</span>
                @if (r.declineReason && (r.status === 'declined' || r.status === 'cancelled')) {
                  <span class="block text-xs text-text-muted">{{ 'driver.trip.reason' | transloco }}: {{ r.declineReason }}</span>
                }
              </span>
              <span [class]="badge(r.status)">{{ 'transport.status.' + r.status | transloco }}</span>
              @if (r.driver.phone) {
                <a class="ui-icon-btn size-8" [href]="'tel:' + r.driver.phone" [attr.aria-label]="'crm.client.phone' | transloco"><span class="icon-[heroicons--phone] size-4" aria-hidden="true"></span></a>
              }
              <button type="button" class="ui-icon-btn size-8" [attr.aria-label]="'chat.write' | transloco" (click)="write(r.driver.userId)"><span class="icon-[heroicons--chat-bubble-left-right] size-4" aria-hidden="true"></span></button>
              @if (r.status === 'requested' || r.status === 'accepted') {
                <button type="button" class="ui-icon-btn size-8" [attr.aria-label]="'transport.callOff' | transloco" [disabled]="busy()" (click)="callOff(r)"><span class="icon-[heroicons--x-mark] size-4" aria-hidden="true"></span></button>
              }
            </li>
          }
        </ul>
      } @else if (!asking()) {
        <p class="text-sm text-text-muted">{{ 'transport.empty' | transloco }}</p>
      }

      @if (asking()) {
        <form class="flex flex-col gap-3 border-t border-border pt-3" novalidate (submit)="ask($event, from.value, to.value, starts.value, passengers.value, price.value, note.value)">
          <div class="ui-field">
            <span class="ui-label" id="transport-driver">{{ 'transport.driver' | transloco }}</span>
            @if (driverOptions().length > 0) {
              <app-select-field [options]="driverOptions()" labelledBy="transport-driver" [placeholder]="'transport.pickDriver' | transloco" [value]="driverId()" (valueChange)="driverId.set($event)" />
              @if (chosen(); as d) {
                <p class="ui-hint">@if (d.region) { {{ d.region }} · }{{ 'transport.trips' | transloco: { count: d.trips } }}@if (d.about) { · {{ d.about }}}</p>
              }
            } @else if (catalog.hasValue()) {
              <p class="text-sm text-text-muted">{{ 'transport.noDrivers' | transloco }}</p>
            }
          </div>
          <div class="grid gap-3 sm:grid-cols-2">
            <div class="ui-field">
              <label class="ui-label" for="transport-from">{{ 'driver.trip.from' | transloco }}</label>
              <input #from id="transport-from" class="ui-input" maxlength="200" [value]="group().tour.meetingPoint ?? ''" />
            </div>
            <div class="ui-field">
              <label class="ui-label" for="transport-to">{{ 'driver.trip.to' | transloco }}</label>
              <input #to id="transport-to" class="ui-input" maxlength="200" [placeholder]="'driver.trip.toPlaceholder' | transloco" />
            </div>
            <div class="ui-field">
              <label class="ui-label" for="transport-starts">{{ 'transport.pickup' | transloco }}</label>
              <input #starts id="transport-starts" class="ui-input" type="datetime-local" [value]="pickup()" />
            </div>
            <div class="grid grid-cols-2 gap-3">
              <div class="ui-field">
                <label class="ui-label" for="transport-passengers">{{ 'driver.trip.passengers' | transloco }}</label>
                <input #passengers id="transport-passengers" class="ui-input" type="number" inputmode="numeric" [value]="group().totals.active || ''" />
              </div>
              <div class="ui-field">
                <label class="ui-label" for="transport-price">{{ 'driver.trip.price' | transloco }}</label>
                <input #price id="transport-price" class="ui-input" type="number" inputmode="numeric" />
              </div>
            </div>
          </div>
          <div class="ui-field">
            <label class="ui-label" for="transport-note">{{ 'crm.payment.note' | transloco }}</label>
            <input #note id="transport-note" class="ui-input" maxlength="2000" [placeholder]="'transport.notePlaceholder' | transloco" />
          </div>
          <div class="flex gap-2">
            <button type="submit" class="ui-btn-primary" [disabled]="busy() || !driverId()"><span class="icon-[heroicons--paper-airplane] size-4" aria-hidden="true"></span>{{ 'transport.send' | transloco }}</button>
            <button type="button" class="ui-btn-ghost" (click)="asking.set(false)">{{ 'crm.cancel' | transloco }}</button>
          </div>
        </form>
      }
    </section>
  `,
})
export class TourTransport {
  readonly group = input.required<CrmGroup>();

  private readonly http = inject(HttpClient);
  private readonly transloco = inject(TranslocoService);
  private readonly notify = inject(Notify);
  private readonly confirm = injectConfirm();
  private readonly navigate = injectSiteNavigate();
  private readonly lang = injectLang();
  protected readonly asking = signal(false);
  protected readonly busy = signal(false);

  private readonly loadedRequests = httpResource<TransportRequest[]>(
    () => `/crm/tours/${this.group().tour.id}/transport`,
  );
  private readonly fresh = signal<TransportRequest[] | null>(null);
  protected readonly requests = computed(() => this.fresh() ?? loaded(this.loadedRequests) ?? []);
  // The catalog loads once the organizer starts a request.
  protected readonly catalog = httpResource<CatalogDriver[]>(() =>
    this.asking()
      ? {
          url: '/crm/drivers',
          params: {
            date: this.group().tour.startDate,
            seats: Math.max(this.group().totals.active, 1),
          },
        }
      : undefined,
  );
  protected readonly driverOptions = computed(() =>
    (loaded(this.catalog) ?? []).map((d) => {
      const seats = Math.max(...d.vehicles.map((v) => v.seats));
      const busy = d.busy ? ` · ${this.transloco.translate('transport.busy')}` : '';
      return {
        value: d.userId,
        label: `${d.name} · ${this.transloco.translate('driver.vehicle.seatsCount', { count: seats })}${busy}`,
      };
    }),
  );
  protected readonly driverId = linkedSignal(() => {
    const free = (loaded(this.catalog) ?? []).find((d) => !d.busy);
    return free?.userId ?? '';
  });
  protected readonly chosen = computed(() =>
    (loaded(this.catalog) ?? []).find((d) => d.userId === this.driverId()),
  );
  // The tour's start at the meeting time, for the pickup field.
  protected readonly pickup = computed(() => {
    const tour = this.group().tour;
    return `${tour.startDate}T${tour.meetingTime ?? '07:00'}`;
  });
  protected readonly badge = tripBadge;

  protected async ask(
    event: Event,
    routeFrom: string,
    routeTo: string,
    starts: string,
    passengers: string,
    price: string,
    note: string,
  ): Promise<void> {
    event.preventDefault();
    if (!routeFrom.trim() || !routeTo.trim() || !starts) {
      this.notify.warning({ title: 'transport.errors.required' });
      return;
    }
    const whole = (s: string): number | null => (/^\d+$/.test(s) ? Number(s) : null);
    this.busy.set(true);
    try {
      this.fresh.set(
        await firstValueFrom(
          this.http.post<TransportRequest[]>(`/crm/tours/${this.group().tour.id}/transport`, {
            driverId: this.driverId(),
            routeFrom: routeFrom.trim(),
            routeTo: routeTo.trim(),
            startsAt: fromLocalInput(starts),
            passengers: whole(passengers),
            price: whole(price),
            note: note.trim() || null,
          }),
        ),
      );
      this.asking.set(false);
      this.notify.success({ title: 'transport.sent' });
    } catch {
      this.notify.error({ title: 'crm.saveError' });
    } finally {
      this.busy.set(false);
    }
  }

  protected async callOff(r: TransportRequest): Promise<void> {
    if (!(await this.confirm({ title: 'transport.callOffConfirm', danger: true }))) return;
    this.busy.set(true);
    try {
      this.fresh.set(
        await firstValueFrom(this.http.delete<TransportRequest[]>(`/crm/transport/${r.id}`)),
      );
    } catch {
      this.notify.error({ title: 'crm.saveError' });
    } finally {
      this.busy.set(false);
    }
  }

  protected async write(partnerId: string): Promise<void> {
    try {
      const chat = await openChat(this.http, { partnerId });
      await this.navigate(`/partner/messages/${chat.id}`);
    } catch {
      this.notify.error({ title: 'chat.openError' });
    }
  }

  protected vehicle(r: TransportRequest): string {
    return r.vehicle ? vehicleLabel(r.vehicle) : '';
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
}
