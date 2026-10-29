import type { CrmClient, DriverTrip, DriverVehicle, Paginated } from '@/core/api/types';
import { loaded } from '@/core/lib/resource';
import { clientName } from '@/pages/partner/crm';
import { fromLocalInput, toLocalInput, vehicleLabel } from '@/pages/partner/driver/driver';
import { FieldErrors } from '@/ui/field-errors';
import { SelectField, injectOptions } from '@/ui/select-field';
import { HttpClient, httpResource } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  linkedSignal,
  output,
  signal,
} from '@angular/core';
import { FormField, form, submit, validateStandardSchema } from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';
import { z } from 'zod/mini';

// A trip the driver writes down (taken on by phone, WhatsApp…) or edits:
// when, from where to where, the vehicle, the customer, passengers, price
// and a note. Times are typed in Bishkek time.
@Component({
  selector: 'app-trip-form',
  imports: [FieldErrors, FormField, SelectField, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <form class="flex flex-col gap-3" novalidate (submit)="save($event)">
      <div class="grid gap-3 sm:grid-cols-2">
        <div class="ui-field">
          <label class="ui-label" [for]="prefix() + 'from'">{{ 'driver.trip.from' | transloco }}</label>
          <input class="ui-input" [id]="prefix() + 'from'" autocomplete="off" [placeholder]="'driver.trip.fromPlaceholder' | transloco" [formField]="tripForm.routeFrom" />
          <app-field-errors [state]="tripForm.routeFrom()" />
        </div>
        <div class="ui-field">
          <label class="ui-label" [for]="prefix() + 'to'">{{ 'driver.trip.to' | transloco }}</label>
          <input class="ui-input" [id]="prefix() + 'to'" autocomplete="off" [placeholder]="'driver.trip.toPlaceholder' | transloco" [formField]="tripForm.routeTo" />
          <app-field-errors [state]="tripForm.routeTo()" />
        </div>
        <div class="ui-field">
          <label class="ui-label" [for]="prefix() + 'starts'">{{ 'driver.trip.startsAt' | transloco }}</label>
          <input class="ui-input" type="datetime-local" [id]="prefix() + 'starts'" [formField]="tripForm.startsAt" />
          <app-field-errors [state]="tripForm.startsAt()" />
        </div>
        <div class="ui-field">
          <label class="ui-label" [for]="prefix() + 'ends'">{{ 'driver.trip.endsAt' | transloco }}</label>
          <input class="ui-input" type="datetime-local" [id]="prefix() + 'ends'" [formField]="tripForm.endsAt" />
          <app-field-errors [state]="tripForm.endsAt()" />
        </div>
        <div class="ui-field">
          <span class="ui-label" [id]="prefix() + 'vehicle'">{{ 'driver.trip.vehicle' | transloco }}</span>
          <app-select-field [options]="vehicleOptions()" [labelledBy]="prefix() + 'vehicle'" [formField]="tripForm.vehicleId" />
        </div>
        @if (!trip()?.tour) {
          <div class="ui-field">
            <span class="ui-label" [id]="prefix() + 'client'">{{ 'driver.trip.client' | transloco }}</span>
            <app-select-field [options]="clientOptions()" [labelledBy]="prefix() + 'client'" [formField]="tripForm.clientId" />
          </div>
        }
        <div class="ui-field">
          <label class="ui-label" [for]="prefix() + 'passengers'">{{ 'driver.trip.passengers' | transloco }}</label>
          <input class="ui-input" type="number" inputmode="numeric" [id]="prefix() + 'passengers'" [formField]="tripForm.passengers" />
          <app-field-errors [state]="tripForm.passengers()" />
        </div>
        <div class="ui-field">
          <label class="ui-label" [for]="prefix() + 'price'">{{ 'driver.trip.price' | transloco }}</label>
          <input class="ui-input" type="number" inputmode="numeric" [id]="prefix() + 'price'" [formField]="tripForm.price" />
          <app-field-errors [state]="tripForm.price()" />
        </div>
      </div>
      <div class="ui-field">
        <label class="ui-label" [for]="prefix() + 'note'">{{ 'crm.client.note' | transloco }}</label>
        <textarea class="ui-input" rows="2" [id]="prefix() + 'note'" [formField]="tripForm.note"></textarea>
        <app-field-errors [state]="tripForm.note()" />
      </div>
      @if (failed()) {
        <p class="ui-error" role="alert">{{ 'crm.saveError' | transloco }}</p>
      }
      <div class="flex gap-2">
        <button type="submit" class="ui-btn-primary" [disabled]="tripForm().submitting()">{{ (trip() ? 'crm.save' : 'driver.trip.add') | transloco }}</button>
        <button type="button" class="ui-btn-ghost" (click)="cancelled.emit()">{{ 'crm.cancel' | transloco }}</button>
      </div>
    </form>
  `,
})
export class TripForm {
  // The trip to edit; none: a new one.
  readonly trip = input<DriverTrip | null>(null);
  readonly saved = output<DriverTrip>();
  readonly cancelled = output();

  private readonly http = inject(HttpClient);
  private readonly transloco = inject(TranslocoService);
  protected readonly failed = signal(false);
  protected readonly prefix = computed(() => `trip-${this.trip()?.id ?? 'new'}-`);

  private readonly vehicles = httpResource<DriverVehicle[]>(() => '/driver/vehicles');
  private readonly clients = httpResource<Paginated<CrmClient>>(() => ({
    url: '/crm/clients',
    params: { limit: 200 },
  }));
  protected readonly vehicleOptions = injectOptions(() => [
    { value: '', key: 'driver.trip.noVehicle' },
    ...(loaded(this.vehicles) ?? [])
      .filter((v) => v.active || v.id === this.trip()?.vehicle?.id)
      .map((v) => ({ value: v.id, label: vehicleLabel(v) })),
  ]);
  protected readonly clientOptions = injectOptions(() => [
    { value: '', key: 'driver.trip.noClient' },
    ...(loaded(this.clients)?.items ?? []).map((c) => ({ value: c.id, label: clientName(c) })),
  ]);

  private readonly model = linkedSignal(() => {
    const t = this.trip();
    return {
      routeFrom: t?.routeFrom ?? '',
      routeTo: t?.routeTo ?? '',
      startsAt: t ? toLocalInput(t.startsAt) : '',
      endsAt: t?.endsAt ? toLocalInput(t.endsAt) : '',
      vehicleId: t?.vehicle?.id ?? '',
      clientId: t?.client?.id ?? '',
      passengers: t?.passengers?.toString() ?? '',
      price: t?.price?.toString() ?? '',
      note: t?.note ?? '',
    };
  });

  protected readonly tripForm = form(this.model, (path) => {
    const t = (key: string): string => this.transloco.translate(key);
    const place = z
      .string()
      .check(
        z.trim(),
        z.minLength(1, t('driver.errors.place')),
        z.maxLength(200, t('crm.errors.tooLong')),
      );
    const whole = (maxValue: number) =>
      z
        .string()
        .check(
          z.refine(
            (v) => v === '' || (/^\d+$/.test(v) && Number(v) <= maxValue),
            t('driver.errors.number'),
          ),
        );
    validateStandardSchema(
      path,
      z
        .object({
          routeFrom: place,
          routeTo: place,
          startsAt: z.string().check(z.minLength(1, t('driver.errors.startsAt'))),
          endsAt: z.string(),
          vehicleId: z.string(),
          clientId: z.string(),
          passengers: whole(80),
          price: whole(100_000_000),
          note: z.string().check(z.maxLength(2000, t('crm.errors.tooLong'))),
        })
        .check(
          z.refine((v) => !v.endsAt || !v.startsAt || v.endsAt >= v.startsAt, {
            message: t('driver.errors.endsAt'),
            path: ['endsAt'],
          }),
        ),
    );
  });

  protected async save(event: Event): Promise<void> {
    event.preventDefault();
    this.failed.set(false);
    const v = this.model();
    const numberOrNull = (s: string): number | null => (s === '' ? null : Number(s));
    const body = {
      routeFrom: v.routeFrom.trim(),
      routeTo: v.routeTo.trim(),
      startsAt: fromLocalInput(v.startsAt),
      endsAt: fromLocalInput(v.endsAt),
      vehicleId: v.vehicleId || null,
      clientId: v.clientId || null,
      passengers: numberOrNull(v.passengers),
      price: numberOrNull(v.price),
      note: v.note.trim() || null,
    };
    await submit(this.tripForm, async () => {
      try {
        const current = this.trip();
        this.saved.emit(
          await firstValueFrom(
            current
              ? this.http.put<DriverTrip>(`/driver/trips/${current.id}`, body)
              : this.http.post<DriverTrip>('/driver/trips', body),
          ),
        );
      } catch {
        this.failed.set(true);
      }
      return undefined;
    });
  }
}
