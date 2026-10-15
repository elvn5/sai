import { CancellationPolicy, Destination, type Hotel, HotelType } from '@/core/api/types';
import {
  PHONE_PATTERN,
  TELEGRAM_PATTERN,
  emptyToNull,
  optionalPattern,
  optionalUrl,
  requiredHtml,
  telegramUsername,
} from '@/core/lib/listing-form';
import {
  CANCELLATION_OPTIONS,
  DESTINATION_OPTIONS,
  HOTEL_AMENITY_OPTIONS,
  HOTEL_TYPE_OPTIONS,
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

// What the API takes to create or update a hotel.
export interface HotelPayload {
  name: string;
  type: HotelType;
  description: string;
  destination: Destination;
  address: string | null;
  latitude: number | null;
  longitude: number | null;
  priceFrom: number | null;
  stars: number | null;
  amenities: string[];
  cancellationPolicy: CancellationPolicy;
  checkInTime: string | null;
  checkOutTime: string | null;
  coverImageUrl: string | null;
  photos: string[];
  phone: string | null;
  whatsapp: string | null;
  telegram: string | null;
  website: string | null;
}

interface HotelFormModel {
  name: string;
  type: string;
  stars: string;
  priceFrom: number | null;
  photos: string[];
  destination: string;
  address: string;
  latitude: number | null;
  longitude: number | null;
  amenities: string[];
  cancellationPolicy: string;
  checkInTime: string;
  checkOutTime: string;
  description: string;
  whatsapp: string;
  telegram: string;
  phone: string;
  website: string;
}

function toModel(hotel: Hotel | undefined): HotelFormModel {
  return {
    name: hotel?.name ?? '',
    type: hotel?.type ?? HotelType.GUESTHOUSE,
    stars: hotel?.stars ? String(hotel.stars) : '',
    priceFrom: hotel?.priceFrom ?? null,
    photos: hotel?.photos ?? [],
    destination: hotel?.destination ?? '',
    address: hotel?.address ?? '',
    latitude: hotel?.latitude ?? null,
    longitude: hotel?.longitude ?? null,
    amenities: hotel?.amenities ?? [],
    cancellationPolicy: hotel?.cancellationPolicy ?? CancellationPolicy.FLEXIBLE,
    checkInTime: hotel?.checkInTime ?? '14:00',
    checkOutTime: hotel?.checkOutTime ?? '12:00',
    description: hotel?.description ?? '',
    whatsapp: hotel?.whatsapp ?? '',
    telegram: hotel?.telegram ?? '',
    phone: hotel?.phone ?? '',
    website: hotel?.website ?? '',
  };
}

// A partner's hotel: creates one, or edits `hotel`. `save` sends it and
// moves on; a failure shows the save error.
@Component({
  selector: 'app-hotel-form',
  imports: [ChipToggleGroup, FieldErrors, FormField, PhotoUploader, RichTextEditor, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <form class="flex flex-col gap-8" novalidate (submit)="send($event)">
      <fieldset class="flex flex-col gap-5">
        <legend class="mb-3 text-lg font-semibold text-text">{{ 'listing.form.basics' | transloco }}</legend>
        <label class="ui-field">
          {{ 'hotels.nameLabel' | transloco }} *
          <input class="ui-input" [formField]="hotelForm.name" />
          <app-field-errors [state]="hotelForm.name()" />
        </label>
        <div class="grid grid-cols-1 gap-5 sm:grid-cols-3">
          <label class="ui-field">
            {{ 'hotels.typeLabel' | transloco }} *
            <select class="ui-input" [formField]="hotelForm.type">
              @for (option of types; track option.value) {
                <option [value]="option.value">{{ option.labelKey | transloco }}</option>
              }
            </select>
          </label>
          <label class="ui-field">
            {{ 'hotels.starsLabel' | transloco }}
            <select class="ui-input" [formField]="hotelForm.stars">
              <option value="">{{ 'hotels.noStars' | transloco }}</option>
              @for (n of starCounts; track n) {
                <option [value]="'' + n">{{ starsText(n) }}</option>
              }
            </select>
          </label>
          <label class="ui-field">
            <span>{{ 'hotels.priceLabel' | transloco }} <span class="text-text-muted">({{ 'listing.currency' | transloco }})</span></span>
            <input class="ui-input" type="number" inputmode="numeric" [formField]="hotelForm.priceFrom" />
            <app-field-errors [state]="hotelForm.priceFrom()" />
          </label>
        </div>
      </fieldset>

      <fieldset class="flex flex-col gap-3">
        <legend class="mb-1 text-lg font-semibold text-text">{{ 'listing.photos.heading' | transloco }}</legend>
        <p class="text-sm text-text-muted">{{ 'listing.photos.hint' | transloco }}</p>
        <app-photo-uploader folder="hotels" [limit]="maxPhotos" [coverBadge]="true" [formField]="hotelForm.photos" />
      </fieldset>

      <fieldset class="flex flex-col gap-5">
        <legend class="mb-3 text-lg font-semibold text-text">{{ 'listing.form.location' | transloco }}</legend>
        <label class="ui-field">
          {{ 'listing.form.destinationLabel' | transloco }} *
          <select class="ui-input" [formField]="hotelForm.destination">
            <option value="" disabled>{{ 'listing.form.choose' | transloco }}</option>
            @for (option of destinations; track option.value) {
              <option [value]="option.value">{{ option.labelKey | transloco }}</option>
            }
          </select>
          <app-field-errors [state]="hotelForm.destination()" />
        </label>
        <label class="ui-field">
          {{ 'listing.form.addressLabel' | transloco }}
          <input class="ui-input" [formField]="hotelForm.address" />
        </label>
        <div class="grid grid-cols-2 gap-5">
          <label class="ui-field">
            {{ 'listing.form.latitudeLabel' | transloco }}
            <input class="ui-input" type="number" step="any" placeholder="42.4907" [formField]="hotelForm.latitude" />
            <app-field-errors [state]="hotelForm.latitude()" />
          </label>
          <label class="ui-field">
            {{ 'listing.form.longitudeLabel' | transloco }}
            <input class="ui-input" type="number" step="any" placeholder="78.3936" [formField]="hotelForm.longitude" />
            <app-field-errors [state]="hotelForm.longitude()" />
          </label>
        </div>
        <p class="-mt-2 text-xs text-text-muted">{{ 'listing.form.coordsHint' | transloco }}</p>
      </fieldset>

      <fieldset class="flex flex-col gap-5">
        <legend class="mb-3 text-lg font-semibold text-text">{{ 'hotels.conditions' | transloco }}</legend>
        <div class="ui-field">
          <span>{{ 'hotels.amenitiesLabel' | transloco }}</span>
          <app-chip-toggle-group [options]="amenityOptions" [label]="'hotels.amenitiesLabel' | transloco" [formField]="hotelForm.amenities" />
        </div>
        <fieldset class="ui-field">
          <legend class="mb-1">{{ 'listing.form.cancellationLabel' | transloco }} *</legend>
          <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
            @for (option of cancellations; track option.value) {
              <label class="flex cursor-pointer items-start gap-3 rounded-lg border p-3 transition-colors has-checked:border-brand-500 has-checked:bg-brand-500/5" [class.border-border]="model().cancellationPolicy !== option.value">
                <input type="radio" class="mt-1 accent-brand-600" [value]="option.value" [formField]="hotelForm.cancellationPolicy" />
                <span>
                  <span class="block text-sm font-medium text-text">{{ option.labelKey | transloco }}</span>
                  <span class="block text-xs text-text-muted">{{ 'listing.cancellationHint.' + option.value | transloco }}</span>
                </span>
              </label>
            }
          </div>
        </fieldset>
        <div class="grid grid-cols-2 gap-5">
          <label class="ui-field">
            {{ 'hotels.checkInLabel' | transloco }}
            <input class="ui-input" type="time" [formField]="hotelForm.checkInTime" />
          </label>
          <label class="ui-field">
            {{ 'hotels.checkOutLabel' | transloco }}
            <input class="ui-input" type="time" [formField]="hotelForm.checkOutTime" />
          </label>
        </div>
      </fieldset>

      <fieldset class="flex flex-col gap-3">
        <legend class="mb-1 text-lg font-semibold text-text">{{ 'hotels.descriptionLabel' | transloco }} *</legend>
        <app-rich-text-editor [placeholder]="'hotels.descriptionPlaceholder' | transloco" [formField]="hotelForm.description" />
        <app-field-errors [state]="hotelForm.description()" />
      </fieldset>

      <fieldset class="flex flex-col gap-5">
        <legend class="mb-1 text-lg font-semibold text-text">{{ 'listing.form.contacts' | transloco }}</legend>
        <p class="-mt-2 text-sm text-text-muted">{{ 'listing.form.contactsHint' | transloco }}</p>
        <div class="grid grid-cols-1 gap-5 sm:grid-cols-2">
          <label class="ui-field">
            {{ 'listing.form.whatsappLabel' | transloco }}
            <input class="ui-input" type="tel" placeholder="+996 555 123 456" [formField]="hotelForm.whatsapp" />
            <app-field-errors [state]="hotelForm.whatsapp()" />
          </label>
          <label class="ui-field">
            {{ 'listing.form.telegramLabel' | transloco }}
            <input class="ui-input" placeholder="@username" [formField]="hotelForm.telegram" />
            <app-field-errors [state]="hotelForm.telegram()" />
          </label>
          <label class="ui-field">
            {{ 'listing.form.phoneLabel' | transloco }}
            <input class="ui-input" type="tel" placeholder="+996 312 000 000" [formField]="hotelForm.phone" />
            <app-field-errors [state]="hotelForm.phone()" />
          </label>
          <label class="ui-field">
            {{ 'listing.form.websiteLabel' | transloco }}
            <input class="ui-input" type="url" placeholder="https://" [formField]="hotelForm.website" />
            <app-field-errors [state]="hotelForm.website()" />
          </label>
        </div>
      </fieldset>

      @if (failed()) {
        <p class="ui-error" role="alert">{{ 'listing.form.saveError' | transloco }}</p>
      }
      <button type="submit" class="ui-btn-primary py-2.5 text-base" [disabled]="hotelForm().submitting()">{{ submitLabel() }}</button>
    </form>
  `,
})
export class HotelForm {
  readonly hotel = input<Hotel | undefined>(undefined);
  readonly submitLabel = input.required<string>();
  readonly save = input.required<(payload: HotelPayload) => Promise<void>>();

  private readonly transloco = inject(TranslocoService);
  protected readonly types = HOTEL_TYPE_OPTIONS;
  protected readonly destinations = DESTINATION_OPTIONS;
  protected readonly amenityOptions = HOTEL_AMENITY_OPTIONS;
  protected readonly cancellations = CANCELLATION_OPTIONS;
  protected readonly starCounts = [1, 2, 3, 4, 5];
  protected readonly maxPhotos = MAX_PHOTOS;
  protected readonly starsText = (n: number): string => '★'.repeat(n);
  protected readonly failed = signal(false);

  protected readonly model = linkedSignal(() => toModel(this.hotel()));
  protected readonly hotelForm = form(this.model, (path) => {
    const t = (key: string): string => this.transloco.translate(key);
    validateStandardSchema(
      path,
      z.object({
        name: z.string().check(z.trim(), z.minLength(1, t('login.required'))),
        destination: z.string().check(z.minLength(1, t('login.required'))),
        description: requiredHtml(t('login.required')),
        whatsapp: optionalPattern(PHONE_PATTERN, t('listing.form.invalidPhone')),
        phone: optionalPattern(PHONE_PATTERN, t('listing.form.invalidPhone')),
        telegram: optionalPattern(TELEGRAM_PATTERN, t('listing.form.invalidTelegram')),
        website: optionalUrl(t('listing.form.invalidUrl')),
      }),
    );
    maxLength(path.name, 255);
    maxLength(path.address, 255);
    maxLength(path.description, 20000);
    min(path.priceFrom, 0);
    min(path.latitude, -90);
    max(path.latitude, 90);
    min(path.longitude, -180);
    max(path.longitude, 180);
  });

  protected async send(event: Event): Promise<void> {
    event.preventDefault();
    this.failed.set(false);
    await submit(this.hotelForm, async () => {
      const value = this.model();
      try {
        await this.save()({
          name: value.name.trim(),
          type: value.type as HotelType,
          destination: value.destination as Destination,
          description: value.description,
          address: emptyToNull(value.address),
          latitude: value.latitude,
          longitude: value.longitude,
          priceFrom: value.priceFrom,
          stars: value.stars ? Number(value.stars) : null,
          amenities: value.amenities,
          cancellationPolicy: value.cancellationPolicy as CancellationPolicy,
          checkInTime: emptyToNull(value.checkInTime),
          checkOutTime: emptyToNull(value.checkOutTime),
          coverImageUrl: value.photos[0] ?? null,
          photos: value.photos,
          phone: emptyToNull(value.phone),
          whatsapp: emptyToNull(value.whatsapp),
          telegram: telegramUsername(value.telegram),
          website: emptyToNull(value.website),
        });
      } catch {
        this.failed.set(true);
      }
      return undefined;
    });
  }
}
