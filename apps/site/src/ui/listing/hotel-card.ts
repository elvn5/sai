import type { Hotel } from '@/core/api/types';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount } from '@/core/lib/format';
import { HOTEL_AMENITY_OPTIONS, pickOptions } from '@/core/lib/listing-options';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { hotelCoverTransition } from '@/core/lib/view-transitions';
import { RatingStars } from '@/ui/rating-stars';
import { SharedTransition } from '@/ui/shared-transition';
import { SiteLink } from '@/ui/site-link';
import { NgOptimizedImage } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

const CARD_AMENITIES = 4;

@Component({
  selector: 'app-hotel-card',
  imports: [NgOptimizedImage, RatingStars, SharedTransition, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <a [appSiteLink]="'/hotels/' + hotel().slug" class="group flex h-full flex-col overflow-hidden rounded-card border border-border bg-surface motion-lift">
      <div class="relative aspect-[4/3] overflow-hidden bg-slate-100 dark:bg-slate-800" [appSharedTransition]="coverTransition()">
        @if (cover(); as url) {
          <img [ngSrc]="url" fill [priority]="priority()" sizes="(min-width: 1280px) 25vw, (min-width: 640px) 50vw, 100vw" alt="" class="motion-zoom object-cover" />
        } @else {
          <span class="icon-[heroicons--building-office-2] absolute inset-0 m-auto size-12 text-text-muted" aria-hidden="true"></span>
        }
        <span class="absolute top-2 left-2 rounded-full bg-black/65 px-2.5 py-1 text-xs font-medium text-white">{{ 'hotels.types.' + hotel().type | transloco }}</span>
        @if (hotel().cancellationPolicy === 'free') {
          <span class="absolute top-2 right-2 rounded-full bg-green-700 px-2.5 py-1 text-xs font-medium text-white">{{ 'listing.cancellation.free' | transloco }}</span>
        }
      </div>
      <div class="flex flex-1 flex-col gap-2 p-4">
        <p class="flex items-center gap-1 text-sm text-text-muted">
          <span class="icon-[heroicons--map-pin] size-4" aria-hidden="true"></span>{{ 'destinations.' + hotel().destination | transloco }}
        </p>
        <h2 class="text-lg leading-snug font-semibold text-text">
          {{ hotel().name }}
          @if (hotel().stars; as stars) {
            <span class="ml-1 text-sm text-amber-600 dark:text-amber-400" [attr.aria-label]="stars + '★'">{{ starsText(stars) }}</span>
          }
        </h2>
        <div class="flex items-center gap-2 text-sm">
          @if (hotel().reviewCount > 0) {
            <span class="rounded bg-brand-700 px-1.5 py-0.5 font-semibold text-white">{{ hotel().ratingAverage.toFixed(1) }}</span>
            <app-rating-stars [value]="hotel().ratingAverage" />
          }
          <span class="text-text-muted">{{ 'reviews.count' | transloco: { count: hotel().reviewCount } }}</span>
        </div>
        @if (amenities().length > 0) {
          <ul class="flex flex-wrap gap-x-3 gap-y-1 text-xs text-text-muted">
            @for (amenity of amenities(); track amenity.value) {
              <li class="inline-flex items-center gap-1">
                <span [class]="amenity.icon + ' size-3.5'" aria-hidden="true"></span>{{ amenity.labelKey | transloco }}
              </li>
            }
          </ul>
        }
        @if (hotel().priceFrom !== null) {
          <p class="mt-auto pt-2 text-right text-text">
            <span class="text-sm text-text-muted">{{ 'listing.from' | transloco }} </span>
            <span class="text-xl font-bold text-brand-700 dark:text-brand-300">{{ price() }} {{ 'listing.currency' | transloco }}</span>
            <span class="text-sm text-text-muted"> / {{ 'listing.perNight' | transloco }}</span>
          </p>
        }
      </div>
    </a>
  `,
})
export class HotelCard {
  readonly hotel = input.required<Hotel>();
  readonly priority = input(false);
  private readonly apiBaseUrl = inject(APP_CONFIG).apiBaseUrl;
  private readonly lang = injectLang();

  // The cover flies to the hotel page's photos once clicked.
  protected readonly coverTransition = computed(() => hotelCoverTransition(this.hotel().id));
  protected readonly cover = computed(() => {
    const url = this.hotel().coverImageUrl ?? this.hotel().photos[0];
    return url ? resolveMediaUrl(this.apiBaseUrl, url) : null;
  });
  protected readonly amenities = computed(() =>
    pickOptions(HOTEL_AMENITY_OPTIONS, this.hotel().amenities.slice(0, CARD_AMENITIES)),
  );
  protected readonly price = computed(() => formatAmount(this.lang(), this.hotel().priceFrom ?? 0));
  protected readonly starsText = (stars: number): string => '★'.repeat(stars);
}
