import type { Order } from '@/core/api/types';
import { bindPageMeta } from '@/core/seo/seo.service';
import { OrderCard } from '@/ui/order-card';
import { SiteLink } from '@/ui/site-link';
import { HttpClient } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  afterNextRender,
  computed,
  inject,
  input,
  signal,
} from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

const POLL_MS = 3000;
const MAX_POLLS = 40;

const VIEWS = {
  paid: {
    icon: 'icon-[mdi--check-circle] text-emerald-600',
    title: 'booking.result.paid',
    text: 'booking.result.paidText',
  },
  failed: {
    icon: 'icon-[mdi--close-circle] text-red-600',
    title: 'booking.result.failed',
    text: 'booking.result.failedText',
  },
  expired: {
    icon: 'icon-[mdi--timer-off-outline] text-text-muted',
    title: 'booking.result.expired',
    text: 'booking.result.failedText',
  },
  pending: {
    icon: 'icon-[mdi--loading] animate-spin text-brand-600',
    title: 'booking.result.pending',
    text: 'booking.result.pendingText',
  },
} as const;

// /payments/return?order=<id>: where the bank sends the customer back.
// Checks the payment right away, then keeps checking for a while (the
// bank can take a moment to settle). Browser only.
@Component({
  selector: 'app-payment-return-page',
  imports: [OrderCard, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="mx-auto flex max-w-lg flex-col items-center gap-5 px-4 py-16 text-center">
      @if (failed()) {
        <span class="icon-[mdi--alert-circle-outline] size-16 text-text-muted" aria-hidden="true"></span>
        <h1 class="text-2xl font-extrabold text-text">{{ 'booking.result.notFound' | transloco }}</h1>
      } @else {
        <span [class]="view().icon + ' size-16'" aria-hidden="true"></span>
        <h1 class="text-2xl font-extrabold text-text" aria-live="polite">{{ view().title | transloco }}</h1>
        <p class="text-text-muted">{{ view().text | transloco }}</p>
        @if (result(); as item) {
          <app-order-card class="w-full text-left" [order]="item" />
        }
      }
      <a appSiteLink="/profile?tab=bookings" class="ui-btn-primary">{{ 'booking.myBookings' | transloco }}</a>
    </section>
  `,
})
export class PaymentReturnPage {
  readonly order = input<string | undefined>(undefined);
  private readonly http = inject(HttpClient);

  protected readonly result = signal<Order | null>(null);
  protected readonly failed = signal(false);
  protected readonly view = computed(() => {
    const status = this.result()?.status;
    return status === 'paid' || status === 'failed' || status === 'expired'
      ? VIEWS[status]
      : VIEWS.pending;
  });
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
    if (this.result()?.status === 'pending' && ++this.polls < MAX_POLLS)
      this.timer = setTimeout(() => void this.check(), POLL_MS);
  }
}
