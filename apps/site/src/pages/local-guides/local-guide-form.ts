import type { Destination, LocalGuide } from '@/core/api/types';
import {
  PHONE_PATTERN,
  TELEGRAM_PATTERN,
  emptyToNull,
  optionalPattern,
  requiredHtml,
  telegramUsername,
} from '@/core/lib/listing-form';
import {
  DESTINATION_OPTIONS,
  GUIDE_LANGUAGE_OPTIONS,
  GUIDE_SPECIALIZATION_OPTIONS,
} from '@/core/lib/listing-options';
import { ChipToggleGroup } from '@/ui/chip-toggle-group';
import { FieldErrors } from '@/ui/field-errors';
import { PhotoUploader } from '@/ui/photo-uploader';
import { RichTextEditor } from '@/ui/rich-text-editor';
import {
  ChangeDetectionStrategy,
  Component,
  inject,
  input,
  linkedSignal,
  signal,
} from '@angular/core';
import {
  FormField,
  form,
  max,
  maxLength,
  min,
  submit,
  validateStandardSchema,
} from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { z } from 'zod/mini';

const MAX_PHOTOS = 20;

// What the API takes to create or update a guide profile.
export interface LocalGuidePayload {
  name: string;
  headline: string | null;
  bio: string;
  avatarUrl: string | null;
  coverImageUrl: string | null;
  photos: string[];
  languages: string[];
  specializations: string[];
  destinations: Destination[];
  pricePerDay: number | null;
  experienceYears: number | null;
  hasVehicle: boolean;
  phone: string | null;
  whatsapp: string | null;
  telegram: string | null;
}

interface GuideFormModel {
  avatar: string[];
  name: string;
  headline: string;
  pricePerDay: number | null;
  experienceYears: number | null;
  hasVehicle: boolean;
  languages: string[];
  specializations: string[];
  destinations: string[];
  bio: string;
  photos: string[];
  whatsapp: string;
  telegram: string;
  phone: string;
}

function toModel(guide: LocalGuide | undefined, defaultName: string): GuideFormModel {
  return {
    avatar: guide?.avatarUrl ? [guide.avatarUrl] : [],
    name: guide?.name ?? defaultName,
    headline: guide?.headline ?? '',
    pricePerDay: guide?.pricePerDay ?? null,
    experienceYears: guide?.experienceYears ?? null,
    hasVehicle: guide?.hasVehicle ?? false,
    languages: guide?.languages ?? [],
    specializations: guide?.specializations ?? [],
    destinations: guide?.destinations ?? [],
    bio: guide?.bio ?? '',
    photos: guide?.photos ?? [],
    whatsapp: guide?.whatsapp ?? '',
    telegram: guide?.telegram ?? '',
    phone: guide?.phone ?? '',
  };
}

// A guide's profile: creates it, or edits `guide`.
@Component({
  selector: 'app-local-guide-form',
  imports: [ChipToggleGroup, FieldErrors, FormField, PhotoUploader, RichTextEditor, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <form class="flex flex-col gap-8" novalidate (submit)="send($event)">
      <fieldset class="flex flex-col gap-5">
        <legend class="mb-3 text-lg font-semibold text-text">{{ 'listing.form.basics' | transloco }}</legend>
        <div class="ui-field">
          <span>{{ 'localGuides.avatarLabel' | transloco }}</span>
          <div class="w-32"><app-photo-uploader folder="guides" [limit]="1" [formField]="guideForm.avatar" /></div>
        </div>
        <label class="ui-field">
          {{ 'localGuides.nameLabel' | transloco }} *
          <input class="ui-input" [formField]="guideForm.name" />
          <app-field-errors [state]="guideForm.name()" />
        </label>
        <label class="ui-field">
          {{ 'localGuides.headlineLabel' | transloco }}
          <input class="ui-input" [placeholder]="'localGuides.headlinePlaceholder' | transloco" [formField]="guideForm.headline" />
        </label>
        <div class="grid grid-cols-1 gap-5 sm:grid-cols-2">
          <label class="ui-field">
            <span>{{ 'localGuides.priceLabel' | transloco }} <span class="text-text-muted">({{ 'listing.currency' | transloco }})</span></span>
            <input class="ui-input" type="number" inputmode="numeric" [formField]="guideForm.pricePerDay" />
            <app-field-errors [state]="guideForm.pricePerDay()" />
          </label>
          <label class="ui-field">
            {{ 'localGuides.experienceLabel' | transloco }}
            <input class="ui-input" type="number" inputmode="numeric" [formField]="guideForm.experienceYears" />
            <app-field-errors [state]="guideForm.experienceYears()" />
          </label>
        </div>
        <label class="flex items-center gap-2 text-sm text-text">
          <input type="checkbox" class="size-4 accent-brand-600" [formField]="guideForm.hasVehicle" />
          {{ 'localGuides.hasVehicleLabel' | transloco }}
        </label>
      </fieldset>

      <fieldset class="flex flex-col gap-5">
        <legend class="mb-3 text-lg font-semibold text-text">{{ 'localGuides.expertise' | transloco }}</legend>
        <div class="ui-field">
          <span>{{ 'localGuides.languagesLabel' | transloco }} *</span>
          <app-chip-toggle-group [options]="languageOptions" [label]="'localGuides.languagesLabel' | transloco" [formField]="guideForm.languages" />
          <app-field-errors [state]="guideForm.languages()" />
        </div>
        <div class="ui-field">
          <span>{{ 'localGuides.specializationsLabel' | transloco }} *</span>
          <app-chip-toggle-group [options]="specializationOptions" [label]="'localGuides.specializationsLabel' | transloco" [formField]="guideForm.specializations" />
          <app-field-errors [state]="guideForm.specializations()" />
        </div>
        <div class="ui-field">
          <span>{{ 'localGuides.destinationsLabel' | transloco }} *</span>
          <app-chip-toggle-group [options]="destinationOptions" [label]="'localGuides.destinationsLabel' | transloco" [formField]="guideForm.destinations" />
          <app-field-errors [state]="guideForm.destinations()" />
        </div>
      </fieldset>

      <fieldset class="flex flex-col gap-3">
        <legend class="mb-1 text-lg font-semibold text-text">{{ 'localGuides.bioLabel' | transloco }} *</legend>
        <app-rich-text-editor [placeholder]="'localGuides.bioPlaceholder' | transloco" [formField]="guideForm.bio" />
        <app-field-errors [state]="guideForm.bio()" />
      </fieldset>

      <fieldset class="flex flex-col gap-3">
        <legend class="mb-1 text-lg font-semibold text-text">{{ 'listing.photos.heading' | transloco }}</legend>
        <p class="text-sm text-text-muted">{{ 'localGuides.photosHint' | transloco }}</p>
        <app-photo-uploader folder="guides" [limit]="maxPhotos" [coverBadge]="true" [formField]="guideForm.photos" />
      </fieldset>

      <fieldset class="flex flex-col gap-5">
        <legend class="mb-1 text-lg font-semibold text-text">{{ 'listing.form.contacts' | transloco }}</legend>
        <p class="-mt-2 text-sm text-text-muted">{{ 'listing.form.contactsHint' | transloco }}</p>
        <div class="grid grid-cols-1 gap-5 sm:grid-cols-2">
          <label class="ui-field">
            {{ 'listing.form.whatsappLabel' | transloco }}
            <input class="ui-input" type="tel" placeholder="+996 555 123 456" [formField]="guideForm.whatsapp" />
            <app-field-errors [state]="guideForm.whatsapp()" />
          </label>
          <label class="ui-field">
            {{ 'listing.form.telegramLabel' | transloco }}
            <input class="ui-input" placeholder="@username" [formField]="guideForm.telegram" />
            <app-field-errors [state]="guideForm.telegram()" />
          </label>
          <label class="ui-field">
            {{ 'listing.form.phoneLabel' | transloco }}
            <input class="ui-input" type="tel" placeholder="+996 312 000 000" [formField]="guideForm.phone" />
            <app-field-errors [state]="guideForm.phone()" />
          </label>
        </div>
      </fieldset>

      @if (failed()) {
        <p class="ui-error" role="alert">{{ 'listing.form.saveError' | transloco }}</p>
      }
      <button type="submit" class="ui-btn-primary py-2.5 text-base" [disabled]="guideForm().submitting()">{{ submitLabel() }}</button>
    </form>
  `,
})
export class LocalGuideForm {
  readonly guide = input<LocalGuide | undefined>(undefined);
  // The account's name, for a new profile.
  readonly defaultName = input('');
  readonly submitLabel = input.required<string>();
  readonly save = input.required<(payload: LocalGuidePayload) => Promise<void>>();

  private readonly transloco = inject(TranslocoService);
  protected readonly languageOptions = GUIDE_LANGUAGE_OPTIONS;
  protected readonly specializationOptions = GUIDE_SPECIALIZATION_OPTIONS;
  protected readonly destinationOptions = DESTINATION_OPTIONS;
  protected readonly maxPhotos = MAX_PHOTOS;
  protected readonly failed = signal(false);

  private readonly model = linkedSignal(() => toModel(this.guide(), this.defaultName()));
  protected readonly guideForm = form(this.model, (path) => {
    const t = (key: string): string => this.transloco.translate(key);
    const pickOne = z.array(z.string()).check(z.minLength(1, t('listing.form.pickAtLeastOne')));
    validateStandardSchema(
      path,
      z.object({
        name: z.string().check(z.trim(), z.minLength(1, t('login.required'))),
        bio: requiredHtml(t('login.required')),
        languages: pickOne,
        specializations: pickOne,
        destinations: pickOne,
        whatsapp: optionalPattern(PHONE_PATTERN, t('listing.form.invalidPhone')),
        phone: optionalPattern(PHONE_PATTERN, t('listing.form.invalidPhone')),
        telegram: optionalPattern(TELEGRAM_PATTERN, t('listing.form.invalidTelegram')),
      }),
    );
    maxLength(path.name, 255);
    maxLength(path.headline, 160);
    maxLength(path.bio, 20000);
    min(path.pricePerDay, 0);
    min(path.experienceYears, 0);
    max(path.experienceYears, 80);
  });

  protected async send(event: Event): Promise<void> {
    event.preventDefault();
    this.failed.set(false);
    await submit(this.guideForm, async () => {
      const value = this.model();
      try {
        await this.save()({
          name: value.name.trim(),
          headline: emptyToNull(value.headline),
          bio: value.bio,
          avatarUrl: value.avatar[0] ?? null,
          coverImageUrl: value.photos[0] ?? null,
          photos: value.photos,
          languages: value.languages,
          specializations: value.specializations,
          destinations: value.destinations as Destination[],
          pricePerDay: value.pricePerDay,
          experienceYears: value.experienceYears,
          hasVehicle: value.hasVehicle,
          phone: emptyToNull(value.phone),
          whatsapp: emptyToNull(value.whatsapp),
          telegram: telegramUsername(value.telegram),
        });
      } catch {
        this.failed.set(true);
      }
      return undefined;
    });
  }
}
