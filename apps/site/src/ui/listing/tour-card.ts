import type { Tour } from '@/core/api/types';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount, formatDateRange } from '@/core/lib/format';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { SiteLink } from '@/ui/site-link';
import { NgOptimizedImage } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// A tour in a grid: cover, dates, where, title and price.
@Component({
  selector: 'app-tour-card',
  imports: [NgOptimizedImage, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <a [appSiteLink]="'/tours/' + tour().slug" class="group flex h-full flex-col overflow-hidden rounded-card border border-border bg-surface transition hover:shadow-lg">
      <div class="relative aspect-[4/3] overflow-hidden bg-slate-100 dark:bg-slate-800">
        @if (cover(); as url) {
          <img [ngSrc]="url" fill sizes="(min-width: 1024px) 30vw, (min-width: 640px) 50vw, 100vw" alt="" class="object-cover transition group-hover:scale-105" />
        } @else {
          <span class="icon-[mdi--compass-outline] absolute inset-0 m-auto size-12 text-text-muted" aria-hidden="true"></span>
        }
        <span class="absolute top-2 left-2 inline-flex items-center gap-1 rounded-full bg-black/65 px-2.5 py-1 text-xs font-medium text-white">
          <span class="icon-[heroicons--calendar-days] size-3.5" aria-hidden="true"></span>{{ dates() }}
        </span>
      </div>
      <div class="flex flex-1 flex-col gap-2 p-4">
        @if (tour().destination || tour().location) {
          <p class="flex items-center gap-1 text-sm text-text-muted">
            <span class="icon-[heroicons--map-pin] size-4" aria-hidden="true"></span>
            @if (tour().destination) { {{ 'destinations.' + tour().destination | transloco }} } @else { {{ tour().location }} }
          </p>
        }
        <h3 class="text-lg leading-snug font-semibold text-text">{{ tour().title }}</h3>
        <p class="mt-auto flex items-baseline justify-between gap-2 pt-2 text-sm">
          @if (tour().durationDays; as days) {
            <span class="text-text-muted">{{ 'tours.durationValue' | transloco: { days: days } }}</span>
          } @else {
            <span></span>
          }
          @if (tour().price; as price) {
            <span class="font-semibold text-brand-700 dark:text-brand-300">{{ 'tours.priceValue' | transloco: { price: amount(price) } }}</span>
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
  protected readonly dates = computed(() =>
    formatDateRange(this.lang(), this.tour().startDate, this.tour().endDate),
  );

  protected amount(price: number): string {
    return formatAmount(this.lang(), price);
  }
}
