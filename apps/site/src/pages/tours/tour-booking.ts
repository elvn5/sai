import type { Order, Tour } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { injectLang } from '@/core/i18n/current-lang';
import { injectCurrentUrl } from '@/core/lib/current-url';
import { formatAmount, todayInBishkek } from '@/core/lib/format';
import { FieldErrors } from '@/ui/field-errors';
import { SiteLink } from '@/ui/site-link';
import { DOCUMENT } from '@angular/common';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  inject,
  input,
  signal,
  untracked,
} from '@angular/core';
import { FormField, form, maxLength, submit, validateStandardSchema } from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';
import { z } from 'zod/mini';

const MAX_PLACES = 20;

interface BookingForm {
  contactName: string;
  contactPhone: string;
  comment: string;
}

// Books places on a tour and pays online: the server prices the order and
// the visitor goes on to the payment page. Shown while the tour is ahead,
// has a price and an owner, and to anyone but that owner.
@Component({
  selector: 'app-tour-booking',
  // A box of its own: deferred blocks observe their root element.
  host: { class: 'block' },
  imports: [FieldErrors, FormField, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (bookable()) {
      <section class="rounded-card border-2 border-brand-500/40 bg-surface p-4" aria-labelledby="booking-heading">
        <h2 id="booking-heading" class="flex items-center gap-2 font-bold text-text">
          <span class="icon-[mdi--ticket-confirmation-outline] size-5 text-brand-600 dark:text-brand-300" aria-hidden="true"></span>{{ 'booking.title' | transloco }}
        </h2>
        @if (auth.user()) {
          <form class="mt-3 flex flex-col gap-3" novalidate (submit)="book($event)">
            <div class="flex items-center justify-between gap-3">
              <span id="places-label" class="text-sm text-text">{{ 'booking.places' | transloco }}</span>
              <div class="flex items-center gap-2" role="group" aria-labelledby="places-label">
                <button type="button" class="ui-btn-soft ui-btn-xs" [disabled]="quantity() <= 1" [attr.aria-label]="'booking.less' | transloco" (click)="quantity.set(quantity() - 1)">
                  <span class="icon-[heroicons--minus] size-4" aria-hidden="true"></span>
                </button>
                <span class="w-6 text-center font-bold text-text" aria-live="polite">{{ quantity() }}</span>
                <button type="button" class="ui-btn-soft ui-btn-xs" [disabled]="quantity() >= maxPlaces" [attr.aria-label]="'booking.more' | transloco" (click)="quantity.set(quantity() + 1)">
                  <span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>
                </button>
              </div>
            </div>
            <label class="ui-field">
              <span class="sr-only">{{ 'booking.name' | transloco }}</span>
              <input class="ui-input" autocomplete="name" [placeholder]="'booking.name' | transloco" [formField]="bookingForm.contactName" />
              <app-field-errors [state]="bookingForm.contactName()" />
            </label>
            <label class="ui-field">
              <span class="sr-only">{{ 'booking.phone' | transloco }}</span>
              <input class="ui-input" type="tel" autocomplete="tel" [placeholder]="'booking.phone' | transloco" [formField]="bookingForm.contactPhone" />
              <app-field-errors [state]="bookingForm.contactPhone()" />
            </label>
            <label class="ui-field">
              <span class="sr-only">{{ 'booking.comment' | transloco }}</span>
              <textarea class="ui-input" rows="2" [placeholder]="'booking.comment' | transloco" [formField]="bookingForm.comment"></textarea>
            </label>
            <div class="flex items-baseline justify-between border-t border-border pt-3">
              <span class="text-sm text-text-muted">{{ 'booking.total' | transloco }}</span>
              <span class="text-xl font-black text-text">{{ total() }} {{ 'listing.currency' | transloco }}</span>
            </div>
            @if (error(); as key) {
              <p class="ui-error" role="alert">{{ key | transloco }}</p>
            }
            <button type="submit" class="ui-btn-primary py-2.5 text-base" [disabled]="bookingForm().submitting()">
              <span class="icon-[mdi--credit-card-outline] size-5" aria-hidden="true"></span>{{ 'booking.pay' | transloco }}
            </button>
            <p class="flex items-center gap-1 text-xs text-text-muted">
              <span class="icon-[mdi--shield-check-outline] size-4" aria-hidden="true"></span>{{ 'booking.secure' | transloco }}
            </p>
          </form>
        } @else {
          <div class="mt-3 flex flex-col gap-2">
            <p class="text-sm text-text-muted">{{ 'booking.signIn' | transloco }}</p>
            <a class="ui-btn-primary justify-center py-2" [appSiteLink]="loginLink()">{{ 'booking.signInCta' | transloco }}</a>
          </div>
        }
      </section>
    }
  `,
})
export class TourBooking {
  readonly tour = input.required<Tour>();

  protected readonly auth = inject(AuthStore);
  private readonly http = inject(HttpClient);
  private readonly transloco = inject(TranslocoService);
  private readonly location = inject(DOCUMENT).location;
  private readonly lang = injectLang();
  private readonly url = injectCurrentUrl();
  protected readonly maxPlaces = MAX_PLACES;
  protected readonly quantity = signal(1);
  protected readonly error = signal<string | null>(null);

  protected readonly bookable = computed(() => {
    const tour = this.tour();
    return (
      Boolean(tour.price && tour.price > 0 && tour.userId) &&
      tour.startDate >= todayInBishkek() &&
      tour.userId !== this.auth.user()?.id
    );
  });
  protected readonly total = computed(() =>
    formatAmount(this.lang(), (this.tour().price ?? 0) * this.quantity()),
  );
  // Back here after signing in (the login page reads ?redirect).
  protected readonly loginLink = computed(
    () => `/login?redirect=${encodeURIComponent(this.url())}`,
  );

  private readonly model = signal<BookingForm>({ contactName: '', contactPhone: '', comment: '' });
  protected readonly bookingForm = form(this.model, (path) => {
    const required = this.transloco.translate('booking.checkFields');
    validateStandardSchema(
      path,
      z.object({
        contactName: z.string().check(z.trim(), z.minLength(1, required)),
        contactPhone: z.string().check(z.trim(), z.minLength(1, required)),
      }),
    );
    maxLength(path.contactName, 120);
    maxLength(path.contactPhone, 32);
    maxLength(path.comment, 1000);
  });

  constructor() {
    // The account's name and phone, once known, unless typed already.
    effect(() => {
      const user = this.auth.user();
      if (!user) return;
      untracked(() =>
        this.model.update((value) => ({
          ...value,
          contactName: value.contactName || user.firstName || '',
          contactPhone: value.contactPhone || user.phone || '',
        })),
      );
    });
  }

  protected async book(event: Event): Promise<void> {
    event.preventDefault();
    this.error.set(null);
    await submit(this.bookingForm, async () => {
      const { contactName, contactPhone, comment } = this.model();
      try {
        const order = await firstValueFrom(
          this.http.post<Order>('/orders', {
            kind: 'tour',
            itemId: this.tour().id,
            quantity: this.quantity(),
            contactName: contactName.trim(),
            contactPhone: contactPhone.trim(),
            comment: comment.trim() || null,
          }),
        );
        if (order.paymentUrl) this.location.assign(order.paymentUrl);
      } catch (error) {
        const status = error instanceof HttpErrorResponse ? error.status : 0;
        this.error.set(
          status === 400
            ? 'booking.checkFields'
            : status === 502
              ? 'booking.gatewayDown'
              : status === 409
                ? 'booking.unavailable'
                : 'booking.error',
        );
      }
      return undefined;
    });
  }
}
