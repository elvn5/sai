import type { Trail } from '@/core/api/types';
import { APP_CONFIG } from '@/core/config/app-config';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { FRESH_REPORT_DAYS, daysSince, formatGrade } from '@/core/lib/trail-options';
import { trailCoverTransition } from '@/core/lib/view-transitions';
import { SharedTransition } from '@/ui/shared-transition';
import { SiteLink } from '@/ui/site-link';
import { NgOptimizedImage } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// The condition chip over the photo: color, icon and words together.
const SNOWY = ['patchy_snow', 'snow'];
const HAZARD = ['deep_snow', 'ice'];

// A route in the catalog grid: photo with the latest condition, area, name,
// grade, elevation gain and time. On hover the flag takes a step.
@Component({
  selector: 'app-trail-card',
  imports: [NgOptimizedImage, SharedTransition, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <a [appSiteLink]="'/trails/' + trail().slug" class="group flex h-full flex-col overflow-hidden rounded-card border border-border bg-surface shadow-[0_1px_2px_rgb(15_23_42/0.06)] motion-lift dark:shadow-none">
      <div class="relative aspect-[16/10] overflow-hidden bg-linear-to-br from-brand-700 to-secondary-700" [appSharedTransition]="coverTransition()">
        @if (cover(); as url) {
          <img [ngSrc]="url" fill [priority]="priority()" sizes="(min-width: 1024px) 33vw, (min-width: 640px) 50vw, 100vw" alt="" class="motion-zoom object-cover" />
        } @else {
          <span class="icon-[heroicons--flag] absolute inset-0 m-auto size-12 text-white/40" aria-hidden="true"></span>
        }
        <span class="ui-chip-on-image absolute top-2 left-2" [class]="condition().chip">
          @switch (condition().tone) {
            @case ('ok') { <span class="icon-[heroicons--check-circle] size-3.5" aria-hidden="true"></span> }
            @case ('none') { <span class="icon-[heroicons--question-mark-circle] size-3.5" aria-hidden="true"></span> }
            @default { <span class="icon-[heroicons--exclamation-triangle] size-3.5" aria-hidden="true"></span> }
          }
          @if (condition().state; as state) {
            {{ 'trails.state.' + state | transloco }}
          } @else {
            {{ 'trails.conditions.noFresh' | transloco }}
          }
        </span>
      </div>
      <div class="flex flex-1 flex-col gap-1.5 px-3.5 pt-3 pb-3.5">
        <p class="flex items-center gap-1 text-[0.8rem] text-text-muted">
          <span class="icon-[heroicons--map-pin] size-4" aria-hidden="true"></span>{{ 'trails.area.' + trail().area | transloco }}
        </p>
        <h2 class="text-base leading-snug font-semibold text-text">{{ trail().name }}</h2>
        <p class="mt-auto flex flex-wrap gap-x-3 gap-y-1 pt-1 text-[0.8rem] text-text-muted tabular-nums">
          <span class="inline-flex items-center gap-1">
            <span class="icon-[heroicons--flag] icon-anim-step size-4" aria-hidden="true"></span>
            @if (trail().grade === 'trail') { {{ 'trails.grade.trail' | transloco }} } @else { {{ grade() }} }
          </span>
          @if (trail().elevationGain; as gain) {
            <span class="inline-flex items-center gap-1"><span class="icon-[heroicons--arrow-trending-up] size-4" aria-hidden="true"></span>{{ gain }} {{ 'trails.units.m' | transloco }}</span>
          }
          @if (trail().days > 1) {
            <span class="inline-flex items-center gap-1"><span class="icon-[heroicons--clock] size-4" aria-hidden="true"></span>{{ 'trails.units.daysShort' | transloco: { count: trail().days } }}</span>
          } @else if (trail().durationHours; as hours) {
            <span class="inline-flex items-center gap-1"><span class="icon-[heroicons--clock] size-4" aria-hidden="true"></span>{{ 'trails.units.hours' | transloco: { count: hours } }}</span>
          }
        </p>
      </div>
    </a>
  `,
})
export class TrailCard {
  readonly trail = input.required<Trail>();
  readonly priority = input(false);
  private readonly apiBaseUrl = inject(APP_CONFIG).apiBaseUrl;

  protected readonly cover = computed(() => {
    const url = this.trail().coverImageUrl ?? this.trail().photos[0];
    return url ? resolveMediaUrl(this.apiBaseUrl, url) : null;
  });
  // The cover flies to the route page's hero once clicked.
  protected readonly coverTransition = computed(() => trailCoverTransition(this.trail().id));
  protected readonly grade = computed(() => formatGrade(this.trail().grade));
  // A report older than two weeks says nothing about today.
  protected readonly condition = computed(() => {
    const report = this.trail().latestCondition;
    if (!report || daysSince(report.observedOn) > FRESH_REPORT_DAYS) {
      return { tone: 'none', state: null, chip: '' };
    }
    const state = report.trailState;
    if (HAZARD.includes(state)) return { tone: 'danger', state, chip: '!bg-red-700/90' };
    if (SNOWY.includes(state)) return { tone: 'warning', state, chip: '!bg-amber-700/92' };
    return { tone: 'ok', state, chip: '!bg-green-700/90' };
  });
}
