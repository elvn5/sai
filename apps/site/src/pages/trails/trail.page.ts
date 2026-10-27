import type { ConditionReport, Trail } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { APP_CONFIG } from '@/core/config/app-config';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { loaded } from '@/core/lib/resource';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import {
  GEAR,
  KINDS,
  SEASONS,
  formatGrade,
  gradeClass,
  optionIcon,
} from '@/core/lib/trail-options';
import { trailCoverTransition } from '@/core/lib/view-transitions';
import { plainText, touristAttraction } from '@/core/seo/json-ld';
import { SeoService, bindPageMeta } from '@/core/seo/seo.service';
import { NotFoundPage } from '@/pages/not-found/not-found.page';
import { ConditionBadge } from '@/pages/trails/condition-badge';
import { ConditionCard } from '@/pages/trails/condition-card';
import {
  type ConditionReportData,
  ConditionReportDialog,
} from '@/pages/trails/condition-report-dialog';
import { ElevationProfile } from '@/pages/trails/elevation-profile';
import { Trail3d } from '@/pages/trails/trail-3d';
import { TrailCanvas } from '@/pages/trails/trail-canvas';
import { TrailMap } from '@/pages/trails/trail-map';
import { LoadError } from '@/ui/load-error';
import { SharedTransition } from '@/ui/shared-transition';
import { SiteLink } from '@/ui/site-link';
import { Dialog } from '@angular/cdk/dialog';
import { NgOptimizedImage } from '@angular/common';
import { HttpClient, HttpErrorResponse, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, input, signal } from '@angular/core';
import { Router } from '@angular/router';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

const REPORTS_LIMIT = 20;

interface Stat {
  icon: string;
  value: string;
  label: string;
}

// One route: the numbers, the map with its elevation profile, how to get
// there, condition reports, the weather up there.
@Component({
  selector: 'app-trail-page',
  imports: [
    ConditionBadge,
    ConditionCard,
    ElevationProfile,
    LoadError,
    NgOptimizedImage,
    NotFoundPage,
    SharedTransition,
    SiteLink,
    TrailMap,
    TrailCanvas,
    Trail3d,
    TranslocoPipe,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (notFound()) {
      <app-not-found-page />
    } @else if (trail.error()) {
      <div class="mx-auto max-w-5xl px-4 py-12"><app-load-error (retry)="trail.reload()" /></div>
    } @else if (item(); as tr) {
      <section class="relative isolate overflow-hidden px-4 pt-16 pb-10 text-white">
        @if (cover(); as url) {
          <img [ngSrc]="url" fill priority sizes="100vw" alt="" class="-z-20 object-cover" [appSharedTransition]="coverTransition(tr.id)" [sharedAlways]="true" />
          <div class="absolute inset-0 -z-10 bg-linear-to-b from-black/60 via-black/50 to-black/85"></div>
        } @else {
          <div class="absolute inset-0 -z-10 bg-linear-to-br from-brand-800 via-brand-900 to-secondary-900"></div>
        }
        <div class="mx-auto max-w-5xl">
          <a [appSiteLink]="'/trails?area=' + tr.area" class="text-sm text-white/80 hover:text-white">&larr; {{ 'trails.area.' + tr.area | transloco }}</a>
          <div class="mt-3 flex flex-wrap items-center gap-2">
            <span class="rounded-full bg-surface"><span class="block rounded-full px-2.5 py-1 text-xs font-bold" [class]="gradeClass()">
              @if (tr.grade === 'trail') { {{ 'trails.grade.trail' | transloco }} } @else { {{ grade() }} }
            </span></span>
            <span class="inline-flex items-center gap-1 rounded-full bg-white/15 px-2.5 py-1 text-xs">
              <span [class]="kindIcon() + ' size-4'" aria-hidden="true"></span>{{ 'trails.kind.' + tr.kind | transloco }}
            </span>
            @if (!tr.isPublished) {
              <span class="rounded-full bg-amber-500 px-2.5 py-1 text-xs font-semibold text-black">{{ 'trails.draft' | transloco }}</span>
            }
          </div>
          <h1 class="mt-2 text-3xl font-black sm:text-4xl">{{ tr.name }}</h1>
          @if (tr.summary) {
            <p class="mt-2 max-w-2xl text-white/90">{{ tr.summary }}</p>
          }
          <dl class="mt-6 grid grid-cols-2 gap-3 sm:grid-cols-5">
            @for (stat of stats(); track stat.label) {
              <div class="flex flex-col rounded-xl bg-white/10 px-3 py-2">
                <dt class="order-last text-xs text-white/80">{{ stat.label }}</dt>
                <dd class="flex flex-col">
                  <span [class]="stat.icon + ' size-5 text-white/80'" aria-hidden="true"></span>
                  <span class="text-lg font-bold">{{ stat.value }}</span>
                </dd>
              </div>
            }
          </dl>
          <div class="mt-6 flex flex-wrap gap-2">
            <button type="button" class="ui-btn-soft bg-white/15 px-4 py-2 text-base text-white hover:bg-white/25" (click)="openReport()">
              <span class="icon-[heroicons--clipboard-document-list] size-5" aria-hidden="true"></span>{{ 'trails.conditions.add' | transloco }}
            </button>
            @if (isOwner()) {
              <a [appSiteLink]="'/trails/' + tr.slug + '/edit'" class="ui-btn-soft bg-white/15 px-4 py-2 text-base text-white hover:bg-white/25">
                <span class="icon-[heroicons--pencil-square] size-5" aria-hidden="true"></span>{{ 'listing.edit' | transloco }}
              </a>
            }
          </div>
        </div>
      </section>

      <div class="mx-auto grid max-w-5xl gap-8 px-4 py-10 lg:grid-cols-[1fr_320px]">
        <div class="flex min-w-0 flex-col gap-8">
          @if (hasMap()) {
            <section>
              <h2 class="sr-only">{{ 'listing.map.title' | transloco }}</h2>
              @defer (on viewport; hydrate on viewport) {
                <app-trail-map [track]="track()" [startLatitude]="tr.startLatitude" [startLongitude]="tr.startLongitude" [highlight]="highlight()" />
                <app-elevation-profile class="mt-3" [track]="track()" (hover)="highlight.set($event)" />
              } @placeholder {
                <div class="h-80 rounded-card border border-border bg-text/5 sm:h-96"></div>
              }
              @if (tr.gpxUrl) {
                <a [href]="media(tr.gpxUrl)" download class="mt-3 inline-flex items-center gap-1 text-sm font-medium text-brand-700 hover:underline dark:text-brand-300">
                  <span class="icon-[heroicons--arrow-down-tray] size-4" aria-hidden="true"></span>{{ 'trails.detail.downloadGpx' | transloco }}
                </a>
              }
            </section>
          }

          @if (track().length > 1) {
            @defer (on viewport) {
              <app-trail-canvas [track]="track()" [markers]="tr.markers ?? []" />
            } @placeholder { <div class="min-h-64"></div> }
            @defer (on interaction) {
              <app-trail-3d [track]="track()" [markers]="tr.markers ?? []" />
            } @placeholder {
              <button type="button" class="ui-btn-soft">{{ 'trails.editor.show3d' | transloco }}</button>
            }
          }

          @if (tr.segments.length > 0) {
            <section>
              <h2 class="mb-4 text-xl font-bold text-text">{{ 'trails.detail.segments' | transloco }}</h2>
              <ol class="relative ml-3 border-l-2 border-dashed border-border">
                @for (segment of tr.segments; track $index) {
                  <li class="relative pb-5 pl-6 last:pb-0">
                    <span class="absolute top-1 -left-[9px] size-4 rounded-full border-2 border-surface bg-brand-500" aria-hidden="true"></span>
                    <p class="font-medium text-text">{{ segment.from }} → {{ segment.to }}</p>
                    <p class="text-sm text-text-muted">{{ 'trails.units.hours' | transloco: { count: segment.hours } }}</p>
                  </li>
                }
              </ol>
            </section>
          }

          @if (tr.description) {
            <section>
              <h2 class="mb-3 text-xl font-bold text-text">{{ 'trails.detail.description' | transloco }}</h2>
              <div class="prose prose-neutral dark:prose-invert max-w-none" [innerHTML]="tr.description"></div>
            </section>
          }

          @if (tr.checklistUrl) {
            <section class="ui-card p-5">
              <h2 class="text-xl font-bold">{{ 'trails.editor.checklist' | transloco }}</h2>
              <a [href]="media(tr.checklistUrl) + '?download=1'" download class="ui-btn-primary mt-3">
                <span class="icon-[heroicons--arrow-down-tray] size-4" aria-hidden="true"></span>
                {{ 'trails.editor.downloadPdf' | transloco }}
              </a>
            </section>
          }

          <section>
            <div class="mb-4 flex items-center justify-between gap-3">
              <h2 class="text-xl font-bold text-text">{{ 'trails.conditions.title' | transloco }}</h2>
              <button type="button" class="ui-btn-soft" (click)="openReport()">
                <span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>{{ 'trails.conditions.add' | transloco }}
              </button>
            </div>
            @defer (on viewport; hydrate on viewport) {
              @if (reportList().length > 0) {
                <div class="flex flex-col gap-3">
                  @for (report of reportList(); track report.id) {
                    <app-condition-card [report]="report" [canDelete]="report.author.id === auth.user()?.id" (delete)="deleteReport(report)" />
                  }
                </div>
              } @else if (reports.hasValue()) {
                <p class="rounded-card border border-dashed border-border px-6 py-8 text-center text-sm text-text-muted">{{ 'trails.conditions.empty' | transloco }}</p>
              }
            } @placeholder {
              <div class="min-h-32"></div>
            }
          </section>
        </div>

        <aside class="flex flex-col gap-6">
          <div class="ui-card p-4">
            <p class="mb-2 text-sm font-semibold text-text">{{ 'trails.conditions.latest' | transloco }}</p>
            <app-condition-badge [summary]="tr.latestCondition" />
          </div>


          @if (gear().length > 0) {
            <section class="ui-card p-4">
              <h2 class="mb-3 font-semibold text-text">{{ 'trails.detail.gear' | transloco }}</h2>
              <ul class="grid grid-cols-2 gap-2 text-sm text-text">
                @for (entry of gear(); track entry.value) {
                  <li class="flex items-center gap-2">
                    <span [class]="entry.icon + ' size-5 shrink-0 text-brand-600 dark:text-brand-300'" aria-hidden="true"></span>{{ 'trails.gear.' + entry.value | transloco }}
                  </li>
                }
              </ul>
            </section>
          }

          @if (seasons().length > 0) {
            <section class="ui-card p-4">
              <h2 class="mb-3 font-semibold text-text">{{ 'trails.detail.seasons' | transloco }}</h2>
              <ul class="flex flex-wrap gap-2">
                @for (season of seasons(); track season.value) {
                  <li class="inline-flex items-center gap-1 rounded-full bg-text/5 px-2.5 py-1 text-sm text-text">
                    <span [class]="season.icon + ' size-4'" aria-hidden="true"></span>{{ 'trails.season.' + season.value | transloco }}
                  </li>
                }
              </ul>
            </section>
          }

          @if (tr.gettingThere) {
            <section class="ui-card p-4">
              <h2 class="mb-2 flex items-center gap-2 font-semibold text-text">
                <span class="icon-[heroicons--map-pin] size-5 text-brand-600 dark:text-brand-300" aria-hidden="true"></span>{{ 'trails.detail.gettingThere' | transloco }}
              </h2>
              <p class="text-sm whitespace-pre-line text-text">{{ tr.gettingThere }}</p>
            </section>
          }
        </aside>
      </div>
    }
  `,
})
export class TrailPage {
  readonly slug = input.required<string>();

  protected readonly auth = inject(AuthStore);
  private readonly http = inject(HttpClient);
  private readonly dialog = inject(Dialog);
  private readonly router = inject(Router);
  private readonly transloco = inject(TranslocoService);
  private readonly apiBaseUrl = inject(APP_CONFIG).apiBaseUrl;
  private readonly navigate = injectSiteNavigate();

  protected readonly trail = httpResource<Trail>(
    () => `/trails/${encodeURIComponent(this.slug())}`,
  );
  protected readonly reports = httpResource<ConditionReport[]>(() => ({
    url: `/trails/${encodeURIComponent(this.slug())}/conditions`,
    params: { limit: REPORTS_LIMIT },
  }));

  protected readonly item = computed(() => loaded(this.trail));
  protected readonly reportList = computed(() => loaded(this.reports) ?? []);
  protected readonly notFound = computed(() => {
    const error = this.trail.error();
    return error instanceof HttpErrorResponse && error.status === 404;
  });
  protected readonly isOwner = computed(() => {
    const user = this.auth.user();
    return Boolean(user && this.item()?.userId === user.id);
  });
  protected readonly highlight = signal<number | null>(null);

  protected readonly track = computed(() => this.item()?.track ?? []);
  protected readonly hasMap = computed(() => {
    const tr = this.item();
    return Boolean(
      tr && (this.track().length > 1 || (tr.startLatitude !== null && tr.startLongitude !== null)),
    );
  });
  protected readonly cover = computed(() => {
    const tr = this.item();
    const url = tr ? (tr.coverImageUrl ?? tr.photos[0]) : undefined;
    return url ? this.media(url) : null;
  });
  protected readonly grade = computed(() => formatGrade(this.item()?.grade ?? ''));
  protected readonly gradeClass = computed(() => gradeClass(this.item()?.grade ?? ''));
  protected readonly kindIcon = computed(() => optionIcon(KINDS, this.item()?.kind ?? ''));
  protected readonly gear = computed(() =>
    (this.item()?.gear ?? []).map((value) => ({ value, icon: optionIcon(GEAR, value) })),
  );
  protected readonly seasons = computed(() =>
    (this.item()?.seasons ?? []).map((value) => ({ value, icon: optionIcon(SEASONS, value) })),
  );

  protected readonly stats = computed((): Stat[] => {
    const tr = this.item();
    if (!tr) return [];
    const t = (key: string, params?: Record<string, unknown>): string =>
      this.transloco.translate(key, params);
    const m = t('trails.units.m');
    const gain = tr.elevationGain ?? tr.trackStats?.elevationGain ?? null;
    const distance = tr.distanceKm ?? tr.trackStats?.distanceKm ?? null;
    const stats: (Stat | null)[] = [
      tr.summitElevation
        ? {
            icon: 'icon-[heroicons--flag]',
            value: `${tr.summitElevation} ${m}`,
            label: t('trails.detail.summit'),
          }
        : null,
      gain
        ? {
            icon: 'icon-[heroicons--arrow-trending-up]',
            value: `+${gain} ${m}`,
            label: t('trails.detail.gain'),
          }
        : null,
      distance
        ? {
            icon: 'icon-[heroicons--arrows-right-left]',
            value: `${distance} ${t('trails.units.km')}`,
            label: t('trails.detail.distance'),
          }
        : null,
      tr.durationHours
        ? {
            icon: 'icon-[heroicons--clock]',
            value: t('trails.units.hours', { count: tr.durationHours }),
            label: t('trails.detail.duration'),
          }
        : null,
      {
        icon: 'icon-[heroicons--calendar-days]',
        value: t('trails.units.days', { count: tr.days }),
        label: t('trails.detail.days'),
      },
    ];
    return stats.filter((stat): stat is Stat => stat !== null);
  });

  constructor() {
    const seo = inject(SeoService);
    bindPageMeta(() => {
      const tr = this.item();
      if (!tr) return undefined;
      const path = `/trails/${tr.slug}`;
      const description =
        tr.summary || plainText(tr.description, 160) || this.transloco.translate('trails.subtitle');
      const area = this.transloco.translate(`trails.area.${tr.area}`);
      return {
        title: tr.name,
        description,
        image: this.cover(),
        robots: tr.isPublished ? undefined : 'noindex',
        jsonLd: [
          touristAttraction({
            url: seo.pageUrl(path),
            name: tr.name,
            description,
            image: this.cover(),
            latitude: tr.startLatitude,
            longitude: tr.startLongitude,
            elevation: tr.summitElevation,
            area,
          }),
          seo.breadcrumbs([
            { name: this.transloco.translate('trails.title'), path: '/trails' },
            { name: tr.name, path },
          ]),
        ],
      };
    });
  }

  protected media(url: string): string {
    return resolveMediaUrl(this.apiBaseUrl, url);
  }

  // Signed in: the report form; otherwise sign in first and come back.
  protected async openReport(): Promise<void> {
    const tr = this.item();
    if (!tr) return;
    if (!(await this.auth.ensureSession())) {
      await this.navigate(`/login?redirect=${encodeURIComponent(this.router.url)}`);
      return;
    }
    const ref = this.dialog.open<boolean, ConditionReportData>(ConditionReportDialog, {
      data: { trailId: tr.id },
      ariaLabel: this.transloco.translate('trails.conditions.formTitle'),
    });
    ref.closed.subscribe((saved) => {
      if (!saved) return;
      this.reports.reload();
      this.trail.reload();
    });
  }

  protected async deleteReport(report: ConditionReport): Promise<void> {
    await firstValueFrom(this.http.delete(`/conditions/${report.id}`));
    this.reports.reload();
    this.trail.reload();
  }

  // The cover the route's card flew in from (motion.css).
  protected coverTransition(id: string): string {
    return trailCoverTransition(id);
  }
}
