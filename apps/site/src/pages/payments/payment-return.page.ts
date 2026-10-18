import type { Order } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, formatInBishkek } from '@/core/lib/format';
import { bindPageMeta } from '@/core/seo/seo.service';
import { SiteLink } from '@/ui/site-link';
import { StateBlock } from '@/ui/state-block';
import { DOCUMENT } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  ElementRef,
  afterNextRender,
  computed,
  inject,
  input,
  signal,
  viewChild,
} from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

const POLL_MS = 3000;
const MAX_POLLS = 40;

// /payments/return?order=<id>: where the bank sends the customer back.
// Checks the payment right away, then keeps checking for a while (the
// bank can take a moment to settle). Browser only.
@Component({
  selector: 'app-payment-return-page',
  imports: [SiteLink, StateBlock, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  styles: `
    @keyframes confetti {
      0% {
        opacity: 0;
        transform: translateY(8px) scale(0.6);
      }
      30% {
        opacity: 1;
      }
      100% {
        opacity: 0;
        transform: translateY(-18px) scale(1);
      }
    }
    .confetti path {
      animation: confetti 600ms var(--ease-enter) both;
    }
    @media (prefers-reduced-motion: reduce) {
      .confetti {
        display: none;
      }
    }
  `,
  template: `
    <section class="ui-container flex justify-center py-12 sm:py-16">
      <div class="ui-card relative w-full max-w-md overflow-hidden" tabindex="-1" #panel>
        @if (failed()) {
          <app-state-block [card]="false" icon="icon-[heroicons--exclamation-circle]" tone="warning" title="booking.result.notFound">
            <a appSiteLink="/profile?tab=bookings" class="ui-btn-primary">{{ 'booking.myBookings' | transloco }}</a>
          </app-state-block>
        } @else {
          @switch (state()) {
            @case ('paid') {
              <svg class="confetti pointer-events-none absolute inset-x-0 top-3 h-24 w-full" viewBox="0 0 200 120" aria-hidden="true">
                <g fill="none" stroke-width="3" stroke-linecap="round">
                  <path d="M30 30l6 8" stroke="#d62828" /><path d="M170 26l-7 7" stroke="#f4a261" style="animation-delay: 60ms" />
                  <path d="M60 14l2 9" stroke="#ffb703" style="animation-delay: 120ms" /><path d="M142 60l8 3" stroke="#d62828" style="animation-delay: 40ms" />
                  <path d="M44 70l-8 4" stroke="#f4a261" style="animation-delay: 100ms" /><path d="M120 12l-3 8" stroke="#16a34a" style="animation-delay: 80ms" />
                </g>
              </svg>
              <div class="ui-state">
                <span class="ui-state-icon is-success motion-pop">
                  <svg class="size-8" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                    <path class="motion-draw" pathLength="1" d="M5 12.5l4.5 4.5L19 7.5" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" />
                  </svg>
                </span>
                <h1 class="text-lg font-semibold text-text" role="status">{{ 'booking.result.paid' | transloco }}</h1>
                <p class="max-w-xs text-[0.8125rem] text-text-muted">{{ 'booking.result.paidText' | transloco }}</p>
                @if (result(); as order) {
                  <dl class="mt-1 w-full rounded-[0.625rem] bg-text/5 px-3 py-2.5 text-left text-[0.8rem]">
                    <div class="flex justify-between gap-3"><dt class="text-text-muted">{{ 'booking.result.tour' | transloco }}</dt><dd class="truncate font-semibold text-text">{{ order.itemTitle }}</dd></div>
                    <div class="flex justify-between gap-3"><dt class="text-text-muted">{{ 'booking.places' | transloco }}</dt><dd class="font-semibold text-text">{{ order.quantity }}</dd></div>
                    <div class="flex justify-between gap-3"><dt class="text-text-muted">{{ 'booking.result.paidAmount' | transloco }}</dt><dd class="font-semibold text-text tabular-nums">{{ amount(order) }} {{ 'listing.currency' | transloco }}</dd></div>
                  </dl>
                }
                <div class="mt-2 flex flex-wrap justify-center gap-2">
                  <a appSiteLink="/profile?tab=bookings" class="ui-btn-primary">{{ 'booking.myBookings' | transloco }}</a>
                  @if (result()?.itemDate) {
                    <button type="button" class="ui-btn-soft" (click)="addToCalendar()">
                      <span class="icon-[heroicons--calendar-days] size-4" aria-hidden="true"></span>{{ 'booking.result.calendar' | transloco }}
                    </button>
                  }
                </div>
              </div>
            }
            @case ('failed') {
              <div class="ui-state">
                <span class="ui-state-icon is-danger motion-pop"><span class="icon-[heroicons--x-mark]" aria-hidden="true"></span></span>
                <h1 class="text-lg font-semibold text-text" role="status">{{ titleKey() | transloco }}</h1>
                <p class="max-w-xs text-[0.8125rem] text-text-muted">{{ 'booking.result.failedText' | transloco }}</p>
                @if (result(); as order) {
                  @if (order.status === 'pending' && holdUntil(); as time) {
                    <p class="ui-banner mt-1 w-full bg-text/5 text-[0.8rem]">
                      <span class="icon-[heroicons--clock] text-text-muted" aria-hidden="true"></span>{{ 'booking.holdUntil' | transloco: { time: time } }}
                    </p>
                  }
                  <div class="mt-2 flex flex-wrap justify-center gap-2">
                    @if (order.paymentUrl && order.status === 'pending') {
                      <a [href]="order.paymentUrl" class="ui-btn-primary"><span class="icon-[heroicons--credit-card] size-4" aria-hidden="true"></span>{{ 'booking.continuePayment' | transloco }}</a>
                    }
                    @if (order.itemSlug; as slug) {
                      <a [appSiteLink]="'/tours/' + slug" class="ui-btn-ghost">{{ 'booking.openTour' | transloco }}</a>
                    }
                  </div>
                }
              </div>
            }
            @default {
              <div class="ui-state">
                <span class="ui-state-icon is-warning"><span class="icon-[heroicons--arrow-path] animate-spin [animation-duration:1.2s] motion-reduce:animate-none" aria-hidden="true"></span></span>
                <h1 class="text-lg font-semibold text-text" role="status">{{ 'booking.result.pending' | transloco }}</h1>
                <p class="max-w-xs text-[0.8125rem] text-text-muted">{{ 'booking.result.pendingText' | transloco }}</p>
                <div class="mt-1 flex gap-1.5" aria-hidden="true">
                  <span class="size-2 animate-pulse rounded-full bg-warning"></span>
                  <span class="size-2 animate-pulse rounded-full bg-warning/60 [animation-delay:200ms]"></span>
                  <span class="size-2 animate-pulse rounded-full bg-warning/30 [animation-delay:400ms]"></span>
                </div>
                <button type="button" class="ui-btn-soft mt-2" [disabled]="checking()" (click)="checkNow()">
                  <span class="icon-[heroicons--arrow-path] size-4" [class.animate-spin]="checking()" aria-hidden="true"></span>{{ 'booking.checkStatus' | transloco }}
                </button>
              </div>
            }
          }
        }
      </div>
    </section>
  `,
})
export class PaymentReturnPage {
  readonly order = input<string | undefined>(undefined);
  private readonly http = inject(HttpClient);
  private readonly lang = injectLang();
  private readonly document = inject(DOCUMENT);
  private readonly panel = viewChild<ElementRef<HTMLElement>>('panel');

  protected readonly result = signal<Order | null>(null);
  protected readonly failed = signal(false);
  protected readonly checking = signal(false);
  protected readonly state = computed(() => {
    const status = this.result()?.status;
    if (status === 'paid') return 'paid';
    if (status === 'failed' || status === 'expired') return 'failed';
    // Still pending after the last check: the bank didn't confirm.
    if (status === 'pending' && this.gaveUp()) return 'failed';
    return 'pending';
  });
  protected readonly titleKey = computed(() =>
    this.result()?.status === 'expired' ? 'booking.result.expired' : 'booking.result.failed',
  );
  protected readonly holdUntil = computed(() => {
    const order = this.result();
    if (!order || Date.parse(order.expiresAt) <= Date.now()) return null;
    return formatInBishkek(this.lang(), order.expiresAt, { hour: '2-digit', minute: '2-digit' });
  });
  private readonly gaveUp = signal(false);
  private polls = 0;
  private timer: ReturnType<typeof setTimeout> | undefined;

  constructor() {
    afterNextRender(() => {
      if (this.order()) void this.check();
      else this.failed.set(true);
    });
    inject(DestroyRef).onDestroy(() => clearTimeout(this.timer));
    const transloco = inject(TranslocoService);
    bindPageMeta(() => ({
      title: transloco.translate('booking.result.title'),
      description: transloco.translate('booking.result.pendingText'),
      robots: 'noindex',
      alternates: null,
    }));
  }

  // Asks the payments service to check with the bank.
  private async check(): Promise<void> {
    try {
      this.result.set(
        await firstValueFrom(
          this.http.post<Order>(`/orders/${encodeURIComponent(this.order() ?? '')}/check`, {}),
        ),
      );
    } catch {
      this.failed.set(true);
      return;
    }
    if (this.result()?.status !== 'pending') {
      // The answer is in: focus moves to it.
      this.panel()?.nativeElement.focus();
      return;
    }
    if (++this.polls < MAX_POLLS) this.timer = setTimeout(() => void this.check(), POLL_MS);
    else this.gaveUp.set(true);
  }

  protected async checkNow(): Promise<void> {
    clearTimeout(this.timer);
    this.checking.set(true);
    await this.check();
    this.checking.set(false);
  }

  protected amount(order: Order): string {
    return formatAmount(this.lang(), order.amount);
  }

  // The trip as a calendar file (.ics), for the phone's calendar.
  protected addToCalendar(): void {
    const order = this.result();
    if (!order?.itemDate) return;
    const date = order.itemDate.replaceAll('-', '');
    const ics = [
      'BEGIN:VCALENDAR',
      'VERSION:2.0',
      'PRODID:-//Ermekten//Booking//RU',
      'BEGIN:VEVENT',
      `UID:${order.id}@ermekten.com`,
      `DTSTAMP:${new Date().toISOString().replace(/[-:]/g, '').slice(0, 15)}Z`,
      `DTSTART;VALUE=DATE:${date}`,
      `SUMMARY:${order.itemTitle.replace(/[,;]/g, ' ')}`,
      'END:VEVENT',
      'END:VCALENDAR',
    ].join('\r\n');
    const link = this.document.createElement('a');
    link.href = URL.createObjectURL(new Blob([ics], { type: 'text/calendar' }));
    link.download = 'ermekten-booking.ics';
    link.click();
    URL.revokeObjectURL(link.href);
  }
}
