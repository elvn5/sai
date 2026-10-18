import { ReviewTargetType, type Tour } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, formatDateRange, todayInBishkek } from '@/core/lib/format';
import { trackListingView } from '@/core/lib/listing-stats';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { usePageChrome } from '@/core/lib/page-chrome';
import { loaded } from '@/core/lib/resource';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { tourCoverTransition } from '@/core/lib/view-transitions';
import { Notify } from '@/core/notify/notify';
import { plainText, touristTrip } from '@/core/seo/json-ld';
import { SeoService, bindPageMeta } from '@/core/seo/seo.service';
import { NotFoundPage } from '@/pages/not-found/not-found.page';
import { TourBooking } from '@/pages/tours/tour-booking';
import { injectConfirm } from '@/ui/confirm-dialog';
import { ContactCard } from '@/ui/listing/contact-card';
import { PostsFeed } from '@/ui/listing/posts-feed';
import { reviewsResource } from '@/ui/listing/reviews';
import { ReviewsSection } from '@/ui/listing/reviews-section';
import { LoadError } from '@/ui/load-error';
import { SharedTransition } from '@/ui/shared-transition';
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
    SharedTransition,
    SiteLink,
    TourBooking,
    TranslocoPipe,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (notFound()) {
      <app-not-found-page />
    } @else if (tour.error()) {
      <div class="ui-container py-12"><app-load-error (retry)="tour.reload()" /></div>
    } @else if (item(); as item) {
      <article class="pb-14 lg:pt-4">
        <!-- Phones: the photo first, edge to edge, with round buttons on it. -->
        <div class="relative aspect-[3/2] overflow-hidden bg-slate-200 lg:hidden dark:bg-slate-800">
          @if (item.coverImageUrl) {
            <div class="absolute inset-0" [appSharedTransition]="coverTransition(item.id)" [sharedAlways]="true" [sharedMedia]="'(width < 64rem)'">
              <img [ngSrc]="media(item.coverImageUrl)" fill priority sizes="100vw" alt="" class="object-cover" />
            </div>
          }
          <div class="absolute inset-x-3 top-3 flex justify-between">
            <a appSiteLink="/tours" class="flex size-10 items-center justify-center rounded-full bg-white/92 text-slate-900 shadow-sm" [attr.aria-label]="'tours.heading' | transloco">
              <span class="icon-[heroicons--chevron-left] size-5" aria-hidden="true"></span>
            </a>
            <button type="button" class="flex size-10 items-center justify-center rounded-full bg-white/92 text-slate-900 shadow-sm" [attr.aria-label]="'listing.share' | transloco" (click)="share(item)">
              <span class="icon-[heroicons--share] size-5" aria-hidden="true"></span>
            </button>
          </div>
        </div>

        <div class="ui-container relative -mt-4 rounded-t-2xl bg-bg pt-4 lg:mt-0 lg:rounded-none lg:pt-0">
          <div class="hidden items-center justify-between gap-3 lg:flex">
            <nav class="ui-crumbs" [attr.aria-label]="'breadcrumbs.label' | transloco">
              <a appSiteLink="/tours" class="inline-flex items-center gap-1"><span class="icon-[heroicons--chevron-left] size-3" aria-hidden="true"></span>{{ 'tours.heading' | transloco }}</a>
              @if (item.destination) {
                <span class="opacity-50">/</span>
                <a [appSiteLink]="'/tours?destination=' + item.destination">{{ 'destinations.' + item.destination | transloco }}</a>
              }
            </nav>
            <button type="button" class="ui-btn-ghost" (click)="share(item)">
              <span class="icon-[heroicons--share] size-4" aria-hidden="true"></span>{{ 'listing.share' | transloco }}
            </button>
          </div>
          <h1 class="text-xl leading-tight font-extrabold tracking-tight text-text sm:text-[2rem] lg:mt-2.5">{{ item.title }}</h1>
          <div class="mt-2.5 flex flex-wrap items-center gap-1.5">
            @if (item.destination) {
              <a [appSiteLink]="'/tours?destination=' + item.destination" class="ui-badge ui-badge-dest"><span class="icon-[heroicons--map-pin]" aria-hidden="true"></span>{{ 'destinations.' + item.destination | transloco }}</a>
            } @else if (item.location) {
              <span class="ui-badge ui-badge-soft"><span class="icon-[heroicons--map-pin]" aria-hidden="true"></span>{{ item.location }}</span>
            }
            @if (item.durationDays; as days) {
              <span class="ui-badge ui-badge-soft"><span class="icon-[heroicons--clock]" aria-hidden="true"></span>{{ 'tours.durationValue' | transloco: { days: days } }}</span>
            }
            <span class="ui-badge ui-badge-soft"><span class="icon-[heroicons--calendar-days]" aria-hidden="true"></span>{{ dates() }}</span>
            @if (item.cancellationPolicy === 'free') {
              <span class="ui-badge ui-badge-success"><span class="icon-[heroicons--check]" aria-hidden="true"></span>{{ 'listing.cancellation.free' | transloco }}</span>
            }
            @if (item.reviewCount > 0) {
              <a href="#reviews" class="ml-1 inline-flex items-center gap-1 text-[0.8125rem]">
                <span class="icon-[heroicons--star-solid] size-4 text-secondary-400" aria-hidden="true"></span>
                <b class="text-text">{{ item.ratingAverage.toFixed(1) }}</b>
                <span class="text-text-muted">· {{ 'reviews.count' | transloco: { count: item.reviewCount } }}</span>
              </a>
            }
          </div>

          <div class="mt-4 flex flex-col gap-6 lg:grid lg:grid-cols-[minmax(0,1fr)_21.25rem] lg:items-start lg:gap-[1.625rem]">
            <div class="flex min-w-0 flex-col gap-8">
              <div>
                @if (item.coverImageUrl) {
                  <div class="relative hidden aspect-video overflow-hidden rounded-[0.875rem] lg:block" [appSharedTransition]="coverTransition(item.id)" [sharedAlways]="true" [sharedMedia]="'(width >= 64rem)'">
                    <img [ngSrc]="media(item.coverImageUrl)" fill priority sizes="(min-width: 1024px) 60vw, 100vw" alt="" class="object-cover" />
                  </div>
                }
                <nav class="ui-tabs sticky top-15 z-10 -mx-4 bg-bg/92 px-4 backdrop-blur-sm lg:mx-0 lg:mt-4 lg:px-0" [attr.aria-label]="'tours.sections' | transloco">
                  <a href="#about" class="ui-tab" [attr.aria-current]="section() === 'about' ? 'page' : null" (click)="section.set('about')">{{ 'tours.tabs.about' | transloco }}</a>
                  <a href="#reviews" class="ui-tab" [attr.aria-current]="section() === 'reviews' ? 'page' : null" (click)="section.set('reviews')">
                    {{ 'tours.tabs.reviews' | transloco }}
                    @if (item.reviewCount > 0) { · {{ item.reviewCount }} }
                  </a>
                </nav>
                <div id="about" class="prose prose-neutral dark:prose-invert mt-4 max-w-none scroll-mt-28" [innerHTML]="item.content"></div>
                @if (isOwner()) {
                  <div class="mt-6 flex flex-wrap gap-2">
                    <a [appSiteLink]="'/partner/tours/' + item.id" class="ui-btn-soft">
                      <span class="icon-[heroicons--pencil-square] size-4" aria-hidden="true"></span>{{ 'listing.edit' | transloco }}
                    </a>
                    <button type="button" class="ui-btn-danger" [disabled]="deleting()" (click)="remove(item)">
                      <span class="icon-[heroicons--trash] size-4" aria-hidden="true"></span>{{ 'tours.deleteButton' | transloco }}
                    </button>
                  </div>
                }
              </div>

              @defer (on viewport; hydrate on viewport) {
                <app-posts-feed [targetType]="targetType" [targetId]="item.id" [ownerId]="item.userId" />
              } @placeholder {
                <div></div>
              }
              <div id="reviews" class="scroll-mt-28">
                @defer (on viewport; hydrate on viewport) {
                  <app-reviews-section [targetType]="targetType" [targetId]="item.id" [ownerId]="item.userId" [page]="reviewsPage()" (changed)="refresh()" />
                } @placeholder {
                  <div class="ui-skeleton min-h-40"></div>
                }
              </div>
            </div>

            <aside>
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
                  [priceLabel]="null"
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
        </div>
      </article>
    }
  `,
})
export class TourPage {
  readonly slug = input.required<string>();
  // Route data (tour-prefetch.ts).
  readonly prefetched = input<Tour | null>(null);
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
  // The loaded tour, or the one fetched before navigating here
  // (tour-prefetch.ts) until then; undefined while the resource is in error
  // (value() throws).
  protected readonly item = computed(() => {
    if (this.tour.hasValue()) return this.tour.value();
    if (this.tour.error()) return undefined;
    const prefetched = this.prefetched();
    return prefetched?.slug === this.slug() ? prefetched : undefined;
  });
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
  // The tab under the title: description or reviews.
  protected readonly section = signal<'about' | 'reviews'>('about');
  private readonly confirm = injectConfirm();
  private readonly notify = inject(Notify);

  constructor() {
    // Phones: the photo with its own back button replaces the header.
    usePageChrome('immersive');
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

  // The cover the tour's card flew in from (motion.css).
  protected coverTransition(id: string): string {
    return tourCoverTransition(id);
  }

  protected refresh(): void {
    this.tour.reload();
    this.reviews.reload();
  }

  protected async remove(item: Tour): Promise<void> {
    if (
      !(await this.confirm({
        title: 'tours.deleteConfirm',
        text: 'notify.irreversible',
        danger: true,
      }))
    )
      return;
    this.deleting.set(true);
    try {
      await firstValueFrom(this.http.delete(`/tours/${item.id}`));
      this.notify.success({ title: 'notify.deleted' });
      await this.navigate('/tours');
    } catch {
      this.notify.error({ title: 'notify.deleteFailed', text: 'notify.tryAgain' });
    } finally {
      this.deleting.set(false);
    }
  }

  // The phone's share sheet, or the link copied.
  protected async share(item: Tour): Promise<void> {
    const url = location.href;
    if (navigator.share) {
      try {
        await navigator.share({ title: item.title, url });
      } catch {
        // Dismissed.
      }
      return;
    }
    try {
      await navigator.clipboard.writeText(url);
      this.notify.success({ title: 'notify.linkCopied' });
    } catch {
      this.notify.error({ title: 'notify.tryAgain' });
    }
  }
}
