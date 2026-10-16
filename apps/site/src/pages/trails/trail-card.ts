import type { Trail } from '@/core/api/types';
import { APP_CONFIG } from '@/core/config/app-config';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { KINDS, formatGrade, gradeClass, optionIcon } from '@/core/lib/trail-options';
import { ConditionBadge } from '@/pages/trails/condition-badge';
import { SiteLink } from '@/ui/site-link';
import { NgOptimizedImage } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

@Component({
  selector: 'app-trail-card',
  imports: [ConditionBadge, NgOptimizedImage, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <a [appSiteLink]="'/trails/' + trail().slug" class="group flex h-full flex-col overflow-hidden rounded-card border border-border bg-surface motion-lift">
      <div class="relative aspect-video overflow-hidden bg-linear-to-br from-brand-700 to-secondary-700">
        @if (cover(); as url) {
          <img [ngSrc]="url" fill [priority]="priority()" sizes="(min-width: 1024px) 33vw, (min-width: 640px) 50vw, 100vw" alt="" class="motion-zoom object-cover" />
        } @else {
          <span [class]="kindIcon() + ' absolute inset-0 m-auto size-14 text-white/40'" aria-hidden="true"></span>
        }
        <!-- Solid under the tint: readable over any photo. -->
        <span class="absolute top-2 left-2 rounded-full bg-surface">
          <span class="block rounded-full px-2.5 py-1 text-xs font-bold" [class]="gradeClass()">
            @if (trail().grade === 'trail') { {{ 'trails.grade.trail' | transloco }} } @else { {{ grade() }} }
          </span>
        </span>
        @if (trail().summitElevation; as summit) {
          <span class="absolute right-2 bottom-2 rounded-full bg-black/65 px-2.5 py-1 text-xs font-semibold text-white">▲ {{ summit }} {{ 'trails.units.m' | transloco }}</span>
        }
      </div>
      <div class="flex flex-1 flex-col gap-2 p-4">
        <p class="flex items-center gap-1 text-sm text-text-muted">
          <span [class]="kindIcon() + ' size-4'" aria-hidden="true"></span>
          {{ 'trails.kind.' + trail().kind | transloco }} · {{ 'trails.area.' + trail().area | transloco }}
        </p>
        <h2 class="text-lg leading-snug font-semibold text-text">{{ trail().name }}</h2>
        @if (trail().summary) {
          <p class="line-clamp-2 text-sm text-text-muted">{{ trail().summary }}</p>
        }
        <p class="flex flex-wrap gap-x-4 gap-y-1 text-xs text-text-muted">
          @if (trail().durationHours; as hours) {
            <span class="inline-flex items-center gap-1"><span class="icon-[mdi--clock-outline] size-3.5" aria-hidden="true"></span>{{ 'trails.units.hours' | transloco: { count: hours } }}</span>
          }
          @if (trail().days > 1) {
            <span class="inline-flex items-center gap-1"><span class="icon-[mdi--calendar-range] size-3.5" aria-hidden="true"></span>{{ 'trails.units.days' | transloco: { count: trail().days } }}</span>
          }
          @if (trail().distanceKm; as km) {
            <span class="inline-flex items-center gap-1"><span class="icon-[mdi--map-marker-distance] size-3.5" aria-hidden="true"></span>{{ km }} {{ 'trails.units.km' | transloco }}</span>
          }
        </p>
        <div class="mt-auto flex flex-wrap items-center justify-between gap-2 pt-2">
          <app-condition-badge [summary]="trail().latestCondition" />
          @if (trail().upcomingOutings > 0) {
            <span class="inline-flex items-center gap-1 text-xs font-semibold text-brand-700 dark:text-brand-300">
              <span class="icon-[mdi--account-group] size-4" aria-hidden="true"></span>{{ 'trails.outingsCount' | transloco: { count: trail().upcomingOutings } }}
            </span>
          }
        </div>
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
  protected readonly kindIcon = computed(() => optionIcon(KINDS, this.trail().kind));
  protected readonly gradeClass = computed(() => gradeClass(this.trail().grade));
  protected readonly grade = computed(() => formatGrade(this.trail().grade));
}
