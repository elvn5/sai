import type { DriverVehicle } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { formatDate, todayInBishkek } from '@/core/lib/format';
import { loaded } from '@/core/lib/resource';
import { Notify } from '@/core/notify/notify';
import { bindPageMeta } from '@/core/seo/seo.service';
import { daysUntil } from '@/pages/partner/driver/driver';
import { VehicleForm } from '@/pages/partner/driver/vehicle-form';
import { injectConfirm } from '@/ui/confirm-dialog';
import { ListSkeleton } from '@/ui/list-skeleton';
import { LoadError } from '@/ui/load-error';
import { StateBlock } from '@/ui/state-block';
import { HttpClient, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// Documents due within this many days are flagged.
const WARN_DAYS = 30;

// /partner/fleet: the driver's vehicles with seats and plates, and when
// each one's insurance and roadworthiness check run out.
@Component({
  selector: 'app-driver-fleet-page',
  imports: [ListSkeleton, LoadError, StateBlock, TranslocoPipe, VehicleForm],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-5">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 class="text-2xl font-extrabold tracking-tight text-text">{{ 'cabinet.nav.fleet' | transloco }}</h1>
          <p class="text-sm text-text-muted">{{ 'driver.fleet.subtitle' | transloco }}</p>
        </div>
        @if (editing() === null) {
          <button type="button" class="ui-btn-primary" (click)="editing.set('new')"><span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>{{ 'driver.vehicle.add' | transloco }}</button>
        }
      </div>

      @if (editing() === 'new') {
        <section class="ui-card p-4" aria-labelledby="new-vehicle">
          <h2 id="new-vehicle" class="mb-3 font-semibold text-text">{{ 'driver.vehicle.new' | transloco }}</h2>
          <app-vehicle-form (saved)="saved()" (cancelled)="editing.set(null)" />
        </section>
      }

      @if (vehicles.error()) {
        <app-load-error (retry)="vehicles.reload()" />
      } @else if (vehicles.isLoading() && !vehicles.hasValue()) {
        <app-list-skeleton variant="rows" [count]="2" />
      } @else if (list().length === 0) {
        <app-state-block icon="icon-[heroicons--truck]" title="driver.fleet.empty" text="driver.fleet.emptyHint" />
      } @else {
        <ul class="flex flex-col gap-3">
          @for (v of list(); track v.id) {
            <li class="ui-card p-4" [class.opacity-70]="!v.active">
              @if (editing() === v.id) {
                <app-vehicle-form [vehicle]="v" (saved)="saved()" (cancelled)="editing.set(null)" />
              } @else {
                <div class="flex flex-wrap items-start gap-3">
                  <span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-brand-500/10 text-brand-600 dark:text-brand-300"><span class="icon-[heroicons--truck] size-5" aria-hidden="true"></span></span>
                  <div class="min-w-0 flex-1">
                    <p class="flex flex-wrap items-center gap-2">
                      <b class="text-text">{{ v.model }}</b>
                      <span class="ui-badge ui-badge-soft">{{ 'driver.vehicle.kinds.' + v.kind | transloco }}</span>
                      @if (!v.active) {
                        <span class="ui-badge ui-badge-muted">{{ 'driver.vehicle.inactive' | transloco }}</span>
                      }
                    </p>
                    <p class="text-sm text-text-muted">
                      @if (v.plate) { <span class="font-mono">{{ v.plate }}</span> · }{{ 'driver.vehicle.seatsCount' | transloco: { count: v.seats } }}@if (v.year) { · {{ v.year }}}
                    </p>
                    <ul class="mt-2 flex flex-wrap gap-2 text-xs">
                      @for (doc of [{ kind: 'insurance', until: v.insuranceUntil }, { kind: 'inspection', until: v.inspectionUntil }]; track doc.kind) {
                        <li [class]="docClass(doc.until)">
                          {{ 'driver.documents.' + doc.kind | transloco }}:
                          {{ doc.until ? date(doc.until) : ('driver.documents.unknown' | transloco) }}
                        </li>
                      }
                    </ul>
                    @if (v.note) {
                      <p class="mt-2 text-sm text-text-muted">{{ v.note }}</p>
                    }
                  </div>
                  <div class="flex gap-1">
                    <button type="button" class="ui-icon-btn" [attr.aria-label]="'listing.edit' | transloco" (click)="editing.set(v.id)"><span class="icon-[heroicons--pencil-square] size-4" aria-hidden="true"></span></button>
                    <button type="button" class="ui-icon-btn" [attr.aria-label]="'driver.vehicle.delete' | transloco" (click)="remove(v)"><span class="icon-[heroicons--trash] size-4" aria-hidden="true"></span></button>
                  </div>
                </div>
              }
            </li>
          }
        </ul>
      }
    </div>
  `,
})
export class DriverFleetPage {
  private readonly http = inject(HttpClient);
  private readonly transloco = inject(TranslocoService);
  private readonly notify = inject(Notify);
  private readonly confirm = injectConfirm();
  private readonly lang = injectLang();
  private readonly today = todayInBishkek();
  protected readonly vehicles = httpResource<DriverVehicle[]>(() => '/driver/vehicles');
  protected readonly list = computed(() => loaded(this.vehicles) ?? []);
  // The vehicle being edited ('new': adding one).
  protected readonly editing = signal<string | null>(null);

  constructor() {
    bindPageMeta(() => ({
      title: this.transloco.translate('cabinet.nav.fleet'),
      description: this.transloco.translate('driver.fleet.subtitle'),
      robots: 'noindex',
    }));
  }

  protected saved(): void {
    this.editing.set(null);
    this.vehicles.reload();
    this.notify.success({ title: 'driver.vehicle.saved' });
  }

  protected async remove(v: DriverVehicle): Promise<void> {
    if (!(await this.confirm({ title: 'driver.vehicle.deleteConfirm', danger: true }))) return;
    try {
      await firstValueFrom(this.http.delete(`/driver/vehicles/${v.id}`));
      this.vehicles.reload();
    } catch {
      this.notify.error({ title: 'crm.saveError' });
    }
  }

  protected date(day: string): string {
    return formatDate(this.lang(), day, { day: 'numeric', month: 'short', year: 'numeric' });
  }

  // A badge per document: red once it ran out, amber within a month.
  protected docClass(until: string | null): string {
    if (!until) return 'ui-badge ui-badge-muted';
    const left = daysUntil(this.today, until);
    if (left < 0) return 'ui-badge ui-badge-danger';
    return left <= WARN_DAYS ? 'ui-badge ui-badge-warning' : 'ui-badge ui-badge-success';
  }
}
