import type { MountainExperience, ParticipantForm } from '@/core/api/types';
import { FieldErrors } from '@/ui/field-errors';
import {
  ChangeDetectionStrategy,
  Component,
  inject,
  input,
  linkedSignal,
  output,
  signal,
} from '@angular/core';
import { FormField, form, submit, validateStandardSchema } from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { z } from 'zod/mini';

const PHONE = /^\+?[0-9\s\-()]{5,32}$/;
export const EXPERIENCE: MountainExperience[] = ['none', 'hiking', 'trekking', 'mountaineering'];
// The rules and risks text participants accept (the API's WaiverVersion):
// its points are translated under waiver.v1.
export const WAIVER_VERSION = 'v1';
const WAIVER_POINTS = ['health', 'guide', 'gear', 'weather', 'risk', 'insurance'];

// What a participant accepts and fills in: the form (emergency contact,
// experience, health) and — when `waiver` is on — the rules and risks of
// the trip. Used when signing up, to fill it in later, and by the partner
// for a client added by hand (no waiver).
@Component({
  selector: 'app-participant-form',
  imports: [FieldErrors, FormField, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <form class="flex flex-col gap-3" novalidate (submit)="save($event)">
      <div class="grid gap-3 sm:grid-cols-2">
        <div class="ui-field">
          <label class="ui-label" [for]="id('name')">{{ 'participantForm.emergencyName' | transloco }}</label>
          <input class="ui-input" [id]="id('name')" autocomplete="off" [formField]="pf.emergencyName" />
          <app-field-errors [state]="pf.emergencyName()" />
        </div>
        <div class="ui-field">
          <label class="ui-label" [for]="id('phone')">{{ 'participantForm.emergencyPhone' | transloco }}</label>
          <input class="ui-input" type="tel" [id]="id('phone')" autocomplete="off" placeholder="+996 555 123 456" [formField]="pf.emergencyPhone" />
          <app-field-errors [state]="pf.emergencyPhone()" />
        </div>
      </div>
      <fieldset class="ui-field">
        <legend class="ui-label">{{ 'participantForm.experience' | transloco }}</legend>
        <div class="grid gap-1.5 sm:grid-cols-2">
          @for (level of levels; track level) {
            <label class="flex cursor-pointer items-start gap-2 rounded-lg border px-3 py-2 text-sm" [class]="model().experience === level ? 'border-brand-500 bg-brand-500/8' : 'border-border'">
              <input type="radio" class="mt-0.5" [name]="id('experience')" [value]="level" [checked]="model().experience === level" (change)="setExperience(level)" />
              <span>
                <span class="block font-medium text-text">{{ 'participantForm.levels.' + level + '.title' | transloco }}</span>
                <span class="block text-xs text-text-muted">{{ 'participantForm.levels.' + level + '.hint' | transloco }}</span>
              </span>
            </label>
          }
        </div>
        <app-field-errors [state]="pf.experience()" />
      </fieldset>
      <div class="ui-field">
        <label class="ui-label" [for]="id('health')">{{ 'participantForm.health' | transloco }}</label>
        <textarea class="ui-input" rows="2" [id]="id('health')" [placeholder]="'participantForm.healthPlaceholder' | transloco" [formField]="pf.health"></textarea>
        <app-field-errors [state]="pf.health()" />
      </div>
      <div class="ui-field">
        <label class="ui-label" [for]="id('notes')">{{ 'participantForm.notes' | transloco }}</label>
        <textarea class="ui-input" rows="2" [id]="id('notes')" [placeholder]="'participantForm.notesPlaceholder' | transloco" [formField]="pf.notes"></textarea>
        <app-field-errors [state]="pf.notes()" />
      </div>

      @if (waiver()) {
        <div class="rounded-lg border border-border bg-text/3 p-3">
          <p class="font-semibold text-text">{{ 'waiver.v1.title' | transloco }}</p>
          <ul class="mt-2 flex list-disc flex-col gap-1 pl-5 text-sm text-text-muted">
            @for (point of points; track point) {
              <li>{{ 'waiver.v1.' + point | transloco }}</li>
            }
          </ul>
          <label class="mt-3 flex cursor-pointer items-start gap-2 text-sm font-medium text-text">
            <input type="checkbox" class="mt-0.5" [checked]="accepted()" (change)="accepted.set($any($event.target).checked)" />
            {{ 'waiver.v1.accept' | transloco }}
          </label>
          @if (waiverMissing()) {
            <p class="ui-error mt-1" role="alert">{{ 'participantForm.errors.waiver' | transloco }}</p>
          }
        </div>
      }

      @if (failed()) {
        <p class="ui-error" role="alert">{{ failed() | transloco }}</p>
      }
      <div class="flex flex-wrap gap-2">
        <button type="submit" class="ui-btn-primary" [disabled]="pf().submitting() || busy()">{{ submitLabel() | transloco }}</button>
        @if (cancellable()) {
          <button type="button" class="ui-btn-ghost" (click)="cancelled.emit()">{{ 'crm.cancel' | transloco }}</button>
        }
      </div>
    </form>
  `,
})
export class ParticipantFormView {
  // The form to edit (prefilled), if any.
  readonly value = input<ParticipantForm | null | undefined>(null);
  // Ask to accept the rules and risks too.
  readonly waiver = input(false);
  readonly submitLabel = input('participantForm.save');
  readonly cancellable = input(true);
  readonly busy = input(false);
  // Translation key of an error to show (set by the caller).
  readonly failed = input<string | null>(null);
  readonly saved = output<{ form: ParticipantForm; acceptWaiver: boolean }>();
  readonly cancelled = output();

  private readonly transloco = inject(TranslocoService);
  private static next = 0;
  private readonly uid = `pf-${ParticipantFormView.next++}`;
  protected readonly levels = EXPERIENCE;
  protected readonly points = WAIVER_POINTS;
  protected readonly accepted = signal(false);
  protected readonly waiverMissing = signal(false);

  protected readonly model = linkedSignal(() => {
    const v = this.value();
    return {
      emergencyName: v?.emergencyName ?? '',
      emergencyPhone: v?.emergencyPhone ?? '',
      experience: (v?.experience ?? '') as MountainExperience | '',
      health: v?.health ?? '',
      notes: v?.notes ?? '',
    };
  });

  protected readonly pf = form(this.model, (path) => {
    const t = (key: string): string => this.transloco.translate(key);
    validateStandardSchema(
      path,
      z.object({
        emergencyName: z
          .string()
          .check(
            z.trim(),
            z.minLength(1, t('participantForm.errors.emergencyName')),
            z.maxLength(120, t('crm.errors.tooLong')),
          ),
        emergencyPhone: z
          .string()
          .check(z.refine((v) => PHONE.test(v.trim()), t('participantForm.errors.phone'))),
        experience: z.string().check(z.minLength(1, t('participantForm.errors.experience'))),
        health: z.string().check(z.maxLength(2000, t('crm.errors.tooLong'))),
        notes: z.string().check(z.maxLength(2000, t('crm.errors.tooLong'))),
      }),
    );
  });

  protected id(name: string): string {
    return `${this.uid}-${name}`;
  }

  protected setExperience(level: MountainExperience): void {
    this.model.update((m) => ({ ...m, experience: level }));
  }

  protected async save(event: Event): Promise<void> {
    event.preventDefault();
    this.waiverMissing.set(this.waiver() && !this.accepted());
    await submit(this.pf, async () => {
      if (this.waiverMissing()) return undefined;
      const v = this.model();
      this.saved.emit({
        form: {
          emergencyName: v.emergencyName.trim(),
          emergencyPhone: v.emergencyPhone.trim(),
          experience: v.experience as MountainExperience,
          health: v.health.trim() || null,
          notes: v.notes.trim() || null,
        },
        acceptWaiver: this.waiver() && this.accepted(),
      });
      return undefined;
    });
  }
}
