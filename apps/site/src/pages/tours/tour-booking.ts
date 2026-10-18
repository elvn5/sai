import type { Order, Tour } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { injectLang } from '@/core/i18n/current-lang';
import { injectCurrentUrl } from '@/core/lib/current-url';
import { formatAmount, formatDateRange, todayInBishkek } from '@/core/lib/format';
import { PageChrome } from '@/core/lib/page-chrome';
import { loaded } from '@/core/lib/resource';
import { Notify } from '@/core/notify/notify';
import { FieldErrors } from '@/ui/field-errors';
import { SiteLink } from '@/ui/site-link';
import { DOCUMENT, isPlatformBrowser } from '@angular/common';
import { HttpClient, HttpErrorResponse, httpResource } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  ElementRef,
  PLATFORM_ID,
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
  // Below lg the form is a bottom sheet opened from a bar stuck to the
  // bottom of the screen: it slides up (@starting-style) and back down
  // (display transitions).
  styles: `
    @media (width < 64rem) {
      .sheet {
        position: fixed;
        inset: auto 0 0;
        /* Above the cookie notice (z-50). */
        z-index: 61;
        max-height: 90dvh;
        overflow-y: auto;
        border-width: 0;
        border-radius: 1.375rem 1.375rem 0 0;
        box-shadow: 0 -10px 30px rgb(0 0 0 / 0.2);
        padding-bottom: max(1.5rem, env(safe-area-inset-bottom));
        transition:
          transform var(--dur-base) var(--ease-enter),
          display var(--dur-base) allow-discrete;
      }
      .sheet:not([data-open]) {
        display: none;
        transform: translateY(100%);
        transition-duration: var(--dur-fast);
        transition-timing-function: var(--ease-exit);
      }
      .sheet-backdrop {
        position: fixed;
        inset: 0;
        z-index: 60;
        background: rgb(15 23 42 / 0.45);
        transition:
          opacity var(--dur-fast) var(--ease-standard),
          display var(--dur-fast) allow-discrete;
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
    @keyframes pay-progress {
      from {
        transform: translateX(-100%);
      }
      to {
        transform: translateX(250%);
      }
    }
    .pay-progress {
      animation: pay-progress 1.2s var(--ease-standard) infinite;
    }
    @media (prefers-reduced-motion: reduce) {
      .sheet,
      .sheet-backdrop {
        transition: none;
      }
      .pay-progress {
        animation: none;
        transform: none;
        width: 100%;
        opacity: 0.5;
      }
    }
  `,
  template: `
    @if (bookable()) {
      <!-- Phones: price and "Book" stuck to the bottom, under the thumb. -->
      <div class="fixed inset-x-0 bottom-0 z-40 border-t border-border bg-surface/94 px-4 pt-2.5 pb-[max(0.875rem,env(safe-area-inset-bottom))] backdrop-blur-md lg:hidden" animate.enter="motion-sheet-in">
        <div class="mx-auto flex max-w-xl items-center justify-between gap-3">
          <div class="min-w-0 whitespace-nowrap">
            <p class="text-base font-bold text-brand-700 dark:text-brand-300">{{ 'tours.priceValue' | transloco: { price: price() } }}</p>
            <p class="truncate text-[0.7rem] text-text-muted">{{ 'tours.perPersonShort' | transloco }} · {{ dates() }}</p>
          </div>
          <button type="button" class="ui-btn-primary shrink-0 px-3.5 py-2.5" aria-controls="booking-sheet" [attr.aria-expanded]="sheetOpen()" (click)="openSheet()">
            <span class="icon-[heroicons--ticket] size-[1.1rem]" aria-hidden="true"></span>{{ 'booking.open' | transloco }}
          </button>
        </div>
      </div>
      <div class="sheet-backdrop" [attr.data-open]="sheetOpen() ? '' : null" aria-hidden="true" (click)="sheetOpen.set(false)"></div>
      <section
        #sheet
        id="booking-sheet"
        class="sheet rounded-card border border-border bg-surface p-[1.125rem] shadow-[0_12px_30px_-12px_rgb(15_23_42/0.2)] max-lg:pt-2"
        [attr.data-open]="sheetOpen() ? '' : null"
        [attr.role]="sheetOpen() ? 'dialog' : null"
        [attr.aria-modal]="sheetOpen() ? 'true' : null"
        aria-labelledby="booking-heading"
        tabindex="-1"
        (keydown.escape)="sheetOpen.set(false)"
      >
        <span class="mx-auto mb-2.5 block h-[5px] w-10 rounded-full bg-border lg:hidden" aria-hidden="true"></span>
        <div class="flex items-center justify-between gap-2">
          <h2 id="booking-heading" class="flex items-center gap-2 text-base font-semibold text-text">
            <span class="icon-[heroicons--ticket] size-5" aria-hidden="true"></span>{{ 'booking.title' | transloco }}
          </h2>
          <button type="button" class="ui-icon-btn -mr-1.5 size-8 lg:hidden" [attr.aria-label]="'booking.close' | transloco" (click)="sheetOpen.set(false)">
            <span class="icon-[heroicons--x-mark] size-5" aria-hidden="true"></span>
          </button>
        </div>
        <p class="mt-2 max-lg:hidden">
          <span class="text-2xl font-bold text-brand-700 dark:text-brand-300">{{ 'tours.priceValue' | transloco: { price: price() } }}</span>
          <span class="text-text-muted"> {{ 'tours.perPerson' | transloco }}</span>
        </p>
        @if (auth.user()) {
          <form class="mt-3 flex flex-col gap-3" novalidate (submit)="book($event)">
            <div>
              <span id="places-label" class="ui-label">{{ 'booking.places' | transloco }}</span>
              <div class="ui-stepper" role="group" aria-labelledby="places-label">
                <button type="button" [disabled]="quantity() <= 1" [attr.aria-label]="'booking.less' | transloco" (click)="quantity.set(quantity() - 1)">
                  <span class="icon-[heroicons--minus] size-4" aria-hidden="true"></span>
                </button>
                <span class="w-8 overflow-hidden text-center text-base font-bold text-text tabular-nums" aria-live="polite">
                  @for (value of [quantity()]; track value) {
                    <span class="motion-roll-in">{{ value }}</span>
                  }
                </span>
                <button type="button" [disabled]="quantity() >= maxPlaces" [attr.aria-label]="'booking.more' | transloco" (click)="quantity.set(quantity() + 1)">
                  <span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>
                </button>
              </div>
            </div>
            <label class="block">
              <span class="ui-label">{{ 'booking.name' | transloco }}</span>
              <span class="relative block">
                <input class="ui-input" [class.!border-success]="isValid(bookingForm.contactName())" autocomplete="name" [formField]="bookingForm.contactName" />
                @if (isValid(bookingForm.contactName())) {
                  <span class="icon-[heroicons--check-circle] motion-pop pointer-events-none absolute top-1/2 right-3 size-[1.1rem] -translate-y-1/2 text-success" aria-hidden="true"></span>
                }
              </span>
              <app-field-errors [state]="bookingForm.contactName()" />
            </label>
            <label class="block">
              <span class="ui-label">{{ 'booking.phone' | transloco }}</span>
              <span class="relative block">
                <input class="ui-input" [class.!border-success]="isValid(bookingForm.contactPhone())" type="tel" autocomplete="tel" placeholder="+996" [formField]="bookingForm.contactPhone" />
                @if (isValid(bookingForm.contactPhone())) {
                  <span class="icon-[heroicons--check-circle] motion-pop pointer-events-none absolute top-1/2 right-3 size-[1.1rem] -translate-y-1/2 text-success" aria-hidden="true"></span>
                }
              </span>
              <app-field-errors [state]="bookingForm.contactPhone()" />
            </label>
            <label class="block">
              <span class="ui-label">{{ 'booking.commentLabel' | transloco }} <span class="font-normal text-text-muted">{{ 'booking.optional' | transloco }}</span></span>
              <textarea class="ui-input" rows="2" [placeholder]="'booking.commentPlaceholder' | transloco" [formField]="bookingForm.comment"></textarea>
            </label>
            <div class="flex items-center justify-between border-t border-dashed border-border pt-3">
              <span class="text-text-muted">{{ 'booking.total' | transloco }}</span>
              <span class="overflow-hidden text-xl font-bold text-text tabular-nums">
                @for (value of [total()]; track value) {
                  <span class="motion-roll-in">{{ value }}</span>
                }
                {{ 'listing.currency' | transloco }}
              </span>
            </div>
            @if (error(); as key) {
              <p class="ui-error" role="alert"><span class="icon-[heroicons--exclamation-circle] size-4 shrink-0" aria-hidden="true"></span>{{ key | transloco }}</p>
            }
            <!-- Four states: ready, preparing the order, on the way to the
                 payment page (a check draws itself), refused (a shake). -->
            <div>
              <button
                type="submit"
                class="ui-btn-primary ui-btn-lg w-full"
                [class.motion-shake]="payState() === 'error'"
                [class.!bg-success]="payState() === 'success'"
                [disabled]="payState() === 'loading' || payState() === 'success'"
                [attr.aria-busy]="payState() === 'loading'"
                (animationend)="payState() === 'error' && payState.set('idle')"
              >
                @switch (payState()) {
                  @case ('loading') {
                    <span class="icon-[heroicons--arrow-path] size-[1.1rem] animate-spin" aria-hidden="true"></span>{{ 'booking.preparing' | transloco }}
                  }
                  @case ('success') {
                    <svg class="size-5" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path class="motion-draw" pathLength="1" d="M5 12.5l4.5 4.5L19 7.5" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" />
                    </svg>{{ 'booking.redirecting' | transloco }}
                  }
                  @default {
                    <span class="icon-[heroicons--credit-card] size-[1.1rem]" aria-hidden="true"></span>{{ 'booking.pay' | transloco }}
                  }
                }
              </button>
              @if (payState() === 'loading') {
                <div class="mt-2.5 h-[3px] overflow-hidden rounded-full bg-text/9" aria-hidden="true">
                  <div class="pay-progress h-full w-2/5 rounded-full bg-brand-400"></div>
                </div>
              }
            </div>
            <p class="flex items-start gap-1.5 text-xs text-text-muted">
              <span class="icon-[heroicons--shield-check] mt-px size-4 shrink-0" aria-hidden="true"></span>{{ 'booking.secure' | transloco }}
            </p>
            <p class="text-[0.7rem] text-text-muted">
              {{ 'legal.consent.booking' | transloco }}
              <a appSiteLink="/offer" class="underline hover:text-text">{{ 'legal.consent.offer' | transloco }}</a>
              {{ 'legal.consent.and' | transloco }}
              <a appSiteLink="/refunds" class="underline hover:text-text">{{ 'legal.consent.refunds' | transloco }}</a>.
            </p>
          </form>
        } @else {
          <div class="mt-3 flex flex-col gap-2.5">
            <p class="text-sm text-text-muted">{{ 'booking.signIn' | transloco }}</p>
            <a class="ui-btn-primary ui-btn-lg" [appSiteLink]="loginLink()">
              <span class="icon-[heroicons--user-circle] size-[1.1rem]" aria-hidden="true"></span>{{ 'booking.signInCta' | transloco }}
            </a>
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
  private readonly notify = inject(Notify);
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
  protected readonly dates = computed(() =>
    formatDateRange(this.lang(), this.tour().startDate, this.tour().endDate),
  );
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
    // The bottom bar takes the bottom navigation's place on phones.
    const chrome = inject(PageChrome);
    effect(() => chrome.ownBottomBar.set(this.bookable()));
    inject(DestroyRef).onDestroy(() => chrome.ownBottomBar.set(false));
    // The page under the sheet doesn't scroll while it's open.
    const document = inject(DOCUMENT);
    const browser = isPlatformBrowser(inject(PLATFORM_ID));
    effect(() => {
      if (browser) document.body.style.overflow = this.sheetOpen() ? 'hidden' : '';
    });

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

  // A filled-in field that passed its checks: green border and a check.
  protected isValid(state: { touched(): boolean; valid(): boolean; value(): unknown }): boolean {
    return state.touched() && state.valid() && String(state.value()).trim() !== '';
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
        if (status === 502)
          this.notify.error({ title: 'booking.gatewayDownTitle', text: 'booking.notCreated' });
        if (status === 409)
          this.notify.warning({ title: 'booking.unavailable', text: 'booking.contactOrganizer' });
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
