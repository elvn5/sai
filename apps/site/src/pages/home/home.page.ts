import { type BlogPost, type Hotel, type Outing, type Paginated, SeoPage } from '@/core/api/types';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import { FEATURED_DESTINATIONS } from '@/core/lib/listing-options';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { loaded } from '@/core/lib/resource';
import { bindPageMeta, pageSeoResource } from '@/core/seo/seo.service';
import { HeroImage } from '@/pages/home/hero-image';
import { AppLogo } from '@/ui/app-logo';
import { HotelCard } from '@/ui/listing/hotel-card';
import { OutingCard } from '@/ui/outing-card';
import { SiteLink } from '@/ui/site-link';
import { NgOptimizedImage } from '@angular/common';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

const NEWS_LIMIT = 3;
const TOP_HOTELS_LIMIT = 3;
const OUTINGS_LIMIT = 4;

const QUICK_LINKS = [
  { path: '/hotels', icon: 'icon-[heroicons--building-office-2]', key: 'hotels' },
  { path: '/tours', icon: 'icon-[heroicons--map]', key: 'tours' },
  { path: '/local-guides', icon: 'icon-[heroicons--user-group]', key: 'localGuides' },
] as const;

// The home page: where to go, where to stay, who's heading out this
// weekend and the latest news, all rendered on the server (the cached
// page is at most minutes old, so the outings' seats stay close enough).
@Component({
  selector: 'app-home-page',
  imports: [AppLogo, HeroImage, HotelCard, NgOptimizedImage, OutingCard, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="relative isolate overflow-hidden px-4 py-24 text-center sm:py-32">
      <app-hero-image />
      <div class="absolute inset-0 -z-10 bg-linear-to-b from-black/70 via-black/50 to-bg"></div>
      <div class="mx-auto max-w-4xl">
        <app-logo class="mx-auto size-16 text-white" />
        <h1 class="mt-4 text-3xl font-extrabold text-white sm:text-4xl">{{ 'brand.name' | transloco }}</h1>
        <p class="mx-auto mt-3 max-w-xl text-white/90">{{ 'home.tagline' | transloco }}</p>
        <div class="mt-6 flex flex-wrap justify-center gap-3">
          <a appSiteLink="/hotels" class="ui-btn-primary px-5 py-2.5 text-base">{{ 'home.ctaHotels' | transloco }}</a>
          <a appSiteLink="/guides" class="ui-btn-soft bg-white/15 px-5 py-2.5 text-base text-white hover:bg-white/25">{{ 'home.ctaPrimary' | transloco }}</a>
        </div>
      </div>
    </section>

    <nav class="relative z-10 mx-auto -mt-10 max-w-4xl px-4" [attr.aria-label]="'brand.name' | transloco">
      <ul class="grid grid-cols-1 gap-3 sm:grid-cols-3">
        @for (link of quickLinks; track link.key) {
          <li>
            <a [appSiteLink]="link.path" class="flex h-full items-center gap-3 rounded-card border border-border bg-surface p-4 shadow-sm transition hover:shadow-md">
              <span class="flex size-11 shrink-0 items-center justify-center rounded-full bg-brand-500/10">
                <span [class]="link.icon + ' size-6 text-brand-600 dark:text-brand-300'" aria-hidden="true"></span>
              </span>
              <span>
                <span class="block font-semibold text-text">{{ 'home.quick.' + link.key + '.title' | transloco }}</span>
                <span class="block text-sm text-text-muted">{{ 'home.quick.' + link.key + '.description' | transloco }}</span>
              </span>
            </a>
          </li>
        }
      </ul>
    </nav>

    @if (upcoming().length > 0) {
      <section class="mx-auto max-w-4xl px-4 pt-14" aria-labelledby="home-outings">
        <div class="mb-4 flex items-center justify-between gap-4">
          <div>
            <h2 id="home-outings" class="text-xl font-bold text-text">{{ 'home.outingsHeading' | transloco }}</h2>
            <p class="mt-1 text-sm text-text-muted">{{ 'home.outingsSubheading' | transloco }}</p>
          </div>
          <a appSiteLink="/outings" class="shrink-0 text-sm text-brand-700 hover:underline dark:text-brand-300">{{ 'home.viewAllOutings' | transloco }}</a>
        </div>
        <ul class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          @for (outing of upcoming(); track outing.id) {
            <li><app-outing-card [outing]="outing" /></li>
          }
        </ul>
      </section>
    }

    <section class="mx-auto max-w-4xl px-4 py-14" aria-labelledby="home-destinations">
      <h2 id="home-destinations" class="text-xl font-bold text-text">{{ 'home.destinationsHeading' | transloco }}</h2>
      <p class="mt-1 text-sm text-text-muted">{{ 'home.destinationsSubheading' | transloco }}</p>
      <ul class="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        @for (destination of destinations; track destination) {
          <li class="flex flex-col rounded-card border border-border bg-linear-to-br from-brand-500/10 via-surface to-secondary-500/10 p-5">
            <h3 class="flex items-center gap-2 text-lg font-semibold text-text">
              <span class="icon-[heroicons--map-pin] size-5 text-brand-600 dark:text-brand-300" aria-hidden="true"></span>
              {{ 'destinations.' + destination | transloco }}
            </h3>
            <p class="mt-2 flex-1 text-sm text-text-muted">{{ 'home.destinationBlurbs.' + destination | transloco }}</p>
            <p class="mt-4 flex flex-wrap gap-x-4 gap-y-1 text-sm font-medium">
              <a [appSiteLink]="'/hotels?destination=' + destination" class="text-brand-700 hover:underline dark:text-brand-300">{{ 'nav.hotels' | transloco }}</a>
              <a [appSiteLink]="'/tours?destination=' + destination" class="text-brand-700 hover:underline dark:text-brand-300">{{ 'nav.tours' | transloco }}</a>
              <a [appSiteLink]="'/local-guides?destination=' + destination" class="text-brand-700 hover:underline dark:text-brand-300">{{ 'nav.localGuides' | transloco }}</a>
            </p>
          </li>
        }
      </ul>
    </section>

    @if (topHotels().length > 0) {
      <section class="mx-auto max-w-4xl px-4 pb-14" aria-labelledby="home-hotels">
        <div class="mb-6 flex items-center justify-between gap-4">
          <h2 id="home-hotels" class="text-xl font-bold text-text">{{ 'home.topHotels' | transloco }}</h2>
          <a appSiteLink="/hotels" class="shrink-0 text-sm text-brand-700 hover:underline dark:text-brand-300">{{ 'home.viewAllHotels' | transloco }}</a>
        </div>
        <ul class="grid grid-cols-1 gap-4 sm:grid-cols-3">
          @for (hotel of topHotels(); track hotel.id) {
            <li><app-hotel-card [hotel]="hotel" /></li>
          }
        </ul>
      </section>
    }

    <section class="mx-auto max-w-4xl px-4 pb-16" aria-labelledby="home-news">
      <div class="mb-6 flex items-center justify-between gap-4">
        <h2 id="home-news" class="text-xl font-bold text-text">{{ 'home.latestNews' | transloco }}</h2>
        <a appSiteLink="/blog" class="shrink-0 text-sm text-brand-700 hover:underline dark:text-brand-300">{{ 'home.viewAllPosts' | transloco }}</a>
      </div>
      @if (latestPosts().length > 0) {
        <ul class="flex flex-col gap-4">
          @for (post of latestPosts(); track post.id) {
            <li class="ui-card overflow-hidden transition-shadow hover:shadow-md">
              <a [appSiteLink]="'/blog/' + post.slug" class="flex flex-col gap-4 p-4 sm:flex-row">
                @if (post.coverImageUrl) {
                  <div class="relative aspect-video w-full shrink-0 overflow-hidden rounded-lg sm:w-56">
                    <img [ngSrc]="media(post.coverImageUrl)" fill sizes="(min-width: 1280px) 18vw, (min-width: 640px) 35vw, 100vw" alt="" class="object-cover" />
                  </div>
                }
                <div class="flex flex-col justify-center">
                  <h3 class="text-lg font-semibold text-text">{{ post.title }}</h3>
                  @if (post.excerpt) {
                    <p class="mt-2 text-text-muted">{{ post.excerpt }}</p>
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

  protected readonly posts = httpResource<BlogPost[]>(() => ({
    url: '/blog',
    params: { locale: this.lang() },
  }));
  private readonly hotels = httpResource<Paginated<Hotel>>(() => ({
    url: '/hotels',
    params: { sort: 'rating', limit: TOP_HOTELS_LIMIT },
  }));
  private readonly outings = httpResource<Outing[]>(() => ({
    url: '/outings',
    params: { limit: OUTINGS_LIMIT },
  }));

  // A section whose request failed is left out: the rest of the page stands.
  protected readonly latestPosts = computed(() => (loaded(this.posts) ?? []).slice(0, NEWS_LIMIT));
  protected readonly topHotels = computed(() => loaded(this.hotels)?.items ?? []);
  protected readonly upcoming = computed(() => loaded(this.outings) ?? []);

  constructor() {
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
