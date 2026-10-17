import type { LocalGuide } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import {
  DESTINATION_OPTIONS,
  GUIDE_LANGUAGE_OPTIONS,
  GUIDE_SPECIALIZATION_OPTIONS,
  MIN_RATING_OPTIONS,
} from '@/core/lib/listing-options';
import { pagedList } from '@/core/lib/paged-list';
import { injectQueryFilters } from '@/core/lib/query-filters';
import { pageNumber, withQuery } from '@/core/lib/query-link';
import { loaded } from '@/core/lib/resource';
import { itemList } from '@/core/seo/json-ld';
import { SeoService, bindPageMeta } from '@/core/seo/seo.service';
import { LocalGuideCard } from '@/pages/local-guides/local-guide-card';
import { ListSkeleton } from '@/ui/list-skeleton';
import { LoadError } from '@/ui/load-error';
import { Pager } from '@/ui/pager';
import { SiteLink } from '@/ui/site-link';
import { httpResource } from '@angular/common/http';
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
const SORTS = ['rating', 'price_asc', 'price_desc', 'experience', 'newest'] as const;

// Local guides: filters in the URL (not indexed), page N on the server. A
// signed-in visitor with a guide profile gets a link to it; partners without
// one, the button to create it.
@Component({
  selector: 'app-local-guides-list-page',
  imports: [ListSkeleton, LoadError, LocalGuideCard, Pager, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="mx-auto max-w-6xl px-4 py-12">
      <div class="mb-6 flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 class="text-3xl font-extrabold text-text">{{ 'localGuides.heading' | transloco }}</h1>
          <p class="mt-1 text-text-muted">{{ 'localGuides.subheading' | transloco }}</p>
        </div>
        @if (myGuide(); as mine) {
          <a [appSiteLink]="'/local-guides/' + mine.slug" class="ui-btn-soft">
            <span class="icon-[heroicons--user-circle] size-4" aria-hidden="true"></span>{{ 'localGuides.myProfile' | transloco }}
          </a>
        } @else if (auth.isPartner() && mine.hasValue()) {
          <a appSiteLink="/local-guides/new" class="ui-btn-primary">
            <span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>{{ 'localGuides.addButton' | transloco }}
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
            [placeholder]="'localGuides.searchPlaceholder' | transloco"
            [attr.aria-label]="'localGuides.searchPlaceholder' | transloco"
            (input)="onSearch($any($event.target).value)"
          />
        </div>
        <select class="ui-input sm:w-56" [attr.aria-label]="'listing.form.destinationLabel' | transloco" (change)="set({ destination: $any($event.target).value })">
          <option value="" [selected]="!destination()">{{ 'listing.filters.allDestinations' | transloco }}</option>
          @for (option of destinations; track option.value) {
            <option [value]="option.value" [selected]="destination() === option.value">{{ option.labelKey | transloco }}</option>
          }
        </select>
        <select class="ui-input sm:w-44" [attr.aria-label]="'localGuides.languagesLabel' | transloco" (change)="set({ language: $any($event.target).value })">
          <option value="" [selected]="!language()">{{ 'localGuides.anyLanguage' | transloco }}</option>
          @for (option of languages; track option.value) {
            <option [value]="option.value" [selected]="language() === option.value">{{ option.labelKey | transloco }}</option>
          }
        </select>
        <button type="button" class="ui-btn-soft lg:hidden" [attr.aria-expanded]="filtersOpen()" aria-controls="guide-filters" (click)="filtersOpen.set(!filtersOpen())">
          <span class="icon-[heroicons--adjustments-horizontal] size-4" aria-hidden="true"></span>
          {{ 'listing.filters.title' | transloco }}@if (activeFilters() > 0) { ({{ activeFilters() }}) }
        </button>
      </div>

      <div class="mt-6 flex flex-col gap-8 lg:flex-row">
        <aside id="guide-filters" class="lg:block lg:w-64 lg:shrink-0" [class.hidden]="!filtersOpen()" [attr.aria-label]="'listing.filters.title' | transloco">
          <div class="ui-card flex flex-col gap-5 p-4 lg:sticky lg:top-20">
            <label class="ui-field">
              {{ 'localGuides.specializationsLabel' | transloco }}
              <select class="ui-input" (change)="set({ specialization: $any($event.target).value })">
                <option value="" [selected]="!specialization()">{{ 'localGuides.anySpecialization' | transloco }}</option>
                @for (option of specializations; track option.value) {
                  <option [value]="option.value" [selected]="specialization() === option.value">{{ option.labelKey | transloco }}</option>
                }
              </select>
            </label>
            <label class="ui-field">
              <span>{{ 'listing.filters.maxPricePerDay' | transloco }} <span class="text-text-muted">({{ 'listing.currency' | transloco }})</span></span>
              <input class="ui-input" type="number" min="0" step="1000" inputmode="numeric" [value]="maxPrice() ?? ''" (change)="set({ maxPrice: $any($event.target).value })" />
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
              <input type="checkbox" class="size-4 accent-brand-600" [checked]="hasVehicle() === 'true'" (change)="set({ hasVehicle: $any($event.target).checked })" />
              {{ 'localGuides.vehicleFilter' | transloco }}
            </label>
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
            <ul class="motion-reveal grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
              @for (guide of list.items(); track guide.id; let index = $index) {
                <li><app-local-guide-card [guide]="guide" [priority]="index === 0" /></li>
              }
            </ul>
          } @else if (list.first.hasValue()) {
            <div class="flex flex-col items-center gap-3 rounded-card border border-dashed border-border px-6 py-14 text-center">
              <span class="icon-[heroicons--user-group] size-10 text-text-muted" aria-hidden="true"></span>
              <p class="text-text-muted">{{ 'localGuides.empty' | transloco }}</p>
              @if (activeFilters() > 0) {
                <button type="button" class="ui-btn-soft" (click)="reset()">{{ 'listing.filters.reset' | transloco }}</button>
              }
            </div>
          } @else {
            <app-list-skeleton columns="grid-cols-1 sm:grid-cols-2 xl:grid-cols-3" />
          }

          <app-pager [previous]="previousLink()" [next]="nextLink()" [loading]="list.loadingMore()" (loadMore)="list.loadMore()" />
        </div>
      </div>
    </section>
  `,
})
export class LocalGuidesListPage {
  readonly destination = input<string | undefined>(undefined);
  readonly language = input<string | undefined>(undefined);
  readonly specialization = input<string | undefined>(undefined);
  readonly maxPrice = input<string | undefined>(undefined);
  readonly minRating = input<string | undefined>(undefined);
  readonly hasVehicle = input<string | undefined>(undefined);
  readonly sort = input<string | undefined>(undefined);
  readonly search = input<string | undefined>(undefined);
  readonly page = input<string | undefined>(undefined);

  protected readonly auth = inject(AuthStore);
  private readonly transloco = inject(TranslocoService);
  protected readonly set = injectQueryFilters();
  protected readonly destinations = DESTINATION_OPTIONS;
  protected readonly languages = GUIDE_LANGUAGE_OPTIONS;
  protected readonly specializations = GUIDE_SPECIALIZATION_OPTIONS;
  protected readonly ratings = MIN_RATING_OPTIONS;
  protected readonly sorts = SORTS;
  protected readonly filtersOpen = signal(false);

  // The visitor's own guide profile (signed-in visitors only).
  protected readonly mine = httpResource<{ guide: LocalGuide | null }>(() =>
    this.auth.user() ? '/local-guides/mine' : undefined,
  );
  protected readonly myGuide = computed(() => loaded(this.mine)?.guide ?? null);

  protected readonly sortValue = computed(() => this.sort() || 'rating');
  private readonly pageNo = computed(() => pageNumber(this.page()));
  private readonly filters = computed(() => ({
    destination: this.destination() ?? '',
    language: this.language() ?? '',
    specialization: this.specialization() ?? '',
    maxPrice: this.maxPrice() ?? '',
    minRating: this.minRating() ?? '',
    hasVehicle: this.hasVehicle() === 'true' ? 'true' : '',
    search: this.search() ?? '',
  }));
  private readonly apiParams = computed(() => {
    const params: Record<string, string> = { sort: this.sortValue() };
    for (const [key, value] of Object.entries(this.filters())) if (value) params[key] = value;
    return params;
  });

  protected readonly list = pagedList<LocalGuide>({
    url: '/local-guides',
    pageSize: PAGE_SIZE,
    page: this.pageNo,
    params: this.apiParams,
  });

  protected readonly activeFilters = computed(() => {
    const f = this.filters();
    return [
      f.destination,
      f.language,
      f.specialization,
      f.maxPrice,
      f.minRating,
      f.hasVehicle,
    ].filter(Boolean).length;
  });
  private readonly linkQuery = computed(() => ({ ...this.filters(), sort: this.sort() ?? '' }));
  protected readonly previousLink = computed(() =>
    this.pageNo() > 1
      ? withQuery('/local-guides', { ...this.linkQuery(), page: this.pageNo() - 1 })
      : null,
  );
  protected readonly nextLink = computed(() =>
    this.list.hasMore()
      ? withQuery('/local-guides', { ...this.linkQuery(), page: this.list.nextPage() })
      : null,
  );

  private searchTimer: ReturnType<typeof setTimeout> | undefined;

  constructor() {
    const seo = inject(SeoService);
    bindPageMeta(() => {
      if (!this.list.first.hasValue() && !this.list.first.error()) return undefined;
      const page = this.pageNo();
      const filtered = this.activeFilters() > 0 || Boolean(this.search()) || Boolean(this.sort());
      const heading = this.transloco.translate('localGuides.heading');
      return {
        title:
          page > 1
            ? `${heading} — ${this.transloco.translate('pagination.page', { page })}`
            : heading,
        description: this.transloco.translate('localGuides.metaDescription'),
        query: page > 1 && !filtered ? { page } : undefined,
        robots: filtered ? 'noindex, follow' : undefined,
        jsonLd: [
          itemList(
            this.list.items().map((guide) => ({
              url: seo.pageUrl(`/local-guides/${guide.slug}`),
              name: guide.name,
            })),
          ),
          seo.breadcrumbs([{ name: heading, path: '/local-guides' }]),
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
      language: null,
      specialization: null,
      maxPrice: null,
      minRating: null,
      hasVehicle: null,
      search: null,
    });
  }
}
