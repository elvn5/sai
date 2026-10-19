import {
  type BlogPost,
  Destination,
  type Hotel,
  type Outing,
  type Paginated,
  SeoPage,
  type Tour,
} from '@/core/api/types';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import { todayInBishkek } from '@/core/lib/format';
import { FEATURED_DESTINATIONS } from '@/core/lib/listing-options';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { usePageChrome } from '@/core/lib/page-chrome';
import { loaded } from '@/core/lib/resource';
import { bindPageMeta, pageSeoResource } from '@/core/seo/seo.service';
import { AppLogo } from '@/ui/app-logo';
import { DestinationSearch } from '@/ui/destination-search';
import { HotelCard } from '@/ui/listing/hotel-card';
import { TourCard } from '@/ui/listing/tour-card';
import { MountainScene } from '@/ui/mountain-scene';
import { OutingCard } from '@/ui/outing-card';
import { SiteLink } from '@/ui/site-link';
import { NgOptimizedImage } from '@angular/common';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

const NEWS_LIMIT = 3;
const TOP_HOTELS_LIMIT = 3;
const OUTINGS_LIMIT = 4;
const TOURS_LIMIT = 6;

const QUICK_LINKS = [
  { path: '/hotels', icon: 'icon-[heroicons--building-office-2] icon-anim-bounce', key: 'hotels' },
  { path: '/tours', icon: 'icon-[heroicons--map] icon-anim-flip', key: 'tours' },
  { path: '/local-guides', icon: 'icon-[heroicons--user-group] icon-anim-pop', key: 'localGuides' },
] as const;

// The home page (the feed): the nearest tours, who's heading out this
// weekend, where to go, where to stay and the latest news, all rendered on the server (the cached
// page is at most minutes old, so the outings' seats stay close enough).
@Component({
  selector: 'app-home-page',
  imports: [
    AppLogo,
    DestinationSearch,
    MountainScene,
    HotelCard,
    NgOptimizedImage,
    OutingCard,
    SiteLink,
    TourCard,
    TranslocoPipe,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="relative isolate -mt-[3.8125rem] overflow-hidden pt-[3.8125rem] lg:mt-0 lg:pt-0">
      <app-mountain-scene palette="sunset" [lake]="false" [parallax]="true" />
      <div class="absolute inset-0 -z-10 bg-linear-to-b from-black/40 via-black/15 via-55% to-bg"></div>
      <div class="ui-container pt-14 pb-20 text-white sm:pt-16 sm:pb-24 lg:text-center">
        <span class="motion-enter hidden size-14 items-center justify-center rounded-2xl bg-brand-600 lg:mx-auto lg:flex">
          <app-logo class="size-9 text-white" />
        </span>
        <h1 class="motion-enter motion-enter-1 text-3xl font-extrabold tracking-tight sm:text-[2.875rem] lg:mt-3.5">{{ 'brand.name' | transloco }}</h1>
        <p class="motion-enter motion-enter-2 mt-2 max-w-xl text-[0.9rem] text-white/92 sm:text-[1.0625rem] lg:mx-auto">{{ 'home.tagline' | transloco }}</p>
        <app-destination-search class="motion-enter motion-enter-3 mt-6 max-w-[38.75rem] lg:mx-auto" />
        <p class="motion-enter motion-enter-4 mt-4 hidden flex-wrap items-center justify-center gap-3 text-[0.8rem] text-white/90 sm:flex">
          <span>{{ 'search.popular' | transloco }}</span>
          @for (destination of popular; track destination) {
            <a [appSiteLink]="'/tours?destination=' + destination" class="underline underline-offset-2 hover:text-white">{{ 'destinations.' + destination | transloco }}</a>
          }
        </p>
      </div>
    </section>

    <nav class="ui-container relative z-10 -mt-[2.125rem]" [attr.aria-label]="'brand.name' | transloco">
      <ul class="motion-enter motion-enter-4 grid grid-cols-1 gap-2 sm:grid-cols-3 sm:gap-3.5">
        @for (link of quickLinks; track link.key) {
          <li>
            <a [appSiteLink]="link.path" class="group flex h-full items-center gap-3 rounded-card border border-border bg-surface p-3 shadow-[0_1px_2px_rgb(15_23_42/0.06)] motion-lift sm:p-4">
              <span class="flex size-[2.375rem] shrink-0 items-center justify-center rounded-full bg-brand-500/10 sm:size-[2.875rem]">
                <span [class]="link.icon + ' size-5 text-brand-600 sm:size-[1.375rem] dark:text-brand-300'" aria-hidden="true"></span>
              </span>
              <span class="min-w-0">
                <span class="block text-sm font-semibold text-text sm:text-base">{{ 'home.quick.' + link.key + '.title' | transloco }}</span>
                <span class="block text-xs text-text-muted sm:text-[0.8125rem]">{{ 'home.quick.' + link.key + '.description' | transloco }}</span>
              </span>
              <span class="icon-[heroicons--arrow-right] icon-anim-nudge ml-auto size-4 shrink-0 text-text-muted" aria-hidden="true"></span>
            </a>
          </li>
        }
      </ul>
    </nav>

    @if (upcomingTours().length > 0) {
      <section class="ui-container pt-10 sm:pt-12" aria-labelledby="home-tours">
        <div class="mb-3.5 flex items-center justify-between gap-4">
          <div>
            <h2 id="home-tours" class="text-xl font-bold tracking-tight text-text">{{ 'home.toursHeading' | transloco }}</h2>
            <p class="text-[0.8125rem] text-text-muted">{{ 'home.toursSubheading' | transloco }}</p>
          </div>
          <a appSiteLink="/tours" class="group flex shrink-0 items-center gap-1 text-[0.84rem] font-medium text-brand-700 dark:text-brand-300">
            {{ 'home.viewAllTours' | transloco }}<span class="icon-[heroicons--arrow-right] icon-anim-nudge size-4" aria-hidden="true"></span>
          </a>
        </div>
        <ul class="motion-reveal grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          @for (tour of upcomingTours(); track tour.id) {
            <li><app-tour-card [tour]="tour" /></li>
          }
        </ul>
      </section>
    }

    @if (upcoming().length > 0) {
      <section class="ui-container pt-12" aria-labelledby="home-outings">
        <div class="mb-3.5 flex items-center justify-between gap-4">
          <div>
            <h2 id="home-outings" class="text-xl font-bold tracking-tight text-text">{{ 'home.outingsHeading' | transloco }}</h2>
            <p class="text-[0.8125rem] text-text-muted">{{ 'home.outingsSubheading' | transloco }}</p>
          </div>
          <a appSiteLink="/outings" class="group flex shrink-0 items-center gap-1 text-[0.84rem] font-medium text-brand-700 dark:text-brand-300">
            {{ 'home.viewAllOutings' | transloco }}<span class="icon-[heroicons--arrow-right] icon-anim-nudge size-4" aria-hidden="true"></span>
          </a>
        </div>
        <ul class="motion-reveal grid grid-cols-1 gap-3 sm:grid-cols-2">
          @for (outing of upcoming(); track outing.id) {
            <li><app-outing-card [outing]="outing" /></li>
          }
        </ul>
      </section>
    }

    <section class="ui-container py-12" aria-labelledby="home-destinations">
      <h2 id="home-destinations" class="text-xl font-bold tracking-tight text-text">{{ 'home.destinationsHeading' | transloco }}</h2>
      <p class="text-[0.8125rem] text-text-muted">{{ 'home.destinationsSubheading' | transloco }}</p>
      <ul class="motion-reveal mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        @for (destination of destinations; track destination) {
          <li class="group flex flex-col rounded-card border border-border bg-linear-to-br from-brand-500/10 via-surface to-secondary-500/10 p-5 dark:from-brand-500/14 dark:to-secondary-500/12">
            <h3 class="flex items-center gap-1.5 text-base font-semibold text-text">
              <span class="icon-[heroicons--map-pin] icon-anim-bounce size-[1.15rem] text-brand-600 dark:text-brand-300" aria-hidden="true"></span>
              {{ 'destinations.' + destination | transloco }}
            </h3>
            <p class="mt-1.5 flex-1 text-[0.8125rem] text-text-muted">{{ 'home.destinationBlurbs.' + destination | transloco }}</p>
            <p class="mt-3 flex flex-wrap gap-x-3 gap-y-1 text-[0.8125rem] font-medium">
              <a [appSiteLink]="'/hotels?destination=' + destination" class="text-brand-700 hover:underline dark:text-brand-300">{{ 'nav.hotels' | transloco }}</a>
              <a [appSiteLink]="'/tours?destination=' + destination" class="text-brand-700 hover:underline dark:text-brand-300">{{ 'nav.tours' | transloco }}</a>
              <a [appSiteLink]="'/local-guides?destination=' + destination" class="text-brand-700 hover:underline dark:text-brand-300">{{ 'nav.localGuides' | transloco }}</a>
            </p>
          </li>
        }
      </ul>
    </section>

    @if (topHotels().length > 0) {
      <section class="ui-container pb-12" aria-labelledby="home-hotels">
        <div class="mb-3.5 flex items-center justify-between gap-4">
          <h2 id="home-hotels" class="text-xl font-bold tracking-tight text-text">{{ 'home.topHotels' | transloco }}</h2>
          <a appSiteLink="/hotels" class="group flex shrink-0 items-center gap-1 text-[0.84rem] font-medium text-brand-700 dark:text-brand-300">
            {{ 'home.viewAllHotels' | transloco }}<span class="icon-[heroicons--arrow-right] icon-anim-nudge size-4" aria-hidden="true"></span>
          </a>
        </div>
        <ul class="motion-reveal grid grid-cols-1 gap-4 sm:grid-cols-3">
          @for (hotel of topHotels(); track hotel.id) {
            <li><app-hotel-card [hotel]="hotel" /></li>
          }
        </ul>
      </section>
    }

    <section class="ui-container pb-16" aria-labelledby="home-news">
      <div class="mb-3.5 flex items-center justify-between gap-4">
        <h2 id="home-news" class="text-xl font-bold tracking-tight text-text">{{ 'home.latestNews' | transloco }}</h2>
        <a appSiteLink="/blog" class="group flex shrink-0 items-center gap-1 text-[0.84rem] font-medium text-brand-700 dark:text-brand-300">
          {{ 'home.viewAllPosts' | transloco }}<span class="icon-[heroicons--arrow-right] icon-anim-nudge size-4" aria-hidden="true"></span>
        </a>
      </div>
      @if (latestPosts().length > 0) {
        <ul class="motion-reveal flex flex-col gap-3.5">
          @for (post of latestPosts(); track post.id) {
            <li class="ui-card group overflow-hidden motion-lift">
              <a [appSiteLink]="'/blog/' + post.slug" class="flex flex-col gap-4 p-3.5 sm:flex-row">
                @if (post.coverImageUrl) {
                  <div class="relative aspect-video w-full shrink-0 overflow-hidden rounded-[0.625rem] sm:w-56">
                    <img [ngSrc]="media(post.coverImageUrl)" fill sizes="(min-width: 1280px) 18vw, (min-width: 640px) 35vw, 100vw" alt="" class="motion-zoom object-cover" />
                  </div>
                }
                <div class="flex flex-col justify-center">
                  <h3 class="text-[1.0625rem] font-semibold text-text">{{ post.title }}</h3>
                  @if (post.excerpt) {
                    <p class="mt-1.5 text-sm text-text-muted">{{ post.excerpt }}</p>
                  }
                </div>
              </a>
            </li>
          }
        </ul>
      } @else if (posts.hasValue()) {
        <p class="text-text-muted">{{ 'blog.empty' | transloco }}</p>
      }
    </section>
  `,
})
export class HomePage {
  private readonly config = inject(APP_CONFIG);
  private readonly lang = injectLang();
  protected readonly quickLinks = QUICK_LINKS;
  protected readonly destinations = FEATURED_DESTINATIONS;
  // Hero links under the search: the most asked-for places.
  protected readonly popular = [Destination.SONG_KOL, Destination.KARAKOL, Destination.ALA_ARCHA];

  protected readonly posts = httpResource<BlogPost[]>(() => ({
    url: '/blog',
    params: { locale: this.lang() },
  }));
  private readonly hotels = httpResource<Paginated<Hotel>>(() => ({
    url: '/hotels',
    params: { sort: 'rating', limit: TOP_HOTELS_LIMIT },
  }));
  private readonly tours = httpResource<Tour[]>(() => ({
    url: '/tours',
    params: { locale: this.lang() },
  }));
  private readonly outings = httpResource<Outing[]>(() => ({
    url: '/outings',
    params: { limit: OUTINGS_LIMIT },
  }));

  // A section whose request failed is left out: the rest of the page stands.
  protected readonly latestPosts = computed(() => (loaded(this.posts) ?? []).slice(0, NEWS_LIMIT));
  protected readonly topHotels = computed(() => loaded(this.hotels)?.items ?? []);
  protected readonly upcoming = computed(() => loaded(this.outings) ?? []);
  // Tours still ahead, soonest first.
  protected readonly upcomingTours = computed(() => {
    const today = todayInBishkek();
    return (loaded(this.tours) ?? [])
      .filter((tour) => tour.startDate >= today)
      .sort((a, b) => a.startDate.localeCompare(b.startDate))
      .slice(0, TOURS_LIMIT);
  });

  constructor() {
    // The hero runs under a see-through header on phones.
    usePageChrome('overlayHeader');
    const transloco = inject(TranslocoService);
    const overrides = pageSeoResource(SeoPage.HOME);
    bindPageMeta(() => {
      // Wait for the overrides (or their absence) so the server renders the final <head>.
      if (overrides.isLoading()) return undefined;
      const custom = overrides.hasValue() ? overrides.value() : null;
      return {
        title: custom?.title || undefined,
        description: custom?.description || transloco.translate('home.tagline'),
        image: custom?.ogImageUrl ?? `${this.config.siteUrl}/media/hero-mountain-1920.webp`,
      };
    });
  }

  protected media(url: string): string {
    return resolveMediaUrl(this.config.apiBaseUrl, url);
  }
}
