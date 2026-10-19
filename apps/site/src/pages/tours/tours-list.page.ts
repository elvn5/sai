import { type PublicAuthor, SeoPage, type Tour } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import {
  formatAmount,
  formatDate,
  formatDateRange,
  personName,
  todayInBishkek,
} from '@/core/lib/format';
import { DESTINATION_OPTIONS } from '@/core/lib/listing-options';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { FEW_PLACES, placesLeft, placesResource } from '@/core/lib/places';
import { injectQueryFilters } from '@/core/lib/query-filters';
import { withQuery } from '@/core/lib/query-link';
import { loaded } from '@/core/lib/resource';
import { tourCoverTransition } from '@/core/lib/view-transitions';
import { itemList } from '@/core/seo/json-ld';
import { SeoService, bindPageMeta, pageSeoResource } from '@/core/seo/seo.service';
import { TourAlertButton } from '@/pages/tours/tour-alert-button';
import { FavoriteButton } from '@/ui/favorite-button';
import { ListSkeleton } from '@/ui/list-skeleton';
import { LoadError } from '@/ui/load-error';
import { PersonAvatar } from '@/ui/person-avatar';
import { SharedTransition } from '@/ui/shared-transition';
import { SiteLink } from '@/ui/site-link';
import { StateBlock } from '@/ui/state-block';
import {
  CdkMenu,
  CdkMenuItem,
  CdkMenuItemCheckbox,
  CdkMenuItemRadio,
  CdkMenuTrigger,
} from '@angular/cdk/menu';
import { NgOptimizedImage } from '@angular/common';
import { httpResource } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  linkedSignal,
  signal,
} from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

const SORT_KEYS = ['soonest', 'price_asc', 'price_desc', 'rating'] as const;
type SortKey = (typeof SORT_KEYS)[number];

// Trip lengths the filter offers: one day, a weekend, longer.
const DURATIONS = ['1', '2-3', '4+'] as const;

// Destinations shown as pills before "More".
const VISIBLE_DESTINATIONS = 6;

// Tours shown at first, and added by each "Show more".
const PAGE_SIZE = 10;

const isDate = (value: string | undefined): boolean => /^\d{4}-\d{2}-\d{2}$/.test(value ?? '');

function matchesDuration(days: number | null, range: string): boolean {
  if (!days) return false;
  if (range === '1') return days === 1;
  if (range === '2-3') return days >= 2 && days <= 3;
  return days >= 4;
}

// Tours in the current language (each written in one language, with its
// own slug), filtered by destination through links (?destination=, not
// indexed).
@Component({
  selector: 'app-tours-list-page',
  imports: [
    CdkMenu,
    CdkMenuItem,
    CdkMenuItemCheckbox,
    CdkMenuItemRadio,
    CdkMenuTrigger,
    FavoriteButton,
    ListSkeleton,
    LoadError,
    NgOptimizedImage,
    PersonAvatar,
    SharedTransition,
    SiteLink,
    StateBlock,
    TourAlertButton,
    TranslocoPipe,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="ui-container pt-5 pb-14">
      <nav class="ui-crumbs" [attr.aria-label]="'breadcrumbs.label' | transloco">
        <a appSiteLink="/">{{ 'breadcrumbs.home' | transloco }}</a>
        <span class="icon-[heroicons--chevron-right] size-3" aria-hidden="true"></span>
        <span class="text-text" aria-current="page">{{ 'tours.heading' | transloco }}</span>
      </nav>
      <div class="mt-2 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 class="text-[1.875rem] leading-tight font-extrabold tracking-tight text-text">{{ 'tours.heading' | transloco }}</h1>
          <p class="mt-1 text-text-muted">{{ 'tours.metaDescription' | transloco }}</p>
        </div>
        @if (auth.isPartner()) {
          <a appSiteLink="/tours/new" class="ui-btn-soft">
            <span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>{{ 'tours.addButton' | transloco }}
          </a>
        }
      </div>

      <nav class="mt-4 flex flex-wrap gap-2" [attr.aria-label]="'listing.form.destinationLabel' | transloco">
        <a class="ui-pill" [appSiteLink]="filterLink(null)" [attr.aria-current]="destination() ? null : 'page'">{{ 'listing.filters.allDestinations' | transloco }}</a>
        @for (option of shownDestinations(); track option.value) {
          <a class="ui-pill" [appSiteLink]="filterLink(option.value)" [attr.aria-current]="destination() === option.value ? 'page' : null">
            @if (destination() === option.value) {
              <span class="icon-[heroicons--check] size-4" aria-hidden="true"></span>
            }
            {{ option.labelKey | transloco }}
          </a>
        }
        @if (hiddenCount() > 0) {
          <button type="button" class="ui-pill" [attr.aria-expanded]="moreOpen()" (click)="moreOpen.set(true)">
            {{ 'listing.filters.more' | transloco: { count: hiddenCount() } }}
            <span class="icon-[heroicons--chevron-down] size-4" aria-hidden="true"></span>
          </button>
        }
      </nav>

      <div class="mt-4 flex flex-wrap items-center justify-between gap-3 border-t border-border py-2.5">
        <p class="text-[0.84rem] text-text-muted" aria-live="polite">
          @if (tours.hasValue()) {
            {{ 'listing.foundLabel' | transloco }}
            <span class="inline-block overflow-hidden align-bottom font-bold text-text tabular-nums">
              @for (count of [list().length]; track count) {
                <span class="motion-roll-in">{{ count }}</span>
              }
            </span>
          }
        </p>
        <div class="flex flex-wrap items-center gap-2">
          <button type="button" class="ui-btn-soft" aria-controls="tour-dates" [attr.aria-expanded]="datesOpen()" (click)="datesOpen.set(!datesOpen())">
            <span class="icon-[heroicons--calendar-days] size-4" aria-hidden="true"></span>
            @if (from() || to()) {
              {{ datesLabel() }}
            } @else {
              {{ 'tours.dates.button' | transloco }}
            }
          </button>
          <button type="button" class="ui-btn-soft" [cdkMenuTriggerFor]="filtersMenu" [attr.aria-label]="'listing.filters.title' | transloco">
            <span class="icon-[heroicons--adjustments-horizontal] size-4" aria-hidden="true"></span>{{ 'listing.filters.title' | transloco }}
            @if (activeFilters() > 0) {
              <span class="ui-count">{{ activeFilters() }}</span>
            }
          </button>
          <button type="button" class="ui-btn-soft" [cdkMenuTriggerFor]="sortMenu">
            {{ 'listing.sort.' + sortKey() | transloco }}
            <span class="icon-[heroicons--chevron-down] size-4" aria-hidden="true"></span>
          </button>
        </div>
      </div>

      @if (datesOpen()) {
        <form id="tour-dates" class="motion-roll-in mb-2 flex flex-wrap items-end gap-3 rounded-xl border border-border bg-surface p-3" (submit)="applyDates($event, fromInput.value, toInput.value)">
          <label class="block">
            <span class="ui-label">{{ 'tours.dates.from' | transloco }}</span>
            <input #fromInput class="ui-input" type="date" [value]="from() ?? ''" [min]="today" />
          </label>
          <label class="block">
            <span class="ui-label">{{ 'tours.dates.to' | transloco }}</span>
            <input #toInput class="ui-input" type="date" [value]="to() ?? ''" [min]="from() ?? today" />
          </label>
          <div class="flex gap-2">
            <button type="submit" class="ui-btn-primary">{{ 'tours.dates.apply' | transloco }}</button>
            @if (from() || to()) {
              <button type="button" class="ui-btn-ghost" (click)="clearDates()">{{ 'listing.filters.reset' | transloco }}</button>
            }
          </div>
        </form>
      }

      @if (tours.error()) {
        <app-load-error (retry)="tours.reload()" />
      } @else if (tours.hasValue()) {
        @if (list().length > 0) {
          <ul class="motion-reveal mt-2 flex flex-col gap-3.5">
            @for (tour of shown(); track tour.id; let first = $first) {
              <li>
                <a [appSiteLink]="'/tours/' + tour.slug" class="ui-card group flex flex-col gap-4 p-3.5 motion-lift sm:flex-row sm:gap-[1.125rem]">
                  <div class="relative aspect-video w-full shrink-0 overflow-hidden rounded-[0.625rem] bg-slate-200 sm:w-[15.5rem] dark:bg-slate-800" [appSharedTransition]="coverTransition(tour)">
                    @if (tour.coverImageUrl) {
                      <img [ngSrc]="media(tour.coverImageUrl)" fill [priority]="first" sizes="(min-width: 640px) 25vw, 100vw" alt="" class="motion-zoom object-cover" />
                    } @else {
                      <span class="icon-[heroicons--map] absolute inset-0 m-auto size-10 text-text-muted" aria-hidden="true"></span>
                    }
                    <span class="absolute top-2 right-2"><app-favorite-button type="tour" [targetId]="tour.id" /></span>
                  </div>
                  <div class="flex min-w-0 flex-1 flex-col gap-1.5">
                    <div class="flex flex-wrap gap-1.5">
                      @if (tour.destination || tour.location) {
                        <span class="ui-badge ui-badge-dest">
                          <span class="icon-[heroicons--map-pin]" aria-hidden="true"></span>
                          @if (tour.destination) { {{ 'destinations.' + tour.destination | transloco }} } @else { {{ tour.location }} }
                        </span>
                      }
                      @if (tour.cancellationPolicy === 'free') {
                        <span class="ui-badge ui-badge-success"><span class="icon-[heroicons--check]" aria-hidden="true"></span>{{ 'listing.cancellation.free' | transloco }}</span>
                      }
                      @if (tour.durationDays; as days) {
                        <span class="ui-badge ui-badge-soft">{{ 'tours.durationValue' | transloco: { days: days } }}</span>
                      }
                      <span class="ui-badge ui-badge-soft"><span class="icon-[heroicons--calendar-days]" aria-hidden="true"></span>{{ dates(tour) }}</span>
                      @if (left(tour); as count) {
                        @if (count <= fewPlaces) {
                          <span class="ui-badge ui-badge-warning"><span class="icon-[heroicons--fire]" aria-hidden="true"></span>{{ 'tours.places.few' | transloco: { count: count } }}</span>
                        }
                      } @else if (left(tour) === 0) {
                        <span class="ui-badge ui-badge-danger">{{ 'tours.places.none' | transloco }}</span>
                      }
                    </div>
                    <h2 class="text-[1.0625rem] leading-snug font-semibold text-text">{{ tour.title }}</h2>
                    @if (tour.excerpt) {
                      <p class="line-clamp-2 text-[0.8125rem] text-text-muted">{{ tour.excerpt }}</p>
                    }
                    @if (tour.organizer || tour.reviewCount > 0) {
                      <p class="mt-auto flex items-center gap-1.5 pt-1 text-[0.8rem]">
                        @if (tour.organizer; as organizer) {
                          <app-person-avatar [person]="organizer" [size]="20" />
                          <span class="truncate text-text-muted">{{ organizerName(organizer) }}</span>
                        }
                        @if (tour.reviewCount > 0) {
                          @if (tour.organizer) {
                            <span class="text-text-muted" aria-hidden="true">·</span>
                          }
                          <span class="icon-[heroicons--star-solid] size-4 shrink-0 text-secondary-400" aria-hidden="true"></span>
                          <b class="text-text">{{ rating(tour) }}</b>
                          <span class="text-text-muted">({{ tour.reviewCount }})</span>
                        }
                      </p>
                    }
                  </div>
                  @if (tour.price) {
                    <div class="flex shrink-0 flex-col justify-end sm:items-end sm:text-right">
                      <p class="text-xl font-bold text-brand-700 dark:text-brand-300">{{ 'tours.priceValue' | transloco: { price: amount(tour.price) } }}</p>
                      <p class="text-xs text-text-muted">{{ 'tours.perPerson' | transloco }}</p>
                    </div>
                  }
                </a>
              </li>
            }
          </ul>
          @if (list().length > shown().length) {
            <div class="mt-5 flex justify-center">
              <button type="button" class="ui-btn-soft group" (click)="showMore()">
                <span class="icon-[heroicons--arrow-down] icon-anim-bounce size-4" aria-hidden="true"></span>
                {{ 'listing.loadMore' | transloco }}
              </button>
            </div>
          }
        } @else if (all().length > 0 || from() || to()) {
          <app-state-block class="mt-2" icon="icon-[heroicons--magnifying-glass]" title="tours.emptyFiltered" text="tours.emptyFilteredHint">
            <button type="button" class="ui-btn-primary" (click)="resetFilters()">
              <span class="icon-[heroicons--x-mark] size-4" aria-hidden="true"></span>{{ 'listing.filters.reset' | transloco }}
            </button>
          </app-state-block>
        } @else {
          <app-state-block class="mt-2" icon="icon-[heroicons--map]" title="tours.empty" text="tours.emptyHint">
            <a appSiteLink="/outings" class="ui-btn-primary">{{ 'tours.emptyCta' | transloco }}</a>
            <app-tour-alert-button [destination]="destination() ?? ''" />
            @if (destination()) {
              <a appSiteLink="/tours" class="ui-btn-ghost">{{ 'listing.filters.allDestinations' | transloco }}</a>
            }
          </app-state-block>
        }
      } @else {
        <app-list-skeleton class="motion-delayed mt-2" variant="rows" />
      }
    </section>

    <ng-template #sortMenu>
      <div cdkMenu class="motion-drop-in min-w-48 rounded-xl border border-border bg-surface p-1 text-sm text-text shadow-lg">
        @for (key of sortKeys; track key) {
          <button
            cdkMenuItemRadio
            type="button"
            class="flex w-full items-center justify-between gap-3 rounded-lg px-3 py-2 text-left hover:bg-text/5 focus:bg-text/5 focus:outline-none"
            [cdkMenuItemChecked]="sortKey() === key"
            (cdkMenuItemTriggered)="setQuery({ sort: key === 'soonest' ? null : key })"
          >
            {{ 'listing.sort.' + key | transloco }}
            @if (sortKey() === key) {
              <span class="icon-[heroicons--check] size-4 text-brand-600 dark:text-brand-300" aria-hidden="true"></span>
            }
          </button>
        }
      </div>
    </ng-template>

    <ng-template #filtersMenu>
      <div cdkMenu class="motion-drop-in w-72 rounded-xl border border-border bg-surface p-3 text-sm text-text shadow-lg">
        <p class="ui-label">{{ 'tours.filters.duration' | transloco }}</p>
        <div class="flex flex-wrap gap-1.5">
          @for (option of durations; track option) {
            <button cdkMenuItemCheckbox type="button" class="ui-pill px-3 py-1 text-xs" [cdkMenuItemChecked]="duration() === option" [attr.aria-pressed]="duration() === option" (cdkMenuItemTriggered)="setQuery({ duration: duration() === option ? null : option })">
              {{ 'tours.filters.durations.' + option | transloco }}
            </button>
          }
        </div>
        <button
          cdkMenuItemCheckbox
          type="button"
          class="mt-3 flex w-full items-center gap-2 rounded-lg px-1 py-1.5 text-left"
          [cdkMenuItemChecked]="freeCancellation()"
          (cdkMenuItemTriggered)="setQuery({ free: freeCancellation() ? null : '1' })"
        >
          <span class="flex size-[1.125rem] items-center justify-center rounded border" [class]="freeCancellation() ? 'border-brand-600 bg-brand-600 text-white' : 'border-border'">
            @if (freeCancellation()) {
              <span class="icon-[heroicons--check-16-solid] size-3.5" aria-hidden="true"></span>
            }
          </span>
          {{ 'listing.filters.freeCancellation' | transloco }}
        </button>
        @if (activeFilters() > 0) {
          <button cdkMenuItem type="button" class="ui-btn-ghost mt-2 w-full" (cdkMenuItemTriggered)="setQuery({ duration: null, free: null })">{{ 'listing.filters.reset' | transloco }}</button>
        }
      </div>
    </ng-template>
  `,
})
export class ToursListPage {
  readonly destination = input<string | undefined>(undefined);
  readonly sort = input<string | undefined>(undefined);
  readonly duration = input<string | undefined>(undefined);
  readonly free = input<string | undefined>(undefined);
  // Start dates, YYYY-MM-DD (?from=&to=).
  readonly from = input<string | undefined>(undefined);
  readonly to = input<string | undefined>(undefined);
  protected readonly auth = inject(AuthStore);
  private readonly apiBaseUrl = inject(APP_CONFIG).apiBaseUrl;
  private readonly transloco = inject(TranslocoService);
  private readonly lang = injectLang();
  protected readonly setQuery = injectQueryFilters();
  protected readonly sortKeys = SORT_KEYS;
  protected readonly durations = DURATIONS;
  protected readonly moreOpen = signal(false);
  protected readonly datesOpen = signal(false);
  protected readonly fewPlaces = FEW_PLACES;
  protected readonly today = todayInBishkek();

  protected readonly tours = httpResource<Tour[]>(() => ({
    url: '/tours',
    params: {
      locale: this.lang(),
      ...(this.destination() ? { destination: this.destination() as string } : {}),
      ...(isDate(this.from()) ? { from: this.from() as string } : {}),
      ...(isDate(this.to()) ? { to: this.to() as string } : {}),
    },
  }));

  // The first destinations as pills, the rest behind "More" (unless one of
  // them is the chosen one).
  protected readonly shownDestinations = computed(() => {
    const chosen = this.destination();
    const hidden = DESTINATION_OPTIONS.slice(VISIBLE_DESTINATIONS);
    return this.moreOpen() || hidden.some((option) => option.value === chosen)
      ? DESTINATION_OPTIONS
      : DESTINATION_OPTIONS.slice(0, VISIBLE_DESTINATIONS);
  });
  protected readonly hiddenCount = computed(
    () => DESTINATION_OPTIONS.length - this.shownDestinations().length,
  );

  protected readonly sortKey = computed<SortKey>(() => {
    const sort = this.sort();
    return (SORT_KEYS as readonly string[]).includes(sort ?? '') ? (sort as SortKey) : 'soonest';
  });
  protected readonly freeCancellation = computed(() => this.free() === '1');
  protected readonly activeFilters = computed(
    () => (this.duration() ? 1 : 0) + (this.freeCancellation() ? 1 : 0),
  );

  protected readonly all = computed(() => loaded(this.tours) ?? []);
  // Filtered and sorted in the browser: the API returns the whole list.
  protected readonly list = computed(() => {
    const duration = this.duration();
    const free = this.freeCancellation();
    const tours = this.all().filter(
      (tour) =>
        (!free || tour.cancellationPolicy === 'free') &&
        (!duration || matchesDuration(tour.durationDays, duration)),
    );
    const by: Record<SortKey, (a: Tour, b: Tour) => number> = {
      soonest: (a, b) => a.startDate.localeCompare(b.startDate),
      price_asc: (a, b) =>
        (a.price ?? Number.POSITIVE_INFINITY) - (b.price ?? Number.POSITIVE_INFINITY),
      price_desc: (a, b) => (b.price ?? 0) - (a.price ?? 0),
      rating: (a, b) => b.ratingAverage - a.ratingAverage,
    };
    return [...tours].sort(by[this.sortKey()]);
  });
  // How many of the list are shown; back to one page when it changes.
  private readonly limit = linkedSignal({ source: this.list, computation: () => PAGE_SIZE });
  protected readonly shown = computed(() => this.list().slice(0, this.limit()));
  // Free places of the shown tours that have a limit.
  private readonly places = placesResource(() =>
    this.shown()
      .filter((tour) => tour.capacity != null)
      .map((tour) => tour.id),
  );
  protected readonly datesLabel = computed(() => {
    const from = this.from();
    const to = this.to();
    if (from && to) return formatDateRange(this.lang(), from, to);
    return this.transloco.translate(from ? 'tours.dates.fromValue' : 'tours.dates.toValue', {
      date: formatDate(this.lang(), (from ?? to) as string, { day: 'numeric', month: 'short' }),
    });
  });

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

  // Destination links keep the sort and the other filters.
  protected filterLink(destination: string | null): string {
    return withQuery('/tours', {
      destination,
      sort: this.sort(),
      duration: this.duration(),
      free: this.free(),
      from: this.from(),
      to: this.to(),
    });
  }

  protected resetFilters(): void {
    this.setQuery({ duration: null, free: null, from: null, to: null });
  }

  protected showMore(): void {
    this.limit.update((limit) => limit + PAGE_SIZE);
  }

  protected applyDates(event: Event, from: string, to: string): void {
    event.preventDefault();
    this.setQuery({ from: from || null, to: to || null });
    this.datesOpen.set(false);
  }

  protected clearDates(): void {
    this.setQuery({ from: null, to: null });
    this.datesOpen.set(false);
  }

  protected left(tour: Tour): number | null {
    return placesLeft(tour, loaded(this.places));
  }

  protected organizerName(organizer: PublicAuthor): string {
    return personName(organizer);
  }

  protected rating(tour: Tour): string {
    return tour.ratingAverage.toFixed(1);
  }

  protected media(url: string): string {
    return resolveMediaUrl(this.apiBaseUrl, url);
  }

  // A clicked tour's cover flies to the tour page's (motion.css).
  protected coverTransition(tour: Tour): string {
    return tourCoverTransition(tour.id);
  }

  protected dates(tour: Tour): string {
    return formatDateRange(this.lang(), tour.startDate, tour.endDate);
  }

  protected amount(price: number): string {
    return formatAmount(this.lang(), price);
  }
}
