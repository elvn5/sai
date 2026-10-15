import type { Hotel } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import {
  DESTINATION_OPTIONS,
  HOTEL_AMENITY_OPTIONS,
  HOTEL_TYPE_OPTIONS,
  MIN_RATING_OPTIONS,
} from '@/core/lib/listing-options';
import { pagedList } from '@/core/lib/paged-list';
import { injectQueryFilters } from '@/core/lib/query-filters';
import { pageNumber, queryList, withQuery } from '@/core/lib/query-link';
import { itemList } from '@/core/seo/json-ld';
import { SeoService, bindPageMeta } from '@/core/seo/seo.service';
import { ChipToggleGroup } from '@/ui/chip-toggle-group';
import { HotelCard } from '@/ui/listing/hotel-card';
import { LoadError } from '@/ui/load-error';
import { Pager } from '@/ui/pager';
import { SiteLink } from '@/ui/site-link';
import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  computed,
  inject,
  input,
  signal,
} from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

const PAGE_SIZE = 12;
const SEARCH_DEBOUNCE_MS = 300;
const SORTS = ['rating', 'price_asc', 'price_desc', 'newest'] as const;
const FILTER_AMENITIES = HOTEL_AMENITY_OPTIONS.slice(0, 8);

// Hotels, guesthouses and yurt camps. Filters live in the URL
// (/hotels?destination=karakol), so links from the home page and shared
// views work; filtered views aren't indexed. Page N renders on the server.
@Component({
  selector: 'app-hotels-list-page',
  imports: [ChipToggleGroup, HotelCard, LoadError, Pager, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="mx-auto max-w-6xl px-4 py-12">
      <div class="mb-6 flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 class="text-3xl font-extrabold text-text">{{ heading() }}</h1>
          <p class="mt-1 text-text-muted">{{ 'hotels.subheading' | transloco }}</p>
        </div>
        @if (auth.isPartner()) {
          <a appSiteLink="/hotels/new" class="ui-btn-primary">
            <span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>{{ 'hotels.addButton' | transloco }}
          </a>
        }
      </div>

      <div class="flex flex-col gap-3 sm:flex-row">
        <div class="relative flex-1">
          <span class="icon-[heroicons--magnifying-glass] pointer-events-none absolute top-1/2 left-3 size-5 -translate-y-1/2 text-text-muted" aria-hidden="true"></span>
          <input
            type="search"
            class="ui-input pl-10"
            [value]="search() ?? ''"
            [placeholder]="'hotels.searchPlaceholder' | transloco"
            [attr.aria-label]="'hotels.searchPlaceholder' | transloco"
            (input)="onSearch($any($event.target).value)"
          />
        </div>
        <select class="ui-input sm:w-56" [attr.aria-label]="'listing.form.destinationLabel' | transloco" (change)="set({ destination: $any($event.target).value })">
          <option value="" [selected]="!destination()">{{ 'listing.filters.allDestinations' | transloco }}</option>
          @for (option of destinations; track option.value) {
            <option [value]="option.value" [selected]="destination() === option.value">{{ option.labelKey | transloco }}</option>
          }
        </select>
        <button type="button" class="ui-btn-soft lg:hidden" [attr.aria-expanded]="filtersOpen()" aria-controls="hotel-filters" (click)="filtersOpen.set(!filtersOpen())">
          <span class="icon-[heroicons--adjustments-horizontal] size-4" aria-hidden="true"></span>
          {{ 'listing.filters.title' | transloco }}@if (activeFilters() > 0) { ({{ activeFilters() }}) }
        </button>
      </div>

      <div class="mt-6 flex flex-col gap-8 lg:flex-row">
        <aside id="hotel-filters" class="lg:block lg:w-64 lg:shrink-0" [class.hidden]="!filtersOpen()" [attr.aria-label]="'listing.filters.title' | transloco">
          <div class="ui-card flex flex-col gap-5 p-4 lg:sticky lg:top-20">
            <label class="ui-field">
              {{ 'hotels.typeLabel' | transloco }}
              <select class="ui-input" (change)="set({ type: $any($event.target).value })">
                <option value="" [selected]="!type()">{{ 'listing.filters.allTypes' | transloco }}</option>
                @for (option of types; track option.value) {
                  <option [value]="option.value" [selected]="type() === option.value">{{ option.labelKey | transloco }}</option>
                }
              </select>
            </label>
            <label class="ui-field">
              <span>{{ 'listing.filters.maxPrice' | transloco }} <span class="text-text-muted">({{ 'listing.currency' | transloco }})</span></span>
              <input class="ui-input" type="number" min="0" step="500" inputmode="numeric" [value]="maxPrice() ?? ''" (change)="set({ maxPrice: $any($event.target).value })" />
            </label>
            <label class="ui-field">
              {{ 'listing.filters.minRating' | transloco }}
              <select class="ui-input" (change)="set({ minRating: $any($event.target).value })">
                <option value="" [selected]="!minRating()">{{ 'listing.filters.anyRating' | transloco }}</option>
                @for (value of ratings; track value) {
                  <option [value]="value" [selected]="minRating() === '' + value">{{ value }}+ ★</option>
                }
              </select>
            </label>
            <label class="flex items-center gap-2 text-sm text-text">
              <input type="checkbox" class="size-4 accent-brand-600" [checked]="freeCancellation() === 'true'" (change)="set({ freeCancellation: $any($event.target).checked })" />
              {{ 'listing.filters.freeCancellation' | transloco }}
            </label>
            <div class="ui-field">
              <span>{{ 'hotels.amenitiesLabel' | transloco }}</span>
              <app-chip-toggle-group [options]="amenityOptions" [label]="'hotels.amenitiesLabel' | transloco" [value]="amenityList()" (valueChange)="set({ amenities: $event })" />
            </div>
            @if (activeFilters() > 0) {
              <button type="button" class="ui-btn-ghost" (click)="reset()">{{ 'listing.filters.reset' | transloco }}</button>
            }
          </div>
        </aside>

        <div class="flex-1">
          <div class="mb-4 flex items-center justify-between gap-4">
            <p class="text-sm text-text-muted" aria-live="polite">{{ 'listing.found' | transloco: { count: list.total() } }}</p>
            <select class="ui-input w-48 py-1.5" [attr.aria-label]="'listing.sort.rating' | transloco" (change)="set({ sort: $any($event.target).value === 'rating' ? null : $any($event.target).value })">
              @for (value of sorts; track value) {
                <option [value]="value" [selected]="sortValue() === value">{{ 'listing.sort.' + value | transloco }}</option>
              }
            </select>
          </div>

          @if (list.first.error()) {
            <app-load-error (retry)="list.reload()" />
          } @else if (list.items().length > 0) {
            <ul class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
              @for (hotel of list.items(); track hotel.id; let index = $index) {
                <li><app-hotel-card [hotel]="hotel" [priority]="index === 0" /></li>
              }
            </ul>
          } @else if (list.first.hasValue()) {
            <div class="flex flex-col items-center gap-3 rounded-card border border-dashed border-border px-6 py-14 text-center">
              <span class="icon-[heroicons--building-office-2] size-10 text-text-muted" aria-hidden="true"></span>
              <p class="text-text-muted">{{ 'hotels.empty' | transloco }}</p>
              @if (activeFilters() > 0) {
                <button type="button" class="ui-btn-soft" (click)="reset()">{{ 'listing.filters.reset' | transloco }}</button>
              }
            </div>
          }

          <app-pager [previous]="previousLink()" [next]="nextLink()" [loading]="list.loadingMore()" (loadMore)="list.loadMore()" />
        </div>
      </div>
    </section>
  `,
})
export class HotelsListPage {
  readonly destination = input<string | undefined>(undefined);
  readonly type = input<string | undefined>(undefined);
  readonly maxPrice = input<string | undefined>(undefined);
  readonly minRating = input<string | undefined>(undefined);
  readonly amenities = input<string | undefined>(undefined);
  readonly freeCancellation = input<string | undefined>(undefined);
  readonly sort = input<string | undefined>(undefined);
  readonly search = input<string | undefined>(undefined);
  readonly page = input<string | undefined>(undefined);

  protected readonly auth = inject(AuthStore);
  private readonly transloco = inject(TranslocoService);
  protected readonly set = injectQueryFilters();
  protected readonly destinations = DESTINATION_OPTIONS;
  protected readonly types = HOTEL_TYPE_OPTIONS;
  protected readonly ratings = MIN_RATING_OPTIONS;
  protected readonly sorts = SORTS;
  protected readonly amenityOptions = FILTER_AMENITIES;
  protected readonly filtersOpen = signal(false);

  protected readonly amenityList = computed(() => queryList(this.amenities()));
  protected readonly sortValue = computed(() => this.sort() || 'rating');
  private readonly pageNo = computed(() => pageNumber(this.page()));

  // The URL's filters, as the API and the links take them.
  private readonly filters = computed(() => ({
    destination: this.destination() ?? '',
    type: this.type() ?? '',
    maxPrice: this.maxPrice() ?? '',
    minRating: this.minRating() ?? '',
    amenities: this.amenityList().join(','),
    freeCancellation: this.freeCancellation() === 'true' ? 'true' : '',
    search: this.search() ?? '',
  }));
  private readonly apiParams = computed(() => {
    const params: Record<string, string> = { sort: this.sortValue() };
    for (const [key, value] of Object.entries(this.filters())) if (value) params[key] = value;
    return params;
  });

  protected readonly list = pagedList<Hotel>({
    url: '/hotels',
    pageSize: PAGE_SIZE,
    page: this.pageNo,
    params: this.apiParams,
  });

  protected readonly activeFilters = computed(() => {
    const f = this.filters();
    return (
      [f.destination, f.type, f.maxPrice, f.minRating, f.freeCancellation].filter(Boolean).length +
      this.amenityList().length
    );
  });
  private readonly linkQuery = computed(() => ({ ...this.filters(), sort: this.sort() ?? '' }));
  protected readonly previousLink = computed(() =>
    this.pageNo() > 1
      ? withQuery('/hotels', { ...this.linkQuery(), page: this.pageNo() - 1 })
      : null,
  );
  protected readonly nextLink = computed(() =>
    this.list.hasMore()
      ? withQuery('/hotels', { ...this.linkQuery(), page: this.list.nextPage() })
      : null,
  );
  protected readonly heading = computed(() => {
    const destination = this.destination();
    return destination
      ? this.transloco.translate('hotels.headingIn', {
          place: this.transloco.translate(`destinations.${destination}`),
        })
      : this.transloco.translate('hotels.heading');
  });

  private searchTimer: ReturnType<typeof setTimeout> | undefined;

  constructor() {
    const seo = inject(SeoService);
    bindPageMeta(() => {
      if (!this.list.first.hasValue() && !this.list.first.error()) return undefined;
      const page = this.pageNo();
      const filtered = this.activeFilters() > 0 || Boolean(this.search()) || Boolean(this.sort());
      const heading = this.transloco.translate('hotels.heading');
      return {
        title:
          page > 1
            ? `${this.heading()} — ${this.transloco.translate('pagination.page', { page })}`
            : this.heading(),
        description: this.transloco.translate('hotels.metaDescription'),
        image: this.list.items()[0]?.coverImageUrl ?? this.list.items()[0]?.photos[0],
        query: page > 1 && !filtered ? { page } : undefined,
        robots: filtered ? 'noindex, follow' : undefined,
        jsonLd: [
          itemList(
            this.list
              .items()
              .map((hotel) => ({ url: seo.pageUrl(`/hotels/${hotel.slug}`), name: hotel.name })),
          ),
          seo.breadcrumbs([{ name: heading, path: '/hotels' }]),
        ],
      };
    });
    inject(DestroyRef).onDestroy(() => clearTimeout(this.searchTimer));
  }

  protected onSearch(value: string): void {
    clearTimeout(this.searchTimer);
    this.searchTimer = setTimeout(() => this.set({ search: value.trim() }), SEARCH_DEBOUNCE_MS);
  }

  protected reset(): void {
    this.set({
      destination: null,
      type: null,
      maxPrice: null,
      minRating: null,
      amenities: null,
      freeCancellation: null,
      search: null,
    });
  }
}
