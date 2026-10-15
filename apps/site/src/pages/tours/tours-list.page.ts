import { SeoPage, type Tour } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, formatDateRange } from '@/core/lib/format';
import { DESTINATION_OPTIONS } from '@/core/lib/listing-options';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { withQuery } from '@/core/lib/query-link';
import { loaded } from '@/core/lib/resource';
import { itemList } from '@/core/seo/json-ld';
import { SeoService, bindPageMeta, pageSeoResource } from '@/core/seo/seo.service';
import { LoadError } from '@/ui/load-error';
import { RatingStars } from '@/ui/rating-stars';
import { SiteLink } from '@/ui/site-link';
import { NgOptimizedImage } from '@angular/common';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

// Tours in the current language (each written in one language, with its
// own slug), filtered by destination through links (?destination=, not
// indexed).
@Component({
  selector: 'app-tours-list-page',
  imports: [LoadError, NgOptimizedImage, RatingStars, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="mx-auto max-w-4xl px-4 py-12">
      <div class="mb-8 flex items-center justify-between gap-4">
        <h1 class="text-3xl font-extrabold text-text">{{ 'tours.heading' | transloco }}</h1>
        @if (auth.isPartner()) {
          <a appSiteLink="/tours/new" class="ui-btn-primary">
            <span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>{{ 'tours.addButton' | transloco }}
          </a>
        }
      </div>

      <nav class="mb-8 flex flex-wrap gap-2" [attr.aria-label]="'listing.form.destinationLabel' | transloco">
        <a class="ui-pill" appSiteLink="/tours" [attr.aria-current]="destination() ? null : 'page'">{{ 'listing.filters.allDestinations' | transloco }}</a>
        @for (option of destinations; track option.value) {
          <a class="ui-pill" [appSiteLink]="filterLink(option.value)" [attr.aria-current]="destination() === option.value ? 'page' : null">{{ option.labelKey | transloco }}</a>
        }
      </nav>

      @if (tours.error()) {
        <app-load-error (retry)="tours.reload()" />
      } @else if (tours.value(); as list) {
        @if (list.length > 0) {
          <ul class="flex flex-col gap-4">
            @for (tour of list; track tour.id; let first = $first) {
              <li class="ui-card overflow-hidden transition-shadow hover:shadow-md">
                <a [appSiteLink]="'/tours/' + tour.slug" class="flex flex-col gap-4 p-4 sm:flex-row">
                  @if (tour.coverImageUrl) {
                    <div class="relative aspect-video w-full shrink-0 overflow-hidden rounded-lg sm:w-56">
                      <img [ngSrc]="media(tour.coverImageUrl)" fill [priority]="first" sizes="(min-width: 640px) 25vw, 100vw" alt="" class="object-cover" />
                    </div>
                  }
                  <div class="flex flex-1 flex-col justify-center">
                    <div class="mb-2 flex flex-wrap gap-2">
                      @if (tour.destination || tour.location) {
                        <span class="ui-badge inline-flex items-center gap-1 bg-secondary-500/15 text-secondary-800 dark:text-secondary-200">
                          <span class="icon-[heroicons--map-pin] size-3.5" aria-hidden="true"></span>
                          @if (tour.destination) { {{ 'destinations.' + tour.destination | transloco }} } @else { {{ tour.location }} }
                        </span>
                      }
                      @if (tour.cancellationPolicy === 'free') {
                        <span class="ui-badge bg-green-500/15 text-green-800 dark:text-green-300">{{ 'listing.cancellation.free' | transloco }}</span>
                      }
                      @if (tour.durationDays; as days) {
                        <span class="ui-badge bg-text/5 text-text">{{ 'tours.durationValue' | transloco: { days: days } }}</span>
                      }
                      <span class="ui-badge inline-flex items-center gap-1 bg-text/5 text-text">
                        <span class="icon-[heroicons--calendar-days] size-3.5" aria-hidden="true"></span>{{ dates(tour) }}
                      </span>
                    </div>
                    <h2 class="text-xl font-semibold text-text">{{ tour.title }}</h2>
                    @if (tour.excerpt) {
                      <p class="mt-2 text-text-muted">{{ tour.excerpt }}</p>
                    }
                    <div class="mt-2 flex flex-wrap items-center justify-between gap-2">
                      @if (tour.reviewCount > 0) {
                        <span class="flex items-center gap-1.5 text-sm">
                          <app-rating-stars [value]="tour.ratingAverage" />
                          <span class="text-text-muted">{{ 'reviews.count' | transloco: { count: tour.reviewCount } }}</span>
                        </span>
                      }
                      @if (tour.price) {
                        <p class="font-semibold text-brand-700 dark:text-brand-300">{{ 'tours.priceValue' | transloco: { price: amount(tour.price) } }}</p>
                      }
                    </div>
                  </div>
                </a>
              </li>
            }
          </ul>
        } @else {
          <p class="text-text-muted">{{ 'tours.empty' | transloco }}</p>
        }
      }
    </section>
  `,
})
export class ToursListPage {
  readonly destination = input<string | undefined>(undefined);
  protected readonly auth = inject(AuthStore);
  private readonly apiBaseUrl = inject(APP_CONFIG).apiBaseUrl;
  private readonly transloco = inject(TranslocoService);
  private readonly lang = injectLang();
  protected readonly destinations = DESTINATION_OPTIONS;

  protected readonly tours = httpResource<Tour[]>(() => ({
    url: '/tours',
    params: {
      locale: this.lang(),
      ...(this.destination() ? { destination: this.destination() as string } : {}),
    },
  }));

  constructor() {
    const seo = inject(SeoService);
    const overrides = pageSeoResource(SeoPage.TOURS);
    bindPageMeta(() => {
      if (overrides.isLoading() || this.tours.isLoading()) return undefined;
      const custom = overrides.hasValue() ? overrides.value() : null;
      const heading = this.transloco.translate('tours.heading');
      return {
        title: custom?.title || heading,
        description: custom?.description || this.transloco.translate('tours.metaDescription'),
        image: custom?.ogImageUrl,
        robots: this.destination() ? 'noindex, follow' : undefined,
        // Written per language: each language lists its own tours.
        jsonLd: [
          itemList(
            (loaded(this.tours) ?? []).map((tour) => ({
              url: seo.pageUrl(`/tours/${tour.slug}`),
              name: tour.title,
            })),
          ),
          seo.breadcrumbs([{ name: heading, path: '/tours' }]),
        ],
      };
    });
  }

  protected filterLink(destination: string): string {
    return withQuery('/tours', { destination });
  }

  protected media(url: string): string {
    return resolveMediaUrl(this.apiBaseUrl, url);
  }

  protected dates(tour: Tour): string {
    return formatDateRange(this.lang(), tour.startDate, tour.endDate);
  }

  protected amount(price: number): string {
    return formatAmount(this.lang(), price);
  }
}
