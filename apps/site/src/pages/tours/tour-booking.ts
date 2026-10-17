import type { Order, Tour } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { injectLang } from '@/core/i18n/current-lang';
import { injectCurrentUrl } from '@/core/lib/current-url';
import { formatAmount, todayInBishkek } from '@/core/lib/format';
import { loaded } from '@/core/lib/resource';
import { FieldErrors } from '@/ui/field-errors';
import { SiteLink } from '@/ui/site-link';
import { DOCUMENT } from '@angular/common';
import { HttpClient, HttpErrorResponse, httpResource } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  computed,
  effect,
  inject,
  input,
  signal,
  untracked,
  viewChild,
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
// the visitor goes on to the payment page. Shown while online payment is on
// (/payments/config), the tour is ahead, has a price and an owner, and to
// anyone but that owner.
@Component({
  selector: 'app-tour-booking',
  // A box of its own: deferred blocks observe their root element.
  host: { class: 'block' },
  imports: [FieldErrors, FormField, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  // Below lg the form is a bottom sheet: it slides up when opened
  // (@starting-style) and back down when closed (display transitions).
  styles: `
    @media (width < 64rem) {
      .sheet {
        position: fixed;
        inset: auto 0 0;
        /* Above the cookie notice (z-50). */
        z-index: 60;
        max-height: 90dvh;
        overflow-y: auto;
        border-bottom-width: 0;
        border-radius: 1.25rem 1.25rem 0 0;
        box-shadow: 0 -12px 40px rgb(0 0 0 / 0.25);
        padding-bottom: max(1rem, env(safe-area-inset-bottom));
        transition:
          transform var(--dur-slow) var(--ease-enter),
          display var(--dur-slow) allow-discrete;
      }
      .sheet:not([data-open]) {
        display: none;
        transform: translateY(100%);
        transition-duration: var(--dur-base);
        transition-timing-function: var(--ease-exit);
      }
      .sheet-backdrop {
        position: fixed;
        inset: 0;
        z-index: 59;
        background: rgb(0 0 0 / 0.5);
        transition:
          opacity var(--dur-slow) var(--ease-standard),
          display var(--dur-slow) allow-discrete;
      }
      .sheet-backdrop:not([data-open]) {
        display: none;
        opacity: 0;
      }
      @starting-style {
        .sheet[data-open] {
          transform: translateY(100%);
        }
        .sheet-backdrop[data-open] {
          opacity: 0;
        }
      }
    }
    @media (width >= 64rem) {
      .sheet-backdrop {
        display: none;
      }
    }
    @media (prefers-reduced-motion: reduce) {
      .sheet,
      .sheet-backdrop {
        transition: none;
      }
    }
  `,
  template: `
    @if (bookable()) {
      <!-- Phones: a compact card that opens the form as a bottom sheet. -->
      <div class="flex items-center justify-between gap-3 rounded-card border-2 border-brand-500/40 bg-surface p-4 lg:hidden">
        <div>
          <p class="font-bold text-text">{{ 'booking.title' | transloco }}</p>
          <p class="text-sm text-text-muted">{{ 'booking.perPerson' | transloco: { price: price(), currency: ('listing.currency' | transloco) } }}</p>
        </div>
        <button type="button" class="ui-btn-primary shrink-0 px-4 py-2.5" aria-controls="booking-sheet" [attr.aria-expanded]="sheetOpen()" (click)="openSheet()">
          <span class="icon-[mdi--ticket-confirmation-outline] size-5" aria-hidden="true"></span>{{ 'booking.open' | transloco }}
        </button>
      </div>
      <div class="sheet-backdrop" [attr.data-open]="sheetOpen() ? '' : null" aria-hidden="true" (click)="sheetOpen.set(false)"></div>
      <section
        #sheet
        id="booking-sheet"
        class="sheet rounded-card border-2 border-brand-500/40 bg-surface p-4"
        [attr.data-open]="sheetOpen() ? '' : null"
        aria-labelledby="booking-heading"
        tabindex="-1"
        (keydown.escape)="sheetOpen.set(false)"
      >
        <span class="mx-auto -mt-1 mb-3 block h-1.5 w-10 rounded-full bg-text/15 lg:hidden" aria-hidden="true"></span>
        <div class="flex items-center justify-between gap-2">
          <h2 id="booking-heading" class="flex items-center gap-2 font-bold text-text">
            <span class="icon-[mdi--ticket-confirmation-outline] size-5 text-brand-600 dark:text-brand-300" aria-hidden="true"></span>{{ 'booking.title' | transloco }}
          </h2>
          <button type="button" class="ui-btn-ghost ui-btn-xs lg:hidden" [attr.aria-label]="'booking.close' | transloco" (click)="sheetOpen.set(false)">
            <span class="icon-[heroicons--x-mark] size-5" aria-hidden="true"></span>
          </button>
        </div>
        @if (auth.user()) {
          <form class="mt-3 flex flex-col gap-3" novalidate (submit)="book($event)">
            <div class="flex items-center justify-between gap-3">
              <span id="places-label" class="text-sm text-text">{{ 'booking.places' | transloco }}</span>
              <div class="flex items-center gap-2" role="group" aria-labelledby="places-label">
                <button type="button" class="ui-btn-soft ui-btn-xs" [disabled]="quantity() <= 1" [attr.aria-label]="'booking.less' | transloco" (click)="quantity.set(quantity() - 1)">
                  <span class="icon-[heroicons--minus] size-4" aria-hidden="true"></span>
                </button>
                <span class="w-6 overflow-hidden text-center font-bold text-text" aria-live="polite">
                  @for (value of [quantity()]; track value) {
                    <span animate.enter="motion-roll-in">{{ value }}</span>
                  }
                </span>
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
              <span class="overflow-hidden text-xl font-black text-text">
                @for (value of [total()]; track value) {
                  <span animate.enter="motion-roll-in">{{ value }}</span>
                }
                {{ 'listing.currency' | transloco }}
              </span>
            </div>
            @if (error(); as key) {
              <p class="ui-error" role="alert">{{ key | transloco }}</p>
            }
            <p class="text-xs text-text-muted">
              {{ 'legal.consent.booking' | transloco }}
              <a appSiteLink="/offer" class="underline hover:text-text">{{ 'legal.consent.offer' | transloco }}</a>
              {{ 'legal.consent.and' | transloco }}
              <a appSiteLink="/refunds" class="underline hover:text-text">{{ 'legal.consent.refunds' | transloco }}</a>.
            </p>
            <!-- Four states: ready, preparing the order, on the way to the
                 payment page (a check draws itself), refused (a shake). -->
            <button
              type="submit"
              class="ui-btn-primary py-2.5 text-base transition-[background-color,scale] active:scale-[0.98] motion-reduce:active:scale-100"
              [class.motion-shake]="payState() === 'error'"
              [class.bg-green-600]="payState() === 'success'"
              [class.hover:bg-green-600]="payState() === 'success'"
              [disabled]="payState() === 'loading' || payState() === 'success'"
              [attr.aria-busy]="payState() === 'loading'"
              (animationend)="payState() === 'error' && payState.set('idle')"
            >
              @switch (payState()) {
                @case ('loading') {
                  <span class="size-5 animate-spin rounded-full border-2 border-white/40 border-t-white" aria-hidden="true"></span>{{ 'booking.preparing' | transloco }}
                }
                @case ('success') {
                  <svg class="size-5" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                    <path class="motion-draw" pathLength="1" d="M5 12.5l4.5 4.5L19 7.5" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" />
                  </svg>{{ 'booking.redirecting' | transloco }}
                }
                @default {
                  <span class="icon-[mdi--credit-card-outline] size-5" aria-hidden="true"></span>{{ 'booking.pay' | transloco }}
                }
              }
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
  protected readonly payState = signal<'idle' | 'loading' | 'success' | 'error'>('idle');
  // The bottom sheet on phones (the form is always open on wider screens).
  protected readonly sheetOpen = signal(false);
  private readonly sheet = viewChild<ElementRef<HTMLElement>>('sheet');

  // Off until the payments service says otherwise (PAYMENTS_PROVIDER=off).
  private readonly payments = httpResource<{ online: boolean }>(() => '/payments/config');

  protected readonly bookable = computed(() => {
    const tour = this.tour();
    return (
      loaded(this.payments)?.online === true &&
      Boolean(tour.price && tour.price > 0 && tour.userId) &&
      tour.startDate >= todayInBishkek() &&
      tour.userId !== this.auth.user()?.id
    );
  });
  protected readonly price = computed(() => formatAmount(this.lang(), this.tour().price ?? 0));
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

  protected openSheet(): void {
    this.sheetOpen.set(true);
    // Into the sheet, for keyboards and screen readers.
    setTimeout(() => this.sheet()?.nativeElement.focus());
  }

  protected async book(event: Event): Promise<void> {
    event.preventDefault();
    this.error.set(null);
    const valid = await submit(this.bookingForm, async () => {
      this.payState.set('loading');
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
        if (order.paymentUrl) {
          this.payState.set('success');
          // Long enough to see the check mark.
          const url = order.paymentUrl;
          setTimeout(() => this.location.assign(url), 450);
        } else {
          this.payState.set('idle');
        }
      } catch (error) {
        this.payState.set('error');
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
    if (!valid) this.payState.set('error');
  }
}
