import { ReviewTargetType, type Tour } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, formatDateRange, todayInBishkek } from '@/core/lib/format';
import { trackListingView } from '@/core/lib/listing-stats';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { loaded } from '@/core/lib/resource';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { plainText, touristTrip } from '@/core/seo/json-ld';
import { SeoService, bindPageMeta } from '@/core/seo/seo.service';
import { NotFoundPage } from '@/pages/not-found/not-found.page';
import { TourBooking } from '@/pages/tours/tour-booking';
import { ContactCard } from '@/ui/listing/contact-card';
import { PostsFeed } from '@/ui/listing/posts-feed';
import { reviewsResource } from '@/ui/listing/reviews';
import { ReviewsSection } from '@/ui/listing/reviews-section';
import { LoadError } from '@/ui/load-error';
import { SiteLink } from '@/ui/site-link';
import { NgOptimizedImage } from '@angular/common';
import { HttpClient, HttpErrorResponse, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, input, signal } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// One tour (written in one language: no hreflang alternates), with
// booking, contacts, the owner's posts and reviews.
@Component({
  selector: 'app-tour-page',
  imports: [
    ContactCard,
    LoadError,
    NgOptimizedImage,
    NotFoundPage,
    PostsFeed,
    ReviewsSection,
    SiteLink,
    TourBooking,
    TranslocoPipe,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (notFound()) {
      <app-not-found-page />
    } @else if (tour.error()) {
      <div class="mx-auto max-w-6xl px-4 py-12"><app-load-error (retry)="tour.reload()" /></div>
    } @else if (tour.value(); as item) {
      <article class="mx-auto max-w-6xl px-4 py-12">
        <a appSiteLink="/tours" class="text-sm text-brand-700 hover:underline dark:text-brand-300">&larr; {{ 'tours.heading' | transloco }}</a>
        <div class="mt-4 flex flex-wrap gap-2">
          @if (item.destination) {
            <a [appSiteLink]="'/tours?destination=' + item.destination" class="ui-badge inline-flex items-center gap-1 bg-secondary-500/15 text-secondary-800 dark:text-secondary-200">
              <span class="icon-[heroicons--map-pin] size-3.5" aria-hidden="true"></span>{{ 'destinations.' + item.destination | transloco }}
            </a>
          }
          @if (item.location) {
            <span class="ui-badge bg-text/5 text-text">{{ item.location }}</span>
          }
          @if (item.durationDays; as days) {
            <span class="ui-badge bg-text/5 text-text">{{ 'tours.durationValue' | transloco: { days: days } }}</span>
          }
          <span class="ui-badge inline-flex items-center gap-1 bg-text/5 text-text">
            <span class="icon-[heroicons--calendar-days] size-3.5" aria-hidden="true"></span>{{ dates() }}
          </span>
        </div>
        <h1 class="mt-4 text-3xl font-extrabold text-text sm:text-4xl">{{ item.title }}</h1>

        <div class="mt-6 flex flex-col gap-10 lg:flex-row">
          <div class="flex min-w-0 flex-1 flex-col gap-10">
            <div>
              @if (item.coverImageUrl) {
                <div class="relative aspect-video overflow-hidden rounded-lg">
                  <img [ngSrc]="media(item.coverImageUrl)" fill priority sizes="(min-width: 1024px) 60vw, 100vw" alt="" class="object-cover" />
                </div>
              }
              <div class="prose prose-neutral dark:prose-invert mt-8 max-w-none" [innerHTML]="item.content"></div>
              @if (isOwner()) {
                <div class="mt-8 flex flex-wrap gap-2">
                  <a [appSiteLink]="'/partner/tours/' + item.id" class="ui-btn-soft">
                    <span class="icon-[heroicons--pencil-square] size-4" aria-hidden="true"></span>{{ 'listing.edit' | transloco }}
                  </a>
                  <button type="button" class="ui-btn-soft" [disabled]="deleting()" (click)="remove(item)">{{ 'tours.deleteButton' | transloco }}</button>
                </div>
              }
            </div>

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
            <div class="flex flex-col gap-4 lg:sticky lg:top-20">
              <!-- Depends on who's signed in: hydrate early, not on interaction
                   (a replayed click on the anonymous "sign in" would leave). -->
              @defer (on idle; hydrate on idle) {
                <app-tour-booking [tour]="item" />
              }
              <app-contact-card
                [targetType]="targetType"
                [targetId]="item.id"
                [listingName]="item.title"
                [priceLabel]="priceLabel()"
                [ratingAverage]="item.ratingAverage"
                [reviewCount]="item.reviewCount"
                [cancellationPolicy]="item.cancellationPolicy"
                [phone]="item.phone"
                [whatsapp]="item.whatsapp"
                [telegram]="item.telegram"
              />
            </div>
          </aside>
        </div>
      </article>
    }
  `,
})
export class TourPage {
  readonly slug = input.required<string>();
  protected readonly targetType = ReviewTargetType.TOUR;

  private readonly auth = inject(AuthStore);
  private readonly http = inject(HttpClient);
  private readonly transloco = inject(TranslocoService);
  private readonly config = inject(APP_CONFIG);
  private readonly navigate = injectSiteNavigate();
  private readonly lang = injectLang();

  protected readonly tour = httpResource<Tour>(
    () => `/tours/${this.lang()}/${encodeURIComponent(this.slug())}`,
  );
  // The loaded tour, or undefined (value() throws while the resource is in error).
  private readonly item = computed(() => (this.tour.hasValue() ? this.tour.value() : undefined));
  protected readonly reviews = reviewsResource(
    this.targetType,
    computed(() => this.item()?.id),
  );
  protected readonly reviewsPage = computed(() => loaded(this.reviews));
  protected readonly notFound = computed(() => {
    const error = this.tour.error();
    return error instanceof HttpErrorResponse && error.status === 404;
  });
  protected readonly isOwner = computed(() => {
    const user = this.auth.user();
    return Boolean(user && this.item()?.userId === user.id);
  });
  protected readonly dates = computed(() => {
    const item = this.item();
    return item ? formatDateRange(this.lang(), item.startDate, item.endDate) : '';
  });
  protected readonly priceLabel = computed(() => {
    const price = this.item()?.price;
    return price
      ? this.transloco.translate('tours.priceValue', { price: formatAmount(this.lang(), price) })
      : null;
  });
  protected readonly deleting = signal(false);

  constructor() {
    const seo = inject(SeoService);
    bindPageMeta(() => {
      const item = this.item();
      if (!item) return undefined;
      const path = `/tours/${item.slug}`;
      const description =
        item.excerpt ||
        plainText(item.content, 160) ||
        this.transloco.translate('tours.metaDescription');
      const image = item.coverImageUrl ? this.media(item.coverImageUrl) : null;
      return {
        title: item.title,
        description,
        image,
        alternates: null,
        jsonLd: [
          touristTrip({
            url: seo.pageUrl(path),
            name: item.title,
            description,
            image,
            startDate: item.startDate,
            endDate: item.endDate,
            place: item.destination
              ? this.transloco.translate(`destinations.${item.destination}`)
              : item.location,
            price: item.price,
            available: item.startDate >= todayInBishkek(),
            siteUrl: this.config.siteUrl,
          }),
          seo.breadcrumbs([
            { name: this.transloco.translate('tours.heading'), path: '/tours' },
            { name: item.title, path },
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
    return resolveMediaUrl(this.config.apiBaseUrl, url);
  }

  protected refresh(): void {
    this.tour.reload();
    this.reviews.reload();
  }

  protected async remove(item: Tour): Promise<void> {
    if (!window.confirm(this.transloco.translate('tours.deleteConfirm'))) return;
    this.deleting.set(true);
    try {
      await firstValueFrom(this.http.delete(`/tours/${item.id}`));
      await this.navigate('/tours');
    } finally {
      this.deleting.set(false);
    }
  }
}
