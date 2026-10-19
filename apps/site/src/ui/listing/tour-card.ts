import type { Tour } from '@/core/api/types';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, formatDateRange } from '@/core/lib/format';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { tourCoverTransition } from '@/core/lib/view-transitions';
import { FavoriteButton } from '@/ui/favorite-button';
import { SharedTransition } from '@/ui/shared-transition';
import { SiteLink } from '@/ui/site-link';
import { NgOptimizedImage } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// A tour in a grid: cover, dates, where, title and price.
@Component({
  selector: 'app-tour-card',
  imports: [FavoriteButton, NgOptimizedImage, SharedTransition, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <a [appSiteLink]="'/tours/' + tour().slug" class="group flex h-full flex-col overflow-hidden rounded-card border border-border bg-surface shadow-[0_1px_2px_rgb(15_23_42/0.06)] motion-lift dark:shadow-none">
      <div class="relative aspect-[16/10] overflow-hidden bg-slate-200 dark:bg-slate-800" [appSharedTransition]="coverTransition()">
        @if (cover(); as url) {
          <img [ngSrc]="url" fill sizes="(min-width: 1024px) 30vw, (min-width: 640px) 50vw, 100vw" alt="" class="motion-zoom object-cover" />
        } @else {
          <span class="icon-[heroicons--map] absolute inset-0 m-auto size-12 text-text-muted" aria-hidden="true"></span>
        }
        <span class="ui-chip-on-image absolute top-2 left-2">
          <span class="icon-[heroicons--calendar-days] size-3.5" aria-hidden="true"></span>{{ dates() }}
        </span>
        <span class="absolute top-2 right-2"><app-favorite-button type="tour" [targetId]="tour().id" /></span>
      </div>
      <div class="flex flex-1 flex-col gap-1.5 px-3.5 pt-3 pb-3.5">
        @if (tour().destination || tour().location) {
          <p class="flex items-center gap-1 text-[0.8rem] text-text-muted">
            <span class="icon-[heroicons--map-pin] icon-anim-bounce size-4" aria-hidden="true"></span>
            @if (tour().destination) { {{ 'destinations.' + tour().destination | transloco }} } @else { {{ tour().location }} }
          </p>
        }
        <h3 class="text-base leading-snug font-semibold text-text">{{ tour().title }}</h3>
        <p class="mt-auto flex items-baseline justify-between gap-2 pt-1 text-[0.8125rem]">
          @if (tour().durationDays; as days) {
            <span class="text-text-muted">{{ 'tours.durationValue' | transloco: { days: days } }}</span>
          } @else {
            <span></span>
          }
          @if (tour().price; as price) {
            <span class="font-bold text-brand-700 dark:text-brand-300">{{ 'tours.priceValue' | transloco: { price: amount(price) } }}</span>
          }
        </p>
      </div>
    </a>
  `,
})
export class TourCard {
  readonly tour = input.required<Tour>();
  private readonly apiBaseUrl = inject(APP_CONFIG).apiBaseUrl;
  private readonly lang = injectLang();

  protected readonly cover = computed(() => {
    const url = this.tour().coverImageUrl;
    return url ? resolveMediaUrl(this.apiBaseUrl, url) : null;
  });
  // The cover flies to the tour page's once clicked (motion.css).
  protected readonly coverTransition = computed(() => tourCoverTransition(this.tour().id));
  protected readonly dates = computed(() =>
    formatDateRange(this.lang(), this.tour().startDate, this.tour().endDate),
  );

  protected amount(price: number): string {
    return formatAmount(this.lang(), price);
  }
}
