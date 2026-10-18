import { type Hotel, ReviewTargetType } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount } from '@/core/lib/format';
import { HOTEL_AMENITY_OPTIONS, pickOptions } from '@/core/lib/listing-options';
import { trackListingView } from '@/core/lib/listing-stats';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { loaded } from '@/core/lib/resource';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { lodgingBusiness, plainText } from '@/core/seo/json-ld';
import { SeoService, bindPageMeta } from '@/core/seo/seo.service';
import { NotFoundPage } from '@/pages/not-found/not-found.page';
import { injectConfirm } from '@/ui/confirm-dialog';
import { ContactCard } from '@/ui/listing/contact-card';
import { PostsFeed } from '@/ui/listing/posts-feed';
import { reviewsResource } from '@/ui/listing/reviews';
import { ReviewsSection } from '@/ui/listing/reviews-section';
import { LoadError } from '@/ui/load-error';
import { LocationMap } from '@/ui/location-map';
import { PhotoGrid } from '@/ui/photo-grid';
import { RatingStars } from '@/ui/rating-stars';
import { SiteLink } from '@/ui/site-link';
import { HttpClient, HttpErrorResponse, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, input, signal } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// One hotel: photos, amenities, description, map, the owner's posts and
// reviews, with the contact box at the side. Hotels are the same in every
// language (one slug), so the page has hreflang alternates.
@Component({
  selector: 'app-hotel-page',
  imports: [
    ContactCard,
    LoadError,
    LocationMap,
    NotFoundPage,
    PhotoGrid,
    PostsFeed,
    RatingStars,
    ReviewsSection,
    SiteLink,
    TranslocoPipe,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (notFound()) {
      <app-not-found-page />
    } @else if (hotel.error()) {
      <div class="mx-auto max-w-6xl px-4 py-12"><app-load-error (retry)="hotel.reload()" /></div>
    } @else if (hotel.value(); as item) {
      <article class="mx-auto max-w-6xl px-4 py-10">
        <a appSiteLink="/hotels" class="text-sm text-brand-700 hover:underline dark:text-brand-300">&larr; {{ 'hotels.heading' | transloco }}</a>

        <div class="mt-4 flex flex-wrap items-start justify-between gap-4">
          <div>
            <div class="flex flex-wrap gap-2">
              <span class="ui-badge bg-text/5 text-text">{{ 'hotels.types.' + item.type | transloco }}</span>
              <a [appSiteLink]="'/hotels?destination=' + item.destination" class="ui-badge inline-flex items-center gap-1 bg-secondary-500/15 text-secondary-800 dark:text-secondary-200">
                <span class="icon-[heroicons--map-pin] size-3.5" aria-hidden="true"></span>{{ 'destinations.' + item.destination | transloco }}
              </a>
            </div>
            <h1 class="mt-2 text-3xl font-extrabold text-text sm:text-4xl">
              {{ item.name }}
              @if (item.stars; as stars) {
                <span class="text-xl text-amber-600 dark:text-amber-400" [attr.aria-label]="stars + '★'">{{ starsText(stars) }}</span>
              }
            </h1>
            <a href="#reviews" class="mt-2 flex items-center gap-2 text-sm">
              <app-rating-stars [value]="item.ratingAverage" />
              @if (item.reviewCount > 0) {
                <span class="font-semibold text-text">{{ item.ratingAverage.toFixed(1) }}</span>
              }
              <span class="text-text-muted">{{ 'reviews.count' | transloco: { count: item.reviewCount } }}</span>
            </a>
          </div>
          @if (isOwner()) {
            <div class="flex gap-2">
              <a [appSiteLink]="'/hotels/' + item.slug + '/edit'" class="ui-btn-soft">
                <span class="icon-[heroicons--pencil-square] size-4" aria-hidden="true"></span>{{ 'listing.edit' | transloco }}
              </a>
              <button type="button" class="ui-btn-danger" [disabled]="deleting()" (click)="remove(item)">
                <span class="icon-[heroicons--trash] size-4" aria-hidden="true"></span>{{ 'listing.delete' | transloco }}
              </button>
            </div>
          }
        </div>

        <app-photo-grid class="mt-6 block" variant="hero" [photos]="item.photos" [alt]="item.name" />

        <div class="mt-8 flex flex-col gap-10 lg:flex-row">
          <div class="flex min-w-0 flex-1 flex-col gap-10">
            @if (amenities().length > 0) {
              <section aria-labelledby="amenities-heading">
                <h2 id="amenities-heading" class="text-2xl font-bold text-text">{{ 'hotels.amenitiesLabel' | transloco }}</h2>
                <ul class="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3">
                  @for (amenity of amenities(); track amenity.value) {
                    <li class="flex items-center gap-2 text-text">
                      <span [class]="amenity.icon + ' size-5 text-brand-600 dark:text-brand-300'" aria-hidden="true"></span>{{ amenity.labelKey | transloco }}
                    </li>
                  }
                </ul>
              </section>
            }

            <section aria-labelledby="about-heading">
              <h2 id="about-heading" class="text-2xl font-bold text-text">{{ 'hotels.about' | transloco }}</h2>
              <div class="prose prose-neutral dark:prose-invert mt-4 max-w-none" [innerHTML]="item.description"></div>
              @if (item.checkInTime || item.checkOutTime) {
                <div class="mt-4 flex flex-wrap gap-6 text-sm text-text">
                  @if (item.checkInTime) {
                    <span class="flex items-center gap-2">
                      <span class="icon-[heroicons--arrow-right-end-on-rectangle] size-5 text-text-muted" aria-hidden="true"></span>{{ 'hotels.checkIn' | transloco: { time: item.checkInTime } }}
                    </span>
                  }
                  @if (item.checkOutTime) {
                    <span class="flex items-center gap-2">
                      <span class="icon-[heroicons--arrow-left-start-on-rectangle] size-5 text-text-muted" aria-hidden="true"></span>{{ 'hotels.checkOut' | transloco: { time: item.checkOutTime } }}
                    </span>
                  }
                </div>
              }
            </section>

            @if (item.address || item.latitude !== null) {
              @defer (on viewport; hydrate on viewport) {
                <section aria-labelledby="map-heading">
                  <h2 id="map-heading" class="mb-4 text-2xl font-bold text-text">{{ 'listing.map.heading' | transloco }}</h2>
                  <app-location-map [latitude]="item.latitude" [longitude]="item.longitude" [address]="item.address" />
                </section>
              } @placeholder {
                <div class="h-80"></div>
              }
            }

            @defer (on viewport; hydrate on viewport) {
              <app-posts-feed [targetType]="targetType" [targetId]="item.id" [ownerId]="item.userId" />
            } @placeholder {
              <div></div>
            }

            @defer (on viewport; hydrate on viewport) {
              <app-reviews-section [targetType]="targetType" [targetId]="item.id" [ownerId]="item.userId" [page]="reviewsPage()" (changed)="refresh()" />
            } @placeholder {
              <div class="min-h-40"></div>
            }
          </div>

          <aside class="order-first lg:order-none lg:w-80 lg:shrink-0">
            <div class="lg:sticky lg:top-20">
              <app-contact-card
                [targetType]="targetType"
                [targetId]="item.id"
                [listingName]="item.name"
                [priceLabel]="priceLabel()"
                [ratingAverage]="item.ratingAverage"
                [reviewCount]="item.reviewCount"
                [cancellationPolicy]="item.cancellationPolicy"
                [phone]="item.phone"
                [whatsapp]="item.whatsapp"
                [telegram]="item.telegram"
                [website]="item.website"
              />
            </div>
          </aside>
        </div>
      </article>
    }
  `,
})
export class HotelPage {
  readonly slug = input.required<string>();
  protected readonly targetType = ReviewTargetType.HOTEL;

  private readonly auth = inject(AuthStore);
  private readonly http = inject(HttpClient);
  private readonly transloco = inject(TranslocoService);
  private readonly confirm = injectConfirm();
  private readonly apiBaseUrl = inject(APP_CONFIG).apiBaseUrl;
  private readonly navigate = injectSiteNavigate();
  private readonly lang = injectLang();

  protected readonly hotel = httpResource<Hotel>(
    () => `/hotels/${encodeURIComponent(this.slug())}`,
  );
  // The loaded hotel, or undefined (value() throws while the resource is in error).
  private readonly item = computed(() => (this.hotel.hasValue() ? this.hotel.value() : undefined));
  protected readonly reviews = reviewsResource(
    this.targetType,
    computed(() => this.item()?.id),
  );
  protected readonly reviewsPage = computed(() => loaded(this.reviews));
  protected readonly notFound = computed(() => {
    const error = this.hotel.error();
    return error instanceof HttpErrorResponse && error.status === 404;
  });
  protected readonly isOwner = computed(() => {
    const user = this.auth.user();
    return Boolean(user && this.item()?.userId === user.id);
  });
  protected readonly amenities = computed(() =>
    pickOptions(HOTEL_AMENITY_OPTIONS, this.item()?.amenities ?? []),
  );
  protected readonly priceLabel = computed(() => {
    const price = this.item()?.priceFrom;
    return price != null
      ? this.transloco.translate('listing.priceFromPerNight', {
          price: formatAmount(this.lang(), price),
        })
      : null;
  });
  protected readonly deleting = signal(false);
  protected readonly starsText = (stars: number): string => '★'.repeat(stars);

  constructor() {
    const seo = inject(SeoService);
    bindPageMeta(() => {
      const item = this.item();
      if (!item || this.reviews.isLoading()) return undefined;
      const place = this.transloco.translate(`destinations.${item.destination}`);
      const path = `/hotels/${item.slug}`;
      const images = [item.coverImageUrl, ...item.photos]
        .filter((url): url is string => Boolean(url))
        .filter((url, index, all) => all.indexOf(url) === index)
        .map((url) => this.media(url));
      return {
        title: `${item.name} — ${place}`,
        description: this.transloco.translate('hotels.detailMeta', { name: item.name, place }),
        image: images[0] ?? null,
        jsonLd: [
          lodgingBusiness({
            url: seo.pageUrl(path),
            name: item.name,
            description: plainText(item.description),
            images,
            address: item.address,
            locality: place,
            latitude: item.latitude,
            longitude: item.longitude,
            priceFrom: item.priceFrom,
            stars: item.stars,
            amenities: this.amenities().map((amenity) =>
              this.transloco.translate(amenity.labelKey),
            ),
            checkInTime: item.checkInTime,
            checkOutTime: item.checkOutTime,
            telephone: item.phone,
            ratingAverage: item.ratingAverage,
            reviewCount: item.reviewCount,
            reviews: (loaded(this.reviews)?.items ?? []).map((review) => ({
              author:
                review.author.firstName ??
                review.author.username ??
                this.transloco.translate('reviews.anonymous'),
              rating: review.rating,
              text: review.text,
              date: review.createdAt,
            })),
          }),
          seo.breadcrumbs([
            { name: this.transloco.translate('hotels.heading'), path: '/hotels' },
            { name: item.name, path },
          ]),
        ],
      };
    });

    trackListingView(
      this.targetType,
      computed(() => this.item()?.id),
    );
  }

  protected media(url: string): string {
    return resolveMediaUrl(this.apiBaseUrl, url);
  }

  // A review changed the rating shown in the header.
  protected refresh(): void {
    this.hotel.reload();
    this.reviews.reload();
  }

  protected async remove(item: Hotel): Promise<void> {
    if (!(await this.confirm({ title: 'hotels.deleteConfirm', danger: true }))) return;
    this.deleting.set(true);
    try {
      await firstValueFrom(this.http.delete(`/hotels/${item.id}`));
      await this.navigate('/hotels');
    } finally {
      this.deleting.set(false);
    }
  }
}
