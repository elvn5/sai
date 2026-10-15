import type { ConditionReport, Trail } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { injectLang } from '@/core/i18n/current-lang';
import { formatDate } from '@/core/lib/format';
import { injectQueryFilters } from '@/core/lib/query-filters';
import { withQuery } from '@/core/lib/query-link';
import { loaded } from '@/core/lib/resource';
import {
  AREAS,
  GRADES,
  KINDS,
  SEASONS,
  TRAIL_STATES,
  formatGrade,
  optionIcon,
} from '@/core/lib/trail-options';
import { itemList } from '@/core/seo/json-ld';
import { SeoService, bindPageMeta } from '@/core/seo/seo.service';
import { TrailCard } from '@/pages/trails/trail-card';
import { LoadError } from '@/ui/load-error';
import { SiteLink } from '@/ui/site-link';
import { httpResource } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  computed,
  inject,
  input,
} from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

const SEARCH_DEBOUNCE_MS = 300;
const RECENT_REPORTS = 8;

// Grade filter groups: the API takes a comma-separated list of grades.
const GRADE_GROUPS = [
  { value: 'trail', label: null },
  { value: '1a,1b', label: `${formatGrade('1a')}–${formatGrade('1b')}` },
  { value: '2a,2b', label: `${formatGrade('2a')}–${formatGrade('2b')}` },
  { value: GRADES.slice(5).join(','), label: null },
] as const;

// The route catalog with the latest condition reports on top. Filters live
// in the URL (/trails?area=ala_archa); areas are links, so every area's
// list is crawlable, but filtered views aren't indexed.
@Component({
  selector: 'app-trails-list-page',
  imports: [LoadError, SiteLink, TrailCard, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="mx-auto max-w-6xl px-4 py-12">
      <div class="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 class="text-3xl font-extrabold text-text">{{ heading() }}</h1>
          <p class="mt-1 max-w-2xl text-text-muted">{{ 'trails.subtitle' | transloco }}</p>
        </div>
        <div class="flex gap-2">
          <a appSiteLink="/outings" class="ui-btn-soft">
            <span class="icon-[mdi--account-group] size-4" aria-hidden="true"></span>{{ 'nav.outings' | transloco }}
          </a>
          @if (auth.isPartner()) {
            <a appSiteLink="/trails/new" class="ui-btn-primary">
              <span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>{{ 'trails.add' | transloco }}
            </a>
          }
        </div>
      </div>

      @if (recentReports().length > 0) {
        <section class="mb-8" aria-labelledby="recent-reports">
          <h2 id="recent-reports" class="mb-3 flex items-center gap-2 text-sm font-semibold tracking-wide text-text-muted uppercase">
            <span class="relative flex size-2" aria-hidden="true">
              <span class="absolute inline-flex size-full animate-ping rounded-full bg-emerald-400 opacity-60 motion-reduce:hidden"></span>
              <span class="relative inline-flex size-2 rounded-full bg-emerald-500"></span>
            </span>
            {{ 'trails.conditions.recent' | transloco }}
          </h2>
          <ul class="-mx-4 flex snap-x gap-3 overflow-x-auto px-4 pb-2">
            @for (report of recentReports(); track report.id) {
              <li class="w-64 shrink-0 snap-start">
                <a [appSiteLink]="'/trails/' + report.trail.slug" class="ui-card block h-full p-3 transition hover:border-brand-400">
                  <span class="block truncate text-sm font-semibold text-text">{{ report.trail.name }}</span>
                  <span class="mt-1 flex items-center gap-1 text-xs text-text-muted">
                    <span [class]="stateIcon(report.trailState) + ' size-4'" aria-hidden="true"></span>
                    {{ 'trails.state.' + report.trailState | transloco }} · {{ 'trails.traction.' + report.traction | transloco }}
                  </span>
                  @if (report.text) {
                    <span class="mt-1 line-clamp-2 block text-xs text-text">{{ report.text }}</span>
                  }
                  <span class="mt-1 block text-xs text-text-muted">{{ observed(report.observedOn) }}</span>
                </a>
              </li>
            }
          </ul>
        </section>
      }

      <nav class="mb-4 flex flex-wrap gap-2" [attr.aria-label]="'trails.form.area' | transloco">
        <a class="ui-pill px-3 py-1 text-xs" [appSiteLink]="areaLink(null)" [attr.aria-current]="area() ? null : 'page'">{{ 'trails.filters.allAreas' | transloco }}</a>
        @for (chip of areaChips(); track chip.value) {
          <a class="ui-pill px-3 py-1 text-xs" [appSiteLink]="areaLink(chip.value)" [attr.aria-current]="area() === chip.value ? 'page' : null">
            {{ 'trails.area.' + chip.value | transloco }} · {{ chip.count }}
          </a>
        }
      </nav>

      <div class="mb-8 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <div class="relative">
          <span class="icon-[heroicons--magnifying-glass] pointer-events-none absolute top-1/2 left-3 size-5 -translate-y-1/2 text-text-muted" aria-hidden="true"></span>
          <input
            type="search"
            class="ui-input pl-10"
            [value]="search() ?? ''"
            [placeholder]="'trails.filters.search' | transloco"
            [attr.aria-label]="'trails.filters.search' | transloco"
            (input)="onSearch($any($event.target).value)"
          />
        </div>
        <select class="ui-input" [attr.aria-label]="'trails.form.kind' | transloco" (change)="set({ kind: $any($event.target).value })">
          <option value="" [selected]="!kind()">{{ 'trails.filters.anyKind' | transloco }}</option>
          @for (option of kinds; track option.value) {
            <option [value]="option.value" [selected]="kind() === option.value">{{ 'trails.kind.' + option.value | transloco }}</option>
          }
        </select>
        <select class="ui-input" [attr.aria-label]="'trails.form.grade' | transloco" (change)="set({ grade: $any($event.target).value })">
          <option value="" [selected]="!grade()">{{ 'trails.filters.anyGrade' | transloco }}</option>
          @for (group of gradeGroups; track group.value; let i = $index) {
            <option [value]="group.value" [selected]="grade() === group.value">
              @if (group.label) { {{ group.label }} } @else if (i === 0) { {{ 'trails.grade.trail' | transloco }} } @else { {{ 'trails.filters.harder' | transloco }} }
            </option>
          }
        </select>
        <select class="ui-input" [attr.aria-label]="'trails.form.seasons' | transloco" (change)="set({ season: $any($event.target).value })">
          <option value="" [selected]="!season()">{{ 'trails.filters.anySeason' | transloco }}</option>
          @for (option of seasons; track option.value) {
            <option [value]="option.value" [selected]="season() === option.value">{{ 'trails.season.' + option.value | transloco }}</option>
          }
        </select>
      </div>

      @if (trails.error()) {
        <app-load-error (retry)="trails.reload()" />
      } @else if (list().length > 0) {
        <ul class="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
          @for (trail of list(); track trail.id; let index = $index) {
            <li><app-trail-card [trail]="trail" [priority]="index === 0" /></li>
          }
        </ul>
      } @else if (trails.hasValue()) {
        <div class="flex flex-col items-center gap-3 rounded-card border border-dashed border-border px-6 py-14 text-center">
          <span class="icon-[mdi--map-search-outline] size-10 text-text-muted" aria-hidden="true"></span>
          <p class="text-text-muted">{{ 'trails.empty' | transloco }}</p>
        </div>
      }
    </section>
  `,
})
export class TrailsListPage {
  readonly area = input<string | undefined>(undefined);
  readonly kind = input<string | undefined>(undefined);
  readonly season = input<string | undefined>(undefined);
  readonly grade = input<string | undefined>(undefined);
  readonly search = input<string | undefined>(undefined);

  protected readonly auth = inject(AuthStore);
  private readonly transloco = inject(TranslocoService);
  private readonly lang = injectLang();
  protected readonly set = injectQueryFilters();
  protected readonly kinds = KINDS;
  protected readonly seasons = SEASONS;
  protected readonly gradeGroups = GRADE_GROUPS;

  // The URL's filters, as the API takes them.
  private readonly filters = computed(() => {
    const params: Record<string, string> = {};
    const values = {
      area: this.area(),
      kind: this.kind(),
      season: this.season(),
      grade: this.grade(),
      search: this.search()?.trim(),
    };
    for (const [key, value] of Object.entries(values)) if (value) params[key] = value;
    return params;
  });
  protected readonly trails = httpResource<Trail[]>(() => ({
    url: '/trails',
    params: this.filters(),
  }));
  private readonly areas = httpResource<Record<string, number>>(() => '/trails/areas');
  private readonly recent = httpResource<ConditionReport[]>(() => ({
    url: '/conditions',
    params: { limit: RECENT_REPORTS },
  }));

  protected readonly list = computed(() => loaded(this.trails) ?? []);
  protected readonly recentReports = computed(() => loaded(this.recent) ?? []);
  protected readonly areaChips = computed(() => {
    const counts = loaded(this.areas) ?? {};
    return AREAS.filter((area) => (counts[area] ?? 0) > 0).map((area) => ({
      value: area,
      count: counts[area] ?? 0,
    }));
  });
  protected readonly heading = computed(() => {
    const area = this.area();
    return area
      ? `${this.transloco.translate('trails.title')}: ${this.transloco.translate(`trails.area.${area}`)}`
      : this.transloco.translate('trails.title');
  });

  private searchTimer: ReturnType<typeof setTimeout> | undefined;

  constructor() {
    const seo = inject(SeoService);
    bindPageMeta(() => {
      if (this.trails.isLoading() || this.areas.isLoading() || this.recent.isLoading())
        return undefined;
      const heading = this.transloco.translate('trails.title');
      return {
        title: this.heading(),
        description: this.transloco.translate('trails.subtitle'),
        image: this.list()[0]?.coverImageUrl ?? this.list()[0]?.photos[0],
        robots: Object.keys(this.filters()).length > 0 ? 'noindex, follow' : undefined,
        jsonLd: [
          itemList(
            this.list().map((trail) => ({
              url: seo.pageUrl(`/trails/${trail.slug}`),
              name: trail.name,
            })),
          ),
          seo.breadcrumbs([{ name: heading, path: '/trails' }]),
        ],
      };
    });
    inject(DestroyRef).onDestroy(() => clearTimeout(this.searchTimer));
  }

  // The other filters stay: an area link narrows what's already chosen.
  protected areaLink(area: string | null): string {
    return withQuery('/trails', {
      area,
      kind: this.kind(),
      season: this.season(),
      grade: this.grade(),
      search: this.search(),
    });
  }

  protected stateIcon(state: string): string {
    return optionIcon(TRAIL_STATES, state);
  }

  protected observed(date: string): string {
    return formatDate(this.lang(), date, { day: 'numeric', month: 'long' });
  }

  protected onSearch(value: string): void {
    clearTimeout(this.searchTimer);
    this.searchTimer = setTimeout(() => this.set({ search: value.trim() }), SEARCH_DEBOUNCE_MS);
  }
}
