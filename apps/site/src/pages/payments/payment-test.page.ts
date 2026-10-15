import type { Order } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount } from '@/core/lib/format';
import { bindPageMeta } from '@/core/seo/seo.service';
import { HttpClient, httpResource } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  DOCUMENT,
  computed,
  inject,
  input,
  signal,
} from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// /payments/test?order=<id>: stands in for the bank's payment page while
// payments run in test mode (PAYMENTS_PROVIDER=fake): no money moves.
@Component({
  selector: 'app-payment-test-page',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="mx-auto max-w-md px-4 py-16">
      <div class="rounded-card border-2 border-dashed border-amber-500 bg-surface p-6 text-center">
        <p class="mb-4 rounded-lg bg-amber-400/15 px-3 py-2 text-sm font-semibold text-amber-800 dark:text-amber-200">{{ 'booking.test.banner' | transloco }}</p>
        <h1 class="sr-only">{{ 'booking.test.title' | transloco }}</h1>
        @if (item(); as order) {
          <p class="text-text-muted">{{ order.itemTitle }}</p>
          <p class="my-3 text-4xl font-black text-text">{{ amount() }} {{ 'listing.currency' | transloco }}</p>
          <div class="mt-6 flex flex-col gap-2">
            <button type="button" class="ui-btn-primary py-2.5 text-base" [disabled]="busy()" (click)="settle(true)">{{ 'booking.test.pay' | transloco }}</button>
            <button type="button" class="ui-btn-soft py-2.5" [disabled]="busy()" (click)="settle(false)">{{ 'booking.test.decline' | transloco }}</button>
          </div>
        } @else if (orderResource.error() || !order()) {
          <p class="text-text-muted">{{ 'booking.result.notFound' | transloco }}</p>
        }
      </div>
    </section>
  `,
})
export class PaymentTestPage {
  readonly order = input<string | undefined>(undefined);
  private readonly http = inject(HttpClient);
  private readonly location = inject(DOCUMENT).location;
  private readonly lang = injectLang();

  protected readonly orderResource = httpResource<Order>(() => {
    const id = this.order();
    return id ? `/orders/${encodeURIComponent(id)}` : undefined;
  });
  protected readonly item = computed(() =>
    this.orderResource.hasValue() ? this.orderResource.value() : undefined,
  );
  protected readonly amount = computed(() => formatAmount(this.lang(), this.item()?.amount ?? 0));
  protected readonly busy = signal(false);

  constructor() {
    const transloco = inject(TranslocoService);
    bindPageMeta(() => ({
      title: transloco.translate('booking.test.title'),
      description: transloco.translate('booking.test.banner'),
      robots: 'noindex',
      alternates: null,
    }));
  }

  // The fake provider answers with the return page, as a bank would redirect.
  protected async settle(success: boolean): Promise<void> {
    this.busy.set(true);
    try {
      const { returnUrl } = await firstValueFrom(
        this.http.post<{ returnUrl: string }>(`/payments/test/${this.order()}`, { success }),
      );
      this.location.assign(returnUrl);
    } finally {
      this.busy.set(false);
    }
  }
}
