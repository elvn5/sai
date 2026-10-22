import { Analytics } from '@/core/analytics/analytics';
import type { CancellationPolicy, ListingStatEvent, ReviewTargetType } from '@/core/api/types';
import { phoneLink, telegramLink, whatsappLink } from '@/core/lib/contact-links';
import { ListingStats } from '@/core/lib/listing-stats';
import { RatingStars } from '@/ui/rating-stars';
import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

// The "booking" box of a listing. Most bookings here happen over WhatsApp
// or Telegram, so contacting is the main action; clicks are counted for the
// owner's partner cabinet.
@Component({
  selector: 'app-contact-card',
  imports: [RatingStars, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="ui-card flex flex-col gap-4 p-4">
      @if (priceLabel()) {
        <p class="text-2xl font-bold text-brand-700 dark:text-brand-300">{{ priceLabel() }}</p>
      }
      <a href="#reviews" class="flex items-center gap-2 text-sm">
        <app-rating-stars [value]="ratingAverage()" size="md" />
        @if (reviewCount() > 0) {
          <span class="font-semibold text-text">{{ ratingAverage().toFixed(1) }}</span>
        }
        <span class="text-text-muted">{{ 'reviews.count' | transloco: { count: reviewCount() } }}</span>
      </a>
      @if (cancellationPolicy(); as policy) {
        <div
          class="flex items-start gap-2 rounded-lg px-3 py-2 text-sm"
          [class]="policy === 'free' ? 'bg-green-500/10 text-green-800 dark:text-green-300' : 'bg-slate-500/10 text-text-muted'"
        >
          @if (policy === 'free') {
            <span class="icon-[heroicons--check-circle] mt-0.5 size-4 shrink-0" aria-hidden="true"></span>
          } @else {
            <span class="icon-[heroicons--information-circle] mt-0.5 size-4 shrink-0" aria-hidden="true"></span>
          }
          <div>
            <p class="font-medium">{{ 'listing.cancellation.' + policy | transloco }}</p>
            <p class="text-xs opacity-90">{{ 'listing.cancellationHint.' + policy | transloco }}</p>
          </div>
        </div>
      }
      @if (hasContacts()) {
        <div class="flex flex-col gap-2">
          @if (whatsappHref(); as href) {
            <a [href]="href" target="_blank" rel="noopener noreferrer" class="inline-flex items-center justify-center gap-2 rounded-lg bg-[#157a3c] px-4 py-2.5 font-medium text-white hover:bg-[#11632f]" (click)="track('whatsapp')">
              <span class="icon-[mdi--whatsapp] size-5" aria-hidden="true"></span>{{ 'listing.contact.whatsapp' | transloco }}
            </a>
          }
          @if (telegram(); as username) {
            <a [href]="telegramHref(username)" target="_blank" rel="noopener noreferrer" class="inline-flex items-center justify-center gap-2 rounded-lg bg-[#166a96] px-4 py-2.5 font-medium text-white hover:bg-[#12577b]" (click)="track('telegram')">
              <span class="icon-[mdi--telegram] size-5" aria-hidden="true"></span>{{ 'listing.contact.telegram' | transloco }}
            </a>
          }
          @if (phone(); as number) {
            <a [href]="phoneHref(number)" class="ui-btn-soft justify-center py-2" (click)="track('phone')">
              <span class="icon-[heroicons--phone] size-4" aria-hidden="true"></span>{{ number }}
            </a>
          }
          @if (website(); as url) {
            <a [href]="url" target="_blank" rel="noopener noreferrer nofollow" class="ui-btn-ghost justify-center py-2" (click)="track('website')">
              <span class="icon-[heroicons--globe-alt] size-4" aria-hidden="true"></span>{{ 'listing.contact.website' | transloco }}
            </a>
          }
        </div>
      } @else {
        <p class="text-sm text-text-muted">{{ 'listing.contact.none' | transloco }}</p>
      }
    </div>
  `,
})
export class ContactCard {
  readonly targetType = input.required<ReviewTargetType>();
  readonly targetId = input.required<string>();
  readonly listingName = input.required<string>();
  readonly priceLabel = input<string | null>(null);
  readonly ratingAverage = input(0);
  readonly reviewCount = input(0);
  readonly cancellationPolicy = input<CancellationPolicy | null>(null);
  readonly phone = input<string | null>(null);
  readonly whatsapp = input<string | null>(null);
  readonly telegram = input<string | null>(null);
  readonly website = input<string | null>(null);

  private readonly stats = inject(ListingStats);
  private readonly analytics = inject(Analytics);
  private readonly transloco = inject(TranslocoService);
  protected readonly telegramHref = telegramLink;
  protected readonly phoneHref = phoneLink;
  protected readonly whatsappHref = computed(() => {
    const number = this.whatsapp();
    return number
      ? whatsappLink(
          number,
          this.transloco.translate('listing.contact.messageTemplate', { name: this.listingName() }),
        )
      : null;
  });
  protected readonly hasContacts = computed(() =>
    Boolean(this.phone() || this.whatsapp() || this.telegram() || this.website()),
  );

  protected track(event: ListingStatEvent): void {
    this.stats.track(this.targetType(), this.targetId(), event);
    this.analytics.track('contact_organizer', {
      item_type: this.targetType(),
      item_id: this.targetId(),
      method: event,
    });
  }
}
