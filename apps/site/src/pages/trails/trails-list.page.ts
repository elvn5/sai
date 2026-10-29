import type { Trail } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { injectQueryFilters } from '@/core/lib/query-filters';
import { loaded } from '@/core/lib/resource';
import { AREAS, GRADES, KINDS, SEASONS, formatGrade } from '@/core/lib/trail-options';
import { itemList } from '@/core/seo/json-ld';
import { SeoService, bindPageMeta } from '@/core/seo/seo.service';
import { TrailCard } from '@/pages/trails/trail-card';
import { TrailsMap } from '@/pages/trails/trails-map';
import { BottomSheet } from '@/ui/bottom-sheet';
import { ListSkeleton } from '@/ui/list-skeleton';
import { LoadError } from '@/ui/load-error';
import { SelectField, injectOptions } from '@/ui/select-field';
import { SiteLink } from '@/ui/site-link';
import { StateBlock } from '@/ui/state-block';
import { CdkMenu, CdkMenuItemRadio, CdkMenuTrigger } from '@angular/cdk/menu';
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

const SEARCH_DEBOUNCE_MS = 300;

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
  imports: [
    BottomSheet,
    CdkMenu,
    CdkMenuItemRadio,
    CdkMenuTrigger,
    ListSkeleton,
    LoadError,
    SelectField,
    SiteLink,
    StateBlock,
    TrailCard,
    TrailsMap,
    TranslocoPipe,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  styles: `
    .menu-item {
      display: flex;
      width: 100%;
      align-items: center;
      justify-content: space-between;
      gap: 0.75rem;
      border-radius: 0.5rem;
      padding: 0.5rem 0.75rem;
      text-align: left;
    }
    .menu-item:hover,
    .menu-item:focus {
      background: color-mix(in oklab, var(--app-text) 5%, transparent);
      outline: none;
    }
  `,
  template: `
    <section class="ui-container pt-6 pb-14">
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 class="text-2xl leading-tight font-extrabold tracking-tight text-text sm:text-[1.875rem]">{{ heading() }}</h1>
          <p class="mt-1 max-w-[38.75rem] text-text-muted max-sm:hidden">{{ 'trails.subtitle' | transloco }}</p>
        </div>
        @if (auth.isPartner()) {
          <a appSiteLink="/trails/new" class="ui-btn-soft">
            <span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>{{ 'trails.add' | transloco }}
          </a>
        }
      </div>

      <div class="mt-4 flex flex-wrap items-center gap-2">
        <label class="ui-input-icon min-w-0 flex-1 sm:w-72 sm:flex-none">
          <span class="icon-[heroicons--magnifying-glass]" aria-hidden="true"></span>
          <input
            type="search"
            class="ui-input"
            [value]="search() ?? ''"
            [placeholder]="'trails.filters.search' | transloco"
            [attr.aria-label]="'trails.filters.search' | transloco"
            (input)="onSearch($any($event.target).value)"
          />
        </label>
        <!-- Phones: one button, the filters in a bottom sheet. -->
        <button type="button" class="ui-btn-soft h-[2.4rem] sm:hidden" [attr.aria-label]="'listing.filters.title' | transloco" (click)="openSheet()">
          <span class="icon-[heroicons--adjustments-horizontal] size-[1.1rem]" aria-hidden="true"></span>
          @if (activeFilters() > 0) {
            <span class="ui-count">{{ activeFilters() }}</span>
          }
        </button>
        <div class="hidden flex-wrap items-center gap-2 sm:flex">
          <button type="button" class="ui-btn-soft" [cdkMenuTriggerFor]="areaMenu">
            {{ area() ? ('trails.area.' + area() | transloco) : ('trails.filters.allAreas' | transloco) }}
            <span class="icon-[heroicons--chevron-down] size-4" aria-hidden="true"></span>
          </button>
          <button type="button" class="ui-btn-soft" [cdkMenuTriggerFor]="gradeMenu">
            {{ gradeLabel(grade()) }}
            <span class="icon-[heroicons--chevron-down] size-4" aria-hidden="true"></span>
          </button>
          <button type="button" class="ui-btn-soft" [cdkMenuTriggerFor]="seasonMenu">
            {{ season() ? ('trails.season.' + season() | transloco) : ('trails.filters.anySeason' | transloco) }}
            <span class="icon-[heroicons--chevron-down] size-4" aria-hidden="true"></span>
          </button>
        </div>
        <button type="button" class="ui-pill hidden sm:ml-auto sm:inline-flex" [attr.aria-pressed]="grade() === hardest" (click)="set({ grade: grade() === hardest ? null : hardest })">
          @if (grade() === hardest) {
            <span class="icon-[heroicons--check] size-4" aria-hidden="true"></span>
          }
          {{ 'trails.filters.harder' | transloco }}
        </button>
      </div>

      @if (activeFilters() > 0) {
        <div class="mt-2.5 flex gap-1.5 overflow-x-auto sm:hidden">
          @if (area(); as value) {
            <button type="button" class="ui-pill shrink-0 px-2.5 py-1 text-xs" aria-pressed="true" (click)="set({ area: null })">{{ 'trails.area.' + value | transloco }}<span class="icon-[heroicons--x-mark] size-3.5" aria-hidden="true"></span></button>
          }
          @if (grade(); as value) {
            <button type="button" class="ui-pill shrink-0 px-2.5 py-1 text-xs" aria-pressed="true" (click)="set({ grade: null })">{{ gradeLabel(value) }}<span class="icon-[heroicons--x-mark] size-3.5" aria-hidden="true"></span></button>
          }
          @if (season(); as value) {
            <button type="button" class="ui-pill shrink-0 px-2.5 py-1 text-xs" aria-pressed="true" (click)="set({ season: null })">{{ 'trails.season.' + value | transloco }}<span class="icon-[heroicons--x-mark] size-3.5" aria-hidden="true"></span></button>
          }
        </div>
      }

      <div class="mt-4 flex flex-wrap items-end gap-3">
        <div class="flex min-w-40 flex-col gap-1 text-sm text-text">
          <span id="trails-kind">{{ 'trails.form.kind' | transloco }}</span>
          <app-select-field [options]="kindOptions()" labelledBy="trails-kind" [value]="kind() ?? ''" (valueChange)="set({ kind: $event || null })" />
        </div>
        <div class="flex min-w-40 flex-col gap-1 text-sm text-text">
          <span id="trails-sort">{{ 'trails.catalog.sort' | transloco }}</span>
          <app-select-field [options]="sortOptions()" labelledBy="trails-sort" [value]="sort() ?? 'popular'" (valueChange)="set({ sort: $event })" />
        </div>
        <div class="ml-auto flex gap-1 rounded-xl border border-border p-1" role="group" [attr.aria-label]="'trails.catalog.view' | transloco">
          <button type="button" class="ui-btn-ghost" [attr.aria-pressed]="view() !== 'map'" (click)="set({ view: null })"><span class="icon-[heroicons--squares-2x2] size-4" aria-hidden="true"></span>{{ 'trails.catalog.list' | transloco }}</button>
          <button type="button" class="ui-btn-soft" [attr.aria-pressed]="view() === 'map'" (click)="set({ view: 'map' })"><span class="icon-[heroicons--map] size-4" aria-hidden="true"></span>{{ 'trails.map.title' | transloco }}</button>
        </div>
      </div>
      @if (view() === 'map' && !trails.error()) {
        @defer (when view() === 'map') {
          <app-trails-map class="mt-5" [trails]="list()" />
        } @placeholder {
          <div class="mt-5 h-80 rounded-card border border-border bg-text/5"></div>
        }
      }
      <div class="mt-5" [attr.aria-busy]="trails.isLoading()">
        @if (trails.error()) {
          <app-load-error (retry)="trails.reload()" />
        } @else if (list().length > 0) {
          <ul class="motion-reveal grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            @for (trail of list(); track trail.id; let index = $index) {
              <li><app-trail-card [trail]="trail" [priority]="index === 0" /></li>
            }
          </ul>
        } @else if (trails.hasValue()) {
          @if (activeFilters() > 0 || search()) {
            <app-state-block icon="icon-[heroicons--magnifying-glass]" title="trails.empty" text="trails.emptyHint">
              <button type="button" class="ui-btn-primary" (click)="resetAll()">
                <span class="icon-[heroicons--x-mark] size-4" aria-hidden="true"></span>{{ 'listing.filters.reset' | transloco }}
              </button>
            </app-state-block>
          } @else {
            <app-state-block icon="icon-[heroicons--flag]" title="trails.emptyAll" />
          }
        } @else {
          <app-list-skeleton class="motion-delayed" />
        }
      </div>
    </section>

    <app-bottom-sheet [(open)]="sheetOpen" title="listing.filters.title">
      <button action type="button" class="ui-btn-ghost ml-auto px-2 text-[0.8rem]" (click)="draft.set({})">{{ 'listing.filters.resetShort' | transloco }}</button>
      <p class="ui-label">{{ 'trails.form.area' | transloco }}</p>
      <div class="flex flex-wrap gap-1.5">
        @for (chip of areaChips(); track chip.value) {
          <button type="button" class="ui-pill px-2.5 py-1 text-xs" [attr.aria-pressed]="draft().area === chip.value" (click)="toggleDraft('area', chip.value)">{{ 'trails.area.' + chip.value | transloco }}</button>
        }
      </div>
      <p class="ui-label mt-4">{{ 'trails.form.grade' | transloco }}</p>
      <div class="ui-segment text-xs" role="radiogroup" [attr.aria-label]="'trails.form.grade' | transloco">
        @for (group of gradeGroups; track group.value) {
          <button type="button" role="radio" [attr.aria-checked]="draft().grade === group.value" (click)="toggleDraft('grade', group.value)">{{ gradeLabel(group.value) }}</button>
        }
      </div>
      <p class="ui-label mt-4">{{ 'trails.form.seasons' | transloco }}</p>
      <div class="flex flex-wrap gap-1.5">
        @for (option of seasons; track option.value) {
          <button type="button" class="ui-pill px-2.5 py-1 text-xs" [attr.aria-pressed]="draft().season === option.value" (click)="toggleDraft('season', option.value)">{{ 'trails.season.' + option.value | transloco }}</button>
        }
      </div>
      <button footer type="button" class="ui-btn-primary ui-btn-lg w-full" (click)="applyDraft()">
        {{ 'trails.filters.show' | transloco }}
        <span class="inline-block overflow-hidden tabular-nums opacity-85">
          @for (count of [draftCount()]; track count) {
            <span class="motion-roll-in">({{ count ?? '…' }})</span>
          }
        </span>
      </button>
    </app-bottom-sheet>

    <ng-template #areaMenu>
      <div cdkMenu class="motion-drop-in max-h-80 min-w-52 overflow-y-auto rounded-xl border border-border bg-surface p-1 text-sm text-text shadow-lg">
        <button cdkMenuItemRadio type="button" class="menu-item" [cdkMenuItemChecked]="!area()" (cdkMenuItemTriggered)="set({ area: null })">
          {{ 'trails.filters.allAreas' | transloco }}
          @if (!area()) { <span class="icon-[heroicons--check] size-4 text-brand-600 dark:text-brand-300" aria-hidden="true"></span> }
        </button>
        @for (chip of areaChips(); track chip.value) {
          <button cdkMenuItemRadio type="button" class="menu-item" [cdkMenuItemChecked]="area() === chip.value" (cdkMenuItemTriggered)="set({ area: chip.value })">
            <span>{{ 'trails.area.' + chip.value | transloco }} <span class="text-text-muted">· {{ chip.count }}</span></span>
            @if (area() === chip.value) { <span class="icon-[heroicons--check] size-4 text-brand-600 dark:text-brand-300" aria-hidden="true"></span> }
          </button>
        }
      </div>
    </ng-template>
    <ng-template #gradeMenu>
      <div cdkMenu class="motion-drop-in min-w-52 rounded-xl border border-border bg-surface p-1 text-sm text-text shadow-lg">
        <button cdkMenuItemRadio type="button" class="menu-item" [cdkMenuItemChecked]="!grade()" (cdkMenuItemTriggered)="set({ grade: null })">
          {{ 'trails.filters.anyGrade' | transloco }}
          @if (!grade()) { <span class="icon-[heroicons--check] size-4 text-brand-600 dark:text-brand-300" aria-hidden="true"></span> }
        </button>
        @for (group of gradeGroups; track group.value) {
          <button cdkMenuItemRadio type="button" class="menu-item" [cdkMenuItemChecked]="grade() === group.value" (cdkMenuItemTriggered)="set({ grade: group.value })">
            {{ gradeLabel(group.value) }}
            @if (grade() === group.value) { <span class="icon-[heroicons--check] size-4 text-brand-600 dark:text-brand-300" aria-hidden="true"></span> }
          </button>
        }
      </div>
    </ng-template>
    <ng-template #seasonMenu>
      <div cdkMenu class="motion-drop-in min-w-44 rounded-xl border border-border bg-surface p-1 text-sm text-text shadow-lg">
        <button cdkMenuItemRadio type="button" class="menu-item" [cdkMenuItemChecked]="!season()" (cdkMenuItemTriggered)="set({ season: null })">
          {{ 'trails.filters.anySeason' | transloco }}
          @if (!season()) { <span class="icon-[heroicons--check] size-4 text-brand-600 dark:text-brand-300" aria-hidden="true"></span> }
        </button>
        @for (option of seasons; track option.value) {
          <button cdkMenuItemRadio type="button" class="menu-item" [cdkMenuItemChecked]="season() === option.value" (cdkMenuItemTriggered)="set({ season: option.value })">
            {{ 'trails.season.' + option.value | transloco }}
            @if (season() === option.value) { <span class="icon-[heroicons--check] size-4 text-brand-600 dark:text-brand-300" aria-hidden="true"></span> }
          </button>
        }
      </div>
    </ng-template>
  `,
})
export class TrailsListPage {
  readonly area = input<string | undefined>(undefined);
  readonly kind = input<string | undefined>(undefined);
  readonly season = input<string | undefined>(undefined);
  readonly grade = input<string | undefined>(undefined);
  readonly search = input<string | undefined>(undefined);
  readonly view = input<string | undefined>(undefined);
  readonly sort = input<string | undefined>(undefined);

  protected readonly auth = inject(AuthStore);
  private readonly transloco = inject(TranslocoService);
  protected readonly set = injectQueryFilters();
  protected readonly seasons = SEASONS;
  protected readonly sorts = ['popular', 'name', 'elevation', 'newest'];
  protected readonly kindOptions = injectOptions(() => [
    { value: '', key: 'trails.catalog.allKinds' },
    ...KINDS.map((option) => ({ value: option.value, key: `trails.kind.${option.value}` })),
  ]);
  protected readonly sortOptions = injectOptions(() =>
    this.sorts.map((value) => ({ value, key: `trails.catalog.sorts.${value}` })),
  );
  protected readonly gradeGroups = GRADE_GROUPS;
  // "3A and harder": the last grade group, also a pill of its own.
  protected readonly hardest = GRADE_GROUPS[3].value;
  protected readonly sheetOpen = signal(false);
  // The bottom sheet's choices, applied with its button.
  protected readonly draft = signal<Partial<Record<'area' | 'grade' | 'season', string>>>({});

  // The URL's filters, as the API takes them.
  private readonly filters = computed(() => {
    const params: Record<string, string> = {};
    const values = {
      area: this.area(),
      kind: this.kind(),
      season: this.season(),
      grade: this.grade(),
      search: this.search()?.trim(),
      sort: this.sort(),
    };
    for (const [key, value] of Object.entries(values)) if (value) params[key] = value;
    return params;
  });
  protected readonly trails = httpResource<Trail[]>(() => ({
    url: '/trails',
    params: this.filters(),
  }));
  private readonly areas = httpResource<Record<string, number>>(() => '/trails/areas');

  protected readonly list = computed(() => loaded(this.trails) ?? []);
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

  protected readonly activeFilters = computed(
    () => [this.area(), this.grade(), this.season(), this.kind()].filter(Boolean).length,
  );
  // How many routes the sheet's choices would show ("Show 8 routes").
  private readonly draftResult = httpResource<Trail[]>(() => {
    if (!this.sheetOpen()) return undefined;
    const params: Record<string, string> = {};
    const search = this.search()?.trim();
    if (search) params['search'] = search;
    const kind = this.kind();
    if (kind) params['kind'] = kind;
    for (const [key, value] of Object.entries(this.draft())) if (value) params[key] = value;
    return { url: '/trails', params };
  });
  protected readonly draftCount = computed(() => loaded(this.draftResult)?.length ?? null);

  private searchTimer: ReturnType<typeof setTimeout> | undefined;

  constructor() {
    const seo = inject(SeoService);
    bindPageMeta(() => {
      if (this.trails.isLoading() || this.areas.isLoading()) return undefined;
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

  protected gradeLabel(value: string | undefined): string {
    const group = GRADE_GROUPS.find((item) => item.value === value);
    if (!group) return this.transloco.translate('trails.filters.anyGrade');
    if (group.label) return group.label;
    return this.transloco.translate(
      group.value === 'trail' ? 'trails.grade.trail' : 'trails.filters.harder',
    );
  }

  protected openSheet(): void {
    this.draft.set({ area: this.area(), grade: this.grade(), season: this.season() });
    this.sheetOpen.set(true);
  }

  protected toggleDraft(key: 'area' | 'grade' | 'season', value: string): void {
    this.draft.update((draft) => ({ ...draft, [key]: draft[key] === value ? undefined : value }));
  }

  protected applyDraft(): void {
    const { area, grade, season } = this.draft();
    this.set({ area: area ?? null, grade: grade ?? null, season: season ?? null });
    this.sheetOpen.set(false);
  }

  protected resetAll(): void {
    this.set({ area: null, grade: null, season: null, kind: null, search: null });
  }

  protected onSearch(value: string): void {
    clearTimeout(this.searchTimer);
    this.searchTimer = setTimeout(() => this.set({ search: value.trim() }), SEARCH_DEBOUNCE_MS);
  }
}
