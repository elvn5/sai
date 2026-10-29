import type { DriverTrip, TripStatus } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, formatInBishkek } from '@/core/lib/format';
import { downloadCsv } from '@/core/lib/partner-cabinet';
import { loaded } from '@/core/lib/resource';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { bindPageMeta } from '@/core/seo/seo.service';
import {
  TRIP_STATUSES,
  tripBadge,
  tripClientName,
  vehicleLabel,
} from '@/pages/partner/driver/driver';
import { TripForm } from '@/pages/partner/driver/trip-form';
import { ListSkeleton } from '@/ui/list-skeleton';
import { LoadError } from '@/ui/load-error';
import { SiteLink } from '@/ui/site-link';
import { StateBlock } from '@/ui/state-block';
import { DOCUMENT } from '@angular/common';
import { httpResource } from '@angular/common/http';
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

// /partner/trips: the driver's trips — organizers' requests waiting for an
// answer, the ones ahead, done and called off — with what each customer
// still owes; a trip taken on elsewhere is written down here. ?status=
// opens a filter.
@Component({
  selector: 'app-driver-trips-page',
  imports: [ListSkeleton, LoadError, SiteLink, StateBlock, TranslocoPipe, TripForm],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-5">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 class="text-2xl font-extrabold tracking-tight text-text">{{ 'cabinet.nav.trips' | transloco }}</h1>
          <p class="text-sm text-text-muted">{{ 'driver.trips.subtitle' | transloco }}</p>
        </div>
        <div class="flex gap-2">
          <button type="button" class="ui-btn-soft" [disabled]="list().length === 0" (click)="exportCsv()"><span class="icon-[heroicons--arrow-down-tray] size-4" aria-hidden="true"></span>CSV</button>
          @if (!adding()) {
            <button type="button" class="ui-btn-primary" (click)="adding.set(true)"><span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>{{ 'driver.trip.add' | transloco }}</button>
          }
        </div>
      </div>

      @if (adding()) {
        <section class="ui-card p-4" aria-labelledby="new-trip">
          <h2 id="new-trip" class="mb-3 font-semibold text-text">{{ 'driver.trip.new' | transloco }}</h2>
          <app-trip-form (saved)="created($event)" (cancelled)="adding.set(false)" />
        </section>
      }

      <div class="flex flex-wrap gap-1.5" role="group" [attr.aria-label]="'driver.trips.filter' | transloco">
        <button type="button" class="ui-pill" [attr.aria-pressed]="filter() === ''" (click)="filter.set('')">{{ 'driver.trips.all' | transloco }}</button>
        @for (s of statuses; track s) {
          <button type="button" class="ui-pill" [attr.aria-pressed]="filter() === s" (click)="filter.set(s)">{{ 'driver.status.' + s | transloco }}</button>
        }
      </div>

      @if (trips.isLoading() && !trips.hasValue()) {
        <app-list-skeleton variant="rows" [count]="4" />
      } @else if (trips.error()) {
        <app-load-error (retry)="trips.reload()" />
      } @else if (list().length === 0) {
        <app-state-block icon="icon-[heroicons--truck]" title="driver.trips.empty" text="driver.trips.emptyHint" />
      } @else {
        <ul class="ui-card flex flex-col divide-y divide-border">
          @for (trip of list(); track trip.id) {
            <li>
              <a [appSiteLink]="'/partner/trips/' + trip.id" class="flex flex-wrap items-center gap-x-4 gap-y-1 px-4 py-3 hover:bg-text/3">
                <span class="min-w-0 flex-1">
                  <span class="flex flex-wrap items-center gap-2">
                    <span class="truncate font-medium text-text">{{ trip.routeFrom }} → {{ trip.routeTo }}</span>
                    @if (trip.tour) {
                      <span class="ui-badge ui-badge-soft"><span class="icon-[heroicons--map] size-3.5" aria-hidden="true"></span>Ermekten</span>
                    }
                  </span>
                  <span class="block text-xs text-text-muted">
                    {{ when(trip.startsAt) }}
                    @if (clientName(trip); as name) { · {{ name }}}
                    @if (trip.vehicle) { · {{ vehicle(trip) }}}
                    @if (trip.passengers) { · {{ 'driver.trip.people' | transloco: { count: trip.passengers } }}}
                  </span>
                </span>
                @if (trip.price !== null) {
                  <span class="text-sm tabular-nums" [class]="trip.due > 0 ? 'text-amber-700 dark:text-amber-300' : 'text-text-muted'">{{ num(trip.paid) }} / {{ num(trip.price) }}</span>
                }
                <span [class]="badge(trip.status)">{{ 'driver.status.' + trip.status | transloco }}</span>
              </a>
            </li>
          }
        </ul>
      }
    </div>
  `,
})
export class DriverTripsPage {
  readonly status = input<string | undefined>(undefined);

  private readonly transloco = inject(TranslocoService);
  private readonly lang = injectLang();
  private readonly navigate = injectSiteNavigate();
  private readonly document = inject(DOCUMENT);
  protected readonly statuses = TRIP_STATUSES;
  protected readonly filter = linkedSignal<TripStatus | ''>(() => {
    const s = this.status();
    return TRIP_STATUSES.includes(s as TripStatus) ? (s as TripStatus) : '';
  });
  protected readonly adding = signal(false);
  protected readonly trips = httpResource<DriverTrip[]>(() => ({
    url: '/driver/trips',
    params: { status: this.filter() },
  }));
  protected readonly list = computed(() => loaded(this.trips) ?? []);
  protected readonly badge = tripBadge;
  protected readonly clientName = tripClientName;

  constructor() {
    bindPageMeta(() => ({
      title: this.transloco.translate('cabinet.nav.trips'),
      description: this.transloco.translate('driver.trips.subtitle'),
      robots: 'noindex',
    }));
  }

  protected created(trip: DriverTrip): void {
    this.adding.set(false);
    void this.navigate(`/partner/trips/${trip.id}`);
  }

  protected num(value: number): string {
    return formatAmount(this.lang(), value);
  }

  protected vehicle(trip: DriverTrip): string {
    return trip.vehicle ? vehicleLabel(trip.vehicle) : '';
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

  protected exportCsv(): void {
    const t = (key: string): string => this.transloco.translate(key);
    const rows = this.list().map((trip) => [
      this.when(trip.startsAt),
      trip.routeFrom,
      trip.routeTo,
      tripClientName(trip),
      trip.tour?.title ?? null,
      trip.vehicle ? vehicleLabel(trip.vehicle) : null,
      trip.passengers,
      trip.price,
      trip.paid,
      trip.due,
      t(`driver.status.${trip.status}`),
    ]);
    const headers = [
      'driver.trip.startsAt',
      'driver.trip.from',
      'driver.trip.to',
      'driver.trip.client',
      'driver.trip.tour',
      'driver.trip.vehicle',
      'driver.trip.passengers',
      'driver.trip.price',
      'crm.paid',
      'crm.debt',
      'driver.trip.status',
    ].map(t);
    downloadCsv(this.document, 'ermekten-trips.csv', [headers, ...rows]);
  }
}
