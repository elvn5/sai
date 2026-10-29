import type { DriverVehicle } from '@/core/api/types';
import { VEHICLE_KINDS } from '@/pages/partner/driver/driver';
import { FieldErrors } from '@/ui/field-errors';
import { SelectField, injectOptions } from '@/ui/select-field';
import { HttpClient } from '@angular/common/http';
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

// A vehicle: its kind and model, plate, seats, year and when the insurance
// and the roadworthiness check run out (the dashboard warns a month
// ahead). Adds one, or edits it (and takes it out of service).
@Component({
  selector: 'app-vehicle-form',
  imports: [FieldErrors, FormField, SelectField, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <form class="flex flex-col gap-3" novalidate (submit)="save($event)">
      <div class="grid gap-3 sm:grid-cols-3">
        <div class="ui-field">
          <span class="ui-label" [id]="prefix() + 'kind'">{{ 'driver.vehicle.kind' | transloco }}</span>
          <app-select-field [options]="kindOptions()" [labelledBy]="prefix() + 'kind'" [formField]="vehicleForm.kind" />
        </div>
        <div class="ui-field sm:col-span-2">
          <label class="ui-label" [for]="prefix() + 'model'">{{ 'driver.vehicle.model' | transloco }}</label>
          <input class="ui-input" [id]="prefix() + 'model'" autocomplete="off" placeholder="Mercedes Sprinter" [formField]="vehicleForm.model" />
          <app-field-errors [state]="vehicleForm.model()" />
        </div>
        <div class="ui-field">
          <label class="ui-label" [for]="prefix() + 'plate'">{{ 'driver.vehicle.plate' | transloco }}</label>
          <input class="ui-input uppercase" [id]="prefix() + 'plate'" autocomplete="off" placeholder="01KG123ABC" [formField]="vehicleForm.plate" />
          <app-field-errors [state]="vehicleForm.plate()" />
        </div>
        <div class="ui-field">
          <label class="ui-label" [for]="prefix() + 'seats'">{{ 'driver.vehicle.seats' | transloco }}</label>
          <input class="ui-input" type="number" inputmode="numeric" [id]="prefix() + 'seats'" [formField]="vehicleForm.seats" />
          <app-field-errors [state]="vehicleForm.seats()" />
        </div>
        <div class="ui-field">
          <label class="ui-label" [for]="prefix() + 'year'">{{ 'driver.vehicle.year' | transloco }}</label>
          <input class="ui-input" type="number" inputmode="numeric" [id]="prefix() + 'year'" [formField]="vehicleForm.year" />
          <app-field-errors [state]="vehicleForm.year()" />
        </div>
        <div class="ui-field">
          <label class="ui-label" [for]="prefix() + 'insurance'">{{ 'driver.vehicle.insuranceUntil' | transloco }}</label>
          <input class="ui-input" type="date" [id]="prefix() + 'insurance'" [formField]="vehicleForm.insuranceUntil" />
        </div>
        <div class="ui-field">
          <label class="ui-label" [for]="prefix() + 'inspection'">{{ 'driver.vehicle.inspectionUntil' | transloco }}</label>
          <input class="ui-input" type="date" [id]="prefix() + 'inspection'" [formField]="vehicleForm.inspectionUntil" />
        </div>
        @if (vehicle()) {
          <label class="flex items-center gap-2 self-end pb-2 text-sm text-text">
            <input type="checkbox" class="size-4 accent-brand-600" [formField]="vehicleForm.active" />
            {{ 'driver.vehicle.active' | transloco }}
          </label>
        }
      </div>
      <div class="ui-field">
        <label class="ui-label" [for]="prefix() + 'note'">{{ 'crm.client.note' | transloco }}</label>
        <input class="ui-input" [id]="prefix() + 'note'" autocomplete="off" [placeholder]="'driver.vehicle.notePlaceholder' | transloco" [formField]="vehicleForm.note" />
        <app-field-errors [state]="vehicleForm.note()" />
      </div>
      @if (failed()) {
        <p class="ui-error" role="alert">{{ 'crm.saveError' | transloco }}</p>
      }
      <div class="flex gap-2">
        <button type="submit" class="ui-btn-primary" [disabled]="vehicleForm().submitting()">{{ (vehicle() ? 'crm.save' : 'driver.vehicle.add') | transloco }}</button>
        <button type="button" class="ui-btn-ghost" (click)="cancelled.emit()">{{ 'crm.cancel' | transloco }}</button>
      </div>
    </form>
  `,
})
export class VehicleForm {
  // The vehicle to edit; none: a new one.
  readonly vehicle = input<DriverVehicle | null>(null);
  readonly saved = output<DriverVehicle>();
  readonly cancelled = output();

  private readonly http = inject(HttpClient);
  private readonly transloco = inject(TranslocoService);
  protected readonly failed = signal(false);
  protected readonly prefix = computed(() => `vehicle-${this.vehicle()?.id ?? 'new'}-`);
  protected readonly kindOptions = injectOptions(() =>
    VEHICLE_KINDS.map((k) => ({ value: k, key: `driver.vehicle.kinds.${k}` })),
  );

  private readonly model = linkedSignal(() => {
    const v = this.vehicle();
    return {
      kind: v?.kind ?? 'minivan',
      model: v?.model ?? '',
      plate: v?.plate ?? '',
      seats: v?.seats.toString() ?? '',
      year: v?.year?.toString() ?? '',
      insuranceUntil: v?.insuranceUntil ?? '',
      inspectionUntil: v?.inspectionUntil ?? '',
      note: v?.note ?? '',
      active: v?.active ?? true,
    };
  });

  protected readonly vehicleForm = form(this.model, (path) => {
    const t = (key: string): string => this.transloco.translate(key);
    const between = (min: number, max: number, message: string, required: boolean) =>
      z
        .string()
        .check(
          z.refine(
            (v) =>
              (!required && v === '') || (/^\d+$/.test(v) && Number(v) >= min && Number(v) <= max),
            message,
          ),
        );
    validateStandardSchema(
      path,
      z.object({
        kind: z.string(),
        model: z
          .string()
          .check(
            z.trim(),
            z.minLength(1, t('driver.errors.model')),
            z.maxLength(120, t('crm.errors.tooLong')),
          ),
        plate: z.string().check(z.maxLength(20, t('crm.errors.tooLong'))),
        seats: between(1, 80, t('driver.errors.seats'), true),
        year: between(1950, 2100, t('driver.errors.year'), false),
        insuranceUntil: z.string(),
        inspectionUntil: z.string(),
        note: z.string().check(z.maxLength(500, t('crm.errors.tooLong'))),
        active: z.boolean(),
      }),
    );
  });

  protected async save(event: Event): Promise<void> {
    event.preventDefault();
    this.failed.set(false);
    const v = this.model();
    const body = {
      kind: v.kind,
      model: v.model.trim(),
      plate: v.plate.trim().toUpperCase() || null,
      seats: Number(v.seats),
      year: v.year ? Number(v.year) : null,
      insuranceUntil: v.insuranceUntil || null,
      inspectionUntil: v.inspectionUntil || null,
      note: v.note.trim() || null,
      active: v.active,
    };
    await submit(this.vehicleForm, async () => {
      try {
        const current = this.vehicle();
        this.saved.emit(
          await firstValueFrom(
            current
              ? this.http.put<DriverVehicle>(`/driver/vehicles/${current.id}`, body)
              : this.http.post<DriverVehicle>('/driver/vehicles', body),
          ),
        );
      } catch {
        this.failed.set(true);
      }
      return undefined;
    });
  }
}
