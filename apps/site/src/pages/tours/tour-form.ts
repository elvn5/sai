import { CancellationPolicy, type Destination, type Tour } from '@/core/api/types';
import {
  PHONE_PATTERN,
  TELEGRAM_PATTERN,
  emptyToNull,
  optionalPattern,
  requiredHtml,
  telegramUsername,
} from '@/core/lib/listing-form';
import { CANCELLATION_OPTIONS, DESTINATION_OPTIONS } from '@/core/lib/listing-options';
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
  maxLength,
  min,
  submit,
  validate,
  validateStandardSchema,
} from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { z } from 'zod/mini';

// What the API takes to create or update a tour.
export interface TourPayload {
  title: string;
  excerpt: string | null;
  content: string;
  price: number | null;
  durationDays: number | null;
  location: string | null;
  startDate: string;
  endDate: string;
  destination: Destination | null;
  cancellationPolicy: CancellationPolicy;
  coverImageUrl: string | null;
  phone: string | null;
  whatsapp: string | null;
  telegram: string | null;
}

interface TourFormModel {
  title: string;
  excerpt: string;
  price: number | null;
  durationDays: number | null;
  location: string;
  startDate: string;
  endDate: string;
  destination: string;
  cancellationPolicy: string;
  cover: string[];
  content: string;
  whatsapp: string;
  telegram: string;
  phone: string;
}

function toModel(tour: Partial<Tour> | undefined): TourFormModel {
  return {
    title: tour?.title ?? '',
    excerpt: tour?.excerpt ?? '',
    price: tour?.price ?? null,
    durationDays: tour?.durationDays ?? null,
    location: tour?.location ?? '',
    startDate: tour?.startDate ?? '',
    endDate: tour?.endDate ?? '',
    destination: tour?.destination ?? '',
    cancellationPolicy: tour?.cancellationPolicy ?? CancellationPolicy.FLEXIBLE,
    cover: tour?.coverImageUrl ? [tour.coverImageUrl] : [],
    content: tour?.content ?? '',
    whatsapp: tour?.whatsapp ?? '',
    telegram: tour?.telegram ?? '',
    phone: tour?.phone ?? '',
  };
}

// A tour: creates one, or prefills from `tour` (a copy passes the original
// without its dates).
@Component({
  selector: 'app-tour-form',
  imports: [FieldErrors, FormField, PhotoUploader, RichTextEditor, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <form class="flex flex-col gap-6" novalidate (submit)="send($event)">
      <label class="ui-field">
        {{ 'tours.titleLabel' | transloco }} *
        <input class="ui-input" [formField]="tourForm.title" />
        <app-field-errors [state]="tourForm.title()" />
      </label>
      <label class="ui-field">
        {{ 'tours.excerptLabel' | transloco }}
        <input class="ui-input" [formField]="tourForm.excerpt" />
      </label>
      <div class="grid grid-cols-1 gap-5 sm:grid-cols-3">
        <label class="ui-field">
          <span>{{ 'tours.priceLabel' | transloco }} <span class="text-text-muted">({{ 'listing.currency' | transloco }})</span></span>
          <input class="ui-input" type="number" inputmode="numeric" [formField]="tourForm.price" />
          <app-field-errors [state]="tourForm.price()" />
        </label>
        <label class="ui-field">
          {{ 'tours.durationLabel' | transloco }}
          <input class="ui-input" type="number" inputmode="numeric" [formField]="tourForm.durationDays" />
          <app-field-errors [state]="tourForm.durationDays()" />
        </label>
        <label class="ui-field">
          {{ 'tours.locationLabel' | transloco }}
          <input class="ui-input" [formField]="tourForm.location" />
        </label>
      </div>
      <div class="grid grid-cols-1 gap-5 sm:grid-cols-2">
        <label class="ui-field">
          {{ 'tours.startDateLabel' | transloco }} *
          <input class="ui-input" type="date" [formField]="tourForm.startDate" />
          <app-field-errors [state]="tourForm.startDate()" />
        </label>
        <label class="ui-field">
          {{ 'tours.endDateLabel' | transloco }} *
          <input class="ui-input" type="date" [formField]="tourForm.endDate" />
          <app-field-errors [state]="tourForm.endDate()" />
        </label>
      </div>
      <div class="grid grid-cols-1 gap-5 sm:grid-cols-2">
        <label class="ui-field">
          {{ 'listing.form.destinationLabel' | transloco }}
          <select class="ui-input" [formField]="tourForm.destination">
            <option value="">{{ 'listing.form.choose' | transloco }}</option>
            @for (option of destinations; track option.value) {
              <option [value]="option.value">{{ option.labelKey | transloco }}</option>
            }
          </select>
        </label>
        <label class="ui-field">
          {{ 'listing.form.cancellationLabel' | transloco }} *
          <select class="ui-input" [formField]="tourForm.cancellationPolicy">
            @for (option of cancellations; track option.value) {
              <option [value]="option.value">{{ option.labelKey | transloco }}</option>
            }
          </select>
        </label>
      </div>
      <div class="grid grid-cols-1 gap-5 sm:grid-cols-3">
        <label class="ui-field">
          {{ 'listing.form.whatsappLabel' | transloco }}
          <input class="ui-input" type="tel" placeholder="+996 555 123 456" [formField]="tourForm.whatsapp" />
          <app-field-errors [state]="tourForm.whatsapp()" />
        </label>
        <label class="ui-field">
          {{ 'listing.form.telegramLabel' | transloco }}
          <input class="ui-input" placeholder="@username" [formField]="tourForm.telegram" />
          <app-field-errors [state]="tourForm.telegram()" />
        </label>
        <label class="ui-field">
          {{ 'listing.form.phoneLabel' | transloco }}
          <input class="ui-input" type="tel" placeholder="+996 312 000 000" [formField]="tourForm.phone" />
          <app-field-errors [state]="tourForm.phone()" />
        </label>
      </div>
      <div class="ui-field">
        <span>{{ 'tours.coverLabel' | transloco }}</span>
        <div class="w-40"><app-photo-uploader folder="tours" [limit]="1" [formField]="tourForm.cover" /></div>
      </div>
      <div class="ui-field">
        <span>{{ 'tours.contentLabel' | transloco }} *</span>
        <app-rich-text-editor [placeholder]="'tours.contentPlaceholder' | transloco" [formField]="tourForm.content" />
        <app-field-errors [state]="tourForm.content()" />
      </div>

      @if (errorKey(); as key) {
        <p class="ui-error" role="alert">{{ key | transloco }}</p>
      }
      <button type="submit" class="ui-btn-primary py-2.5 text-base" [disabled]="tourForm().submitting()">{{ submitLabel() }}</button>
    </form>
  `,
})
export class TourForm {
  readonly tour = input<Partial<Tour> | undefined>(undefined);
  readonly submitLabel = input.required<string>();
  // Sends the tour; rejects with a translation key to show.
  readonly save = input.required<(payload: TourPayload) => Promise<void>>();

  private readonly transloco = inject(TranslocoService);
  protected readonly destinations = DESTINATION_OPTIONS;
  protected readonly cancellations = CANCELLATION_OPTIONS;
  protected readonly errorKey = signal<string | null>(null);

  private readonly model = linkedSignal(() => toModel(this.tour()));
  protected readonly tourForm = form(this.model, (path) => {
    const t = (key: string): string => this.transloco.translate(key);
    validateStandardSchema(
      path,
      z.object({
        title: z.string().check(z.trim(), z.minLength(1, t('login.required'))),
        startDate: z.string().check(z.minLength(1, t('login.required'))),
        endDate: z.string().check(z.minLength(1, t('login.required'))),
        content: requiredHtml(t('login.required')),
        whatsapp: optionalPattern(PHONE_PATTERN, t('listing.form.invalidPhone')),
        phone: optionalPattern(PHONE_PATTERN, t('listing.form.invalidPhone')),
        telegram: optionalPattern(TELEGRAM_PATTERN, t('listing.form.invalidTelegram')),
      }),
    );
    validate(path.endDate, (field) => {
      const start = field.valueOf(path.startDate);
      const end = field.value();
      return start && end && end < start
        ? { kind: 'dateRange', message: t('tours.dateRangeInvalid') }
        : undefined;
    });
    maxLength(path.title, 255);
    maxLength(path.excerpt, 500);
    maxLength(path.location, 255);
    maxLength(path.content, 20000);
    min(path.price, 0);
    min(path.durationDays, 1);
  });

  protected async send(event: Event): Promise<void> {
    event.preventDefault();
    this.errorKey.set(null);
    await submit(this.tourForm, async () => {
      const value = this.model();
      try {
        await this.save()({
          title: value.title.trim(),
          excerpt: emptyToNull(value.excerpt),
          content: value.content,
          price: value.price,
          durationDays: value.durationDays,
          location: emptyToNull(value.location),
          startDate: value.startDate,
          endDate: value.endDate,
          destination: (value.destination || null) as Destination | null,
          cancellationPolicy: value.cancellationPolicy as CancellationPolicy,
          coverImageUrl: value.cover[0] ?? null,
          phone: emptyToNull(value.phone),
          whatsapp: emptyToNull(value.whatsapp),
          telegram: telegramUsername(value.telegram),
        });
      } catch (error) {
        this.errorKey.set(typeof error === 'string' ? error : 'tours.createError');
      }
      return undefined;
    });
  }
}
