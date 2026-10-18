import { type LocalGuide, ReviewTargetType } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount } from '@/core/lib/format';
import { GUIDE_SPECIALIZATION_OPTIONS, pickOptions } from '@/core/lib/listing-options';
import { trackListingView } from '@/core/lib/listing-stats';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { loaded } from '@/core/lib/resource';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { guideBusiness, plainText } from '@/core/seo/json-ld';
import { SeoService, bindPageMeta } from '@/core/seo/seo.service';
import { NotFoundPage } from '@/pages/not-found/not-found.page';
import { injectConfirm } from '@/ui/confirm-dialog';
import { ContactCard } from '@/ui/listing/contact-card';
import { PostsFeed } from '@/ui/listing/posts-feed';
import { reviewsResource } from '@/ui/listing/reviews';
import { ReviewsSection } from '@/ui/listing/reviews-section';
import { LoadError } from '@/ui/load-error';
import { PhotoGrid } from '@/ui/photo-grid';
import { RatingStars } from '@/ui/rating-stars';
import { SiteLink } from '@/ui/site-link';
import { NgOptimizedImage } from '@angular/common';
import { HttpClient, HttpErrorResponse, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, input, signal } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// A local guide's profile: languages, areas, specializations, bio, photos,
// posts and reviews, with the contact box at the side.
@Component({
  selector: 'app-local-guide-page',
  imports: [
    ContactCard,
    LoadError,
    NgOptimizedImage,
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
    } @else if (guide.error()) {
      <div class="mx-auto max-w-6xl px-4 py-12"><app-load-error (retry)="guide.reload()" /></div>
    } @else if (guide.value(); as item) {
      <article class="mx-auto max-w-6xl px-4 py-10">
        <a appSiteLink="/local-guides" class="text-sm text-brand-700 hover:underline dark:text-brand-300">&larr; {{ 'localGuides.heading' | transloco }}</a>

        <div class="mt-6 flex flex-col gap-6 sm:flex-row sm:items-center">
          @if (avatar(); as url) {
            <img [ngSrc]="url" width="112" height="112" priority alt="" class="size-28 shrink-0 rounded-full object-cover ring-4 ring-brand-500/20" />
          } @else {
            <span class="flex size-28 shrink-0 items-center justify-center rounded-full bg-linear-to-br from-brand-500 to-secondary-500 text-4xl font-bold text-white" aria-hidden="true">{{ item.name.charAt(0).toUpperCase() }}</span>
          }
          <div class="flex-1">
            <h1 class="text-3xl font-extrabold text-text sm:text-4xl">{{ item.name }}</h1>
            @if (item.headline) {
              <p class="mt-1 text-lg text-text-muted">{{ item.headline }}</p>
            }
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
              <a [appSiteLink]="'/local-guides/' + item.slug + '/edit'" class="ui-btn-soft">
                <span class="icon-[heroicons--pencil-square] size-4" aria-hidden="true"></span>{{ 'listing.edit' | transloco }}
              </a>
              <button type="button" class="ui-btn-danger" [disabled]="deleting()" (click)="remove(item)">
                <span class="icon-[heroicons--trash] size-4" aria-hidden="true"></span>{{ 'listing.delete' | transloco }}
              </button>
            </div>
          }
        </div>

        <div class="mt-8 flex flex-col gap-10 lg:flex-row">
          <div class="flex min-w-0 flex-1 flex-col gap-10">
            <section class="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div class="ui-card p-4">
                <h2 class="text-sm font-medium text-text-muted">{{ 'localGuides.languagesLabel' | transloco }}</h2>
                <ul class="mt-2 flex flex-wrap gap-1.5">
                  @for (language of item.languages; track language) {
                    <li class="ui-badge bg-text/5 text-text">{{ 'localGuides.languages.' + language | transloco }}</li>
                  }
                </ul>
              </div>
              <div class="ui-card p-4">
                <h2 class="text-sm font-medium text-text-muted">{{ 'localGuides.destinationsLabel' | transloco }}</h2>
                <ul class="mt-2 flex flex-wrap gap-1.5">
                  @for (destination of item.destinations; track destination) {
                    <li>
                      <a [appSiteLink]="'/local-guides?destination=' + destination" class="ui-badge bg-secondary-500/15 text-secondary-800 dark:text-secondary-200">{{ 'destinations.' + destination | transloco }}</a>
                    </li>
                  }
                </ul>
              </div>
              <div class="ui-card p-4 sm:col-span-2">
                <h2 class="text-sm font-medium text-text-muted">{{ 'localGuides.specializationsLabel' | transloco }}</h2>
                <ul class="mt-2 flex flex-wrap gap-x-5 gap-y-2">
                  @for (option of specializations(); track option.value) {
                    <li class="flex items-center gap-1.5 text-text">
                      <span [class]="option.icon + ' size-5 text-brand-600 dark:text-brand-300'" aria-hidden="true"></span>{{ option.labelKey | transloco }}
                    </li>
                  }
                </ul>
                <div class="mt-3 flex flex-wrap gap-4 text-sm text-text-muted">
                  @if (item.experienceYears; as years) {
                    <span class="flex items-center gap-1">
                      <span class="icon-[heroicons--academic-cap] size-4" aria-hidden="true"></span>{{ 'localGuides.experienceValue' | transloco: { years: years } }}
                    </span>
                  }
                  @if (item.hasVehicle) {
                    <span class="flex items-center gap-1">
                      <span class="icon-[heroicons--truck] size-4" aria-hidden="true"></span>{{ 'localGuides.vehicleBadge' | transloco }}
                    </span>
                  }
                </div>
              </div>
            </section>

            <section aria-labelledby="bio-heading">
              <h2 id="bio-heading" class="text-2xl font-bold text-text">{{ 'localGuides.about' | transloco }}</h2>
              <div class="prose prose-neutral dark:prose-invert mt-4 max-w-none" [innerHTML]="item.bio"></div>
            </section>

            @if (item.photos.length > 0) {
              <section aria-labelledby="photos-heading">
                <h2 id="photos-heading" class="mb-4 text-2xl font-bold text-text">{{ 'listing.photos.heading' | transloco }}</h2>
                <app-photo-grid [photos]="item.photos" [alt]="item.name" />
              </section>
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
export class LocalGuidePage {
  readonly slug = input.required<string>();
  protected readonly targetType = ReviewTargetType.LOCAL_GUIDE;

  private readonly auth = inject(AuthStore);
  private readonly http = inject(HttpClient);
  private readonly transloco = inject(TranslocoService);
  private readonly confirm = injectConfirm();
  private readonly apiBaseUrl = inject(APP_CONFIG).apiBaseUrl;
  private readonly navigate = injectSiteNavigate();
  private readonly lang = injectLang();

  protected readonly guide = httpResource<LocalGuide>(
    () => `/local-guides/${encodeURIComponent(this.slug())}`,
  );
  // The loaded guide, or undefined (value() throws while the resource is in error).
  private readonly item = computed(() => (this.guide.hasValue() ? this.guide.value() : undefined));
  protected readonly reviews = reviewsResource(
    this.targetType,
    computed(() => this.item()?.id),
  );
  protected readonly reviewsPage = computed(() => loaded(this.reviews));
  protected readonly notFound = computed(() => {
    const error = this.guide.error();
    return error instanceof HttpErrorResponse && error.status === 404;
  });
  protected readonly isOwner = computed(() => {
    const user = this.auth.user();
    return Boolean(user && this.item()?.userId === user.id);
  });
  protected readonly avatar = computed(() => {
    const url = this.item()?.avatarUrl;
    return url ? resolveMediaUrl(this.apiBaseUrl, url) : null;
  });
  protected readonly specializations = computed(() =>
    pickOptions(GUIDE_SPECIALIZATION_OPTIONS, this.item()?.specializations ?? []),
  );
  protected readonly priceLabel = computed(() => {
    const price = this.item()?.pricePerDay;
    return price != null
      ? this.transloco.translate('listing.pricePerDay', { price: formatAmount(this.lang(), price) })
      : null;
  });
  protected readonly deleting = signal(false);

  constructor() {
    const seo = inject(SeoService);
    bindPageMeta(() => {
      const item = this.item();
      if (!item || this.reviews.isLoading()) return undefined;
      const path = `/local-guides/${item.slug}`;
      const imagePath = item.coverImageUrl ?? item.avatarUrl;
      const image = imagePath ? resolveMediaUrl(this.apiBaseUrl, imagePath) : null;
      const description: string =
        item.headline ?? this.transloco.translate<string>('localGuides.metaDescription');
      return {
        title: this.transloco.translate<string>('localGuides.detailTitle', { name: item.name }),
        description,
        image,
        type: 'profile',
        jsonLd: [
          guideBusiness({
            url: seo.pageUrl(path),
            name: item.name,
            description: item.headline ?? plainText(item.bio),
            image,
            areas: item.destinations.map((destination) =>
              this.transloco.translate(`destinations.${destination}`),
            ),
            pricePerDay: item.pricePerDay,
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
            { name: this.transloco.translate('localGuides.heading'), path: '/local-guides' },
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

  protected refresh(): void {
    this.guide.reload();
    this.reviews.reload();
  }

  protected async remove(item: LocalGuide): Promise<void> {
    if (!(await this.confirm({ title: 'localGuides.deleteConfirm', danger: true }))) return;
    this.deleting.set(true);
    try {
      await firstValueFrom(this.http.delete(`/local-guides/${item.id}`));
      await this.navigate('/local-guides');
    } finally {
      this.deleting.set(false);
    }
  }
}
