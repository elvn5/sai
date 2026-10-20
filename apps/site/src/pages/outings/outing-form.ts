import type { Outing, Trail } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { emptyToNull } from '@/core/lib/listing-form';
import { loaded } from '@/core/lib/resource';
import { LEVELS, TRANSPORTS } from '@/core/lib/trail-options';
import { DateTimeField } from '@/ui/date-time-field';
import { FieldErrors } from '@/ui/field-errors';
import { type IconOption, IconPicker } from '@/ui/icon-picker';
import { SelectField, type SelectOption } from '@/ui/select-field';
import { httpResource } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  inject,
  input,
  linkedSignal,
  signal,
  untracked,
} from '@angular/core';
import {
  FormField,
  form,
  max,
  maxLength,
  min,
  submit,
  validate,
  validateStandardSchema,
} from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { z } from 'zod/mini';

let nextFormId = 0;

// Kyrgyzstan doesn't observe DST: always UTC+6.
const BISHKEK_OFFSET_MS = 6 * 3_600_000;

// What the API takes to create or update an outing.
export interface OutingPayload {
  trailId: string | null;
  title: string;
  startsAt: string;
  returnsAt: string | null;
  meetingPoint: string;
  transport: string;
  seats: number;
  costPerPerson: number | null;
  level: string;
  description: string | null;
  contact: string | null;
}

interface OutingFormModel {
  trailId: string;
  title: string;
  startsAt: string;
  returnsAt: string;
  meetingPoint: string;
  transport: string;
  seats: number | null;
  costPerPerson: number | null;
  level: string;
  description: string;
  contact: string;
}

// The date fields hold wall-clock time without a zone; outings are
// planned in Bishkek time, so they hold Bishkek wall-clock time.
export function toBishkekInput(iso: string): string {
  return new Date(new Date(iso).getTime() + BISHKEK_OFFSET_MS).toISOString().slice(0, 16);
}

export function fromBishkekInput(value: string): string {
  return `${value}:00+06:00`;
}

// Next Saturday, 06:00 in Bishkek.
export function nextSaturdayMorning(now: number = Date.now()): string {
  const local = new Date(now + BISHKEK_OFFSET_MS);
  const days = (6 - local.getUTCDay() + 7) % 7 || 7;
  return new Date(
    Date.UTC(local.getUTCFullYear(), local.getUTCMonth(), local.getUTCDate() + days, 6, 0),
  )
    .toISOString()
    .slice(0, 16);
}

function toModel(outing: Outing | undefined): OutingFormModel {
  return {
    trailId: outing?.trail?.id ?? '',
    title: outing?.title ?? '',
    startsAt: outing ? toBishkekInput(outing.startsAt) : nextSaturdayMorning(),
    returnsAt: outing?.returnsAt ? toBishkekInput(outing.returnsAt) : '',
    meetingPoint: outing?.meetingPoint ?? '',
    transport: outing?.transport ?? 'car',
    seats: outing?.seats ?? 3,
    costPerPerson: outing?.costPerPerson ?? null,
    level: outing?.level ?? 'beginner',
    description: outing?.description ?? '',
    contact: outing?.contact ?? '',
  };
}

// Organize an outing, or edit `outing`. A route from the catalog is
// optional; ?trail=<slug> picks it (and its name as the title).
@Component({
  selector: 'app-outing-form',
  imports: [DateTimeField, FieldErrors, FormField, IconPicker, SelectField, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <form class="flex flex-col gap-6" novalidate (submit)="send($event)">
      <div class="ui-field">
        <span [id]="ids + '-trail'">{{ 'outings.form.trail' | transloco }}</span>
        <app-select-field [options]="trailOptions()" [labelledBy]="ids + '-trail'" [formField]="outingForm.trailId" />
        <span class="ui-hint">{{ 'outings.form.trailHint' | transloco }}</span>
      </div>
      <label class="ui-field">
        {{ 'outings.form.title' | transloco }} *
        <input class="ui-input py-2.5 text-base" [placeholder]="'outings.form.titlePlaceholder' | transloco" [formField]="outingForm.title" />
        <app-field-errors [state]="outingForm.title()" />
      </label>
      <div class="grid gap-4 sm:grid-cols-2">
        <div class="ui-field">
          <span [id]="ids + '-starts'">{{ 'outings.form.startsAt' | transloco }} *</span>
          <app-date-time-field [earliest]="today" defaultTime="06:00" [labelledBy]="ids + '-starts'" [formField]="outingForm.startsAt" />
          <span class="ui-hint">{{ 'outings.form.bishkekTime' | transloco }}</span>
          <app-field-errors [state]="outingForm.startsAt()" />
        </div>
        <div class="ui-field">
          <span [id]="ids + '-returns'">{{ 'outings.form.returnsAt' | transloco }}</span>
          <app-date-time-field
            [earliest]="startDay()"
            defaultTime="18:00"
            anchor="end"
            [clearable]="true"
            [labelledBy]="ids + '-returns'"
            [formField]="outingForm.returnsAt"
          />
          <app-field-errors [state]="outingForm.returnsAt()" />
        </div>
      </div>
      <label class="ui-field">
        {{ 'outings.form.meetingPoint' | transloco }} *
        <input class="ui-input" [placeholder]="'outings.form.meetingPointPlaceholder' | transloco" [formField]="outingForm.meetingPoint" />
        <app-field-errors [state]="outingForm.meetingPoint()" />
      </label>
      <div class="ui-field">
        <span>{{ 'outings.form.transport' | transloco }}</span>
        <app-icon-picker [options]="transportOptions()" [formField]="outingForm.transport" />
      </div>
      <div class="grid gap-4 sm:grid-cols-2">
        <label class="ui-field">
          {{ 'outings.form.seats' | transloco }}
          <input class="ui-input" type="number" inputmode="numeric" [formField]="outingForm.seats" />
          <span class="ui-hint">{{ 'outings.form.seatsHint' | transloco }}</span>
          <app-field-errors [state]="outingForm.seats()" />
        </label>
        <label class="ui-field">
          {{ 'outings.form.cost' | transloco }}
          <input class="ui-input" type="number" inputmode="numeric" [formField]="outingForm.costPerPerson" />
          <span class="ui-hint">{{ 'outings.form.costHint' | transloco }}</span>
          <app-field-errors [state]="outingForm.costPerPerson()" />
        </label>
      </div>
      <div class="ui-field">
        <span>{{ 'outings.form.level' | transloco }}</span>
        <app-icon-picker [options]="levelOptions()" [formField]="outingForm.level" />
      </div>
      <label class="ui-field">
        {{ 'outings.form.description' | transloco }}
        <textarea class="ui-input" rows="4" [placeholder]="'outings.form.descriptionPlaceholder' | transloco" [formField]="outingForm.description"></textarea>
        <app-field-errors [state]="outingForm.description()" />
      </label>
      <label class="ui-field">
        {{ 'outings.form.contact' | transloco }}
        <input class="ui-input" placeholder="@telegram / +996…" [formField]="outingForm.contact" />
        <span class="ui-hint">{{ 'outings.form.contactHint' | transloco }}</span>
        <app-field-errors [state]="outingForm.contact()" />
      </label>

      @if (failed(); as key) {
        <p class="ui-error" role="alert">{{ key | transloco }}</p>
      }
      <button type="submit" class="ui-btn-primary py-2.5 text-base" [disabled]="outingForm().submitting()">{{ submitLabel() }}</button>
    </form>
  `,
})
export class OutingForm {
  readonly outing = input<Outing | undefined>(undefined);
  // Prefills the route (from a trail page's "organize" link).
  readonly trailSlug = input<string | undefined>(undefined);
  readonly submitLabel = input.required<string>();
  readonly save = input.required<(payload: OutingPayload) => Promise<void>>();

  private readonly transloco = inject(TranslocoService);
  private readonly lang = injectLang();

  private readonly trails = httpResource<Trail[]>(() => ({
    url: '/trails',
    params: { sort: 'name' },
  }));
  protected readonly trailList = computed(() => loaded(this.trails) ?? []);
  protected readonly trailOptions = computed<SelectOption[]>(() => {
    this.lang();
    return [
      { value: '', label: this.transloco.translate('outings.form.noTrail') },
      ...this.trailList().map((trail) => ({ value: trail.id, label: trail.name })),
    ];
  });
  // Ids tying the custom fields to their labels.
  protected readonly ids = `outing-form-${nextFormId++}`;
  // Bishkek's today: no outing starts earlier.
  protected readonly today = new Date(Date.now() + BISHKEK_OFFSET_MS).toISOString().slice(0, 10);
  // Options read the language at render: labels follow the switcher.
  protected readonly transportOptions = computed(() =>
    this.options(TRANSPORTS, 'outings.transport'),
  );
  protected readonly levelOptions = computed(() => this.options(LEVELS, 'outings.level'));
  // Translation key of the submit error.
  protected readonly failed = signal<string | null>(null);

  protected readonly model = linkedSignal(() => toModel(this.outing()));
  // Nobody comes back before leaving.
  protected readonly startDay = computed(() => this.model().startsAt.slice(0, 10) || this.today);
  protected readonly outingForm = form(this.model, (path) => {
    const t = (key: string): string => this.transloco.translate(key);
    const required = t('login.required');
    const message = t('outings.form.checkFields');
    validateStandardSchema(
      path,
      z.object({
        title: z.string().check(z.trim(), z.minLength(1, required)),
        meetingPoint: z.string().check(z.trim(), z.minLength(1, required)),
      }),
    );
    validate(path.startsAt, (field) =>
      field.value() && new Date(fromBishkekInput(field.value())).getTime() > Date.now()
        ? null
        : { kind: 'future', message: t('outings.form.futureStart') },
    );
    validate(path.returnsAt, (field) => {
      const returns = field.value();
      return !returns || returns > field.valueOf(path.startsAt)
        ? null
        : { kind: 'order', message: t('outings.form.returnAfterStart') };
    });
    maxLength(path.title, 160);
    maxLength(path.meetingPoint, 200);
    maxLength(path.description, 2000);
    maxLength(path.contact, 120);
    min(path.seats, 1, { message });
    max(path.seats, 50, { message });
    min(path.costPerPerson, 0, { message });
  });

  constructor() {
    // A new outing from a trail page: that route, named after it.
    effect(() => {
      const slug = this.trailSlug();
      const match = slug ? this.trailList().find((trail) => trail.slug === slug) : undefined;
      if (!match || this.outing()) return;
      untracked(() =>
        this.model.update((value) =>
          value.trailId ? value : { ...value, trailId: match.id, title: value.title || match.name },
        ),
      );
    });
  }

  protected async send(event: Event): Promise<void> {
    event.preventDefault();
    this.failed.set(null);
    await submit(this.outingForm, async () => {
      const value = this.model();
      try {
        await this.save()({
          trailId: value.trailId || null,
          title: value.title.trim(),
          startsAt: fromBishkekInput(value.startsAt),
          returnsAt: value.returnsAt ? fromBishkekInput(value.returnsAt) : null,
          meetingPoint: value.meetingPoint.trim(),
          transport: value.transport,
          seats: value.seats ?? 1,
          costPerPerson: value.costPerPerson,
          level: value.level,
          description: emptyToNull(value.description),
          contact: emptyToNull(value.contact),
        });
      } catch (error) {
        this.failed.set(
          (error as { status?: number }).status === 400
            ? 'outings.form.checkFields'
            : 'outings.form.saveError',
        );
      }
      return undefined;
    });
  }

  private options(options: { value: string; icon: string }[], prefix: string): IconOption[] {
    this.lang();
    return options.map((option) => ({
      ...option,
      label: this.transloco.translate(`${prefix}.${option.value}`),
    }));
  }
}
