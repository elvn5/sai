import type { LocalGuide } from '@/core/api/types';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import { formatAmount } from '@/core/lib/format';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { RatingStars } from '@/ui/rating-stars';
import { SiteLink } from '@/ui/site-link';
import { NgOptimizedImage } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

const CARD_DESTINATIONS = 3;

@Component({
  selector: 'app-local-guide-card',
  imports: [NgOptimizedImage, RatingStars, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <a [appSiteLink]="'/local-guides/' + guide().slug" class="flex h-full flex-col gap-3 rounded-card border border-border bg-surface p-4 transition hover:shadow-lg">
      <div class="flex items-center gap-3">
        @if (avatar(); as url) {
          <img [ngSrc]="url" width="64" height="64" [priority]="priority()" alt="" class="size-16 shrink-0 rounded-full object-cover" />
        } @else {
          <span class="flex size-16 shrink-0 items-center justify-center rounded-full bg-linear-to-br from-brand-500 to-secondary-500 text-xl font-bold text-white" aria-hidden="true">{{ initial() }}</span>
        }
        <div class="min-w-0">
          <h2 class="truncate text-lg font-semibold text-text">{{ guide().name }}</h2>
          @if (guide().headline) {
            <p class="line-clamp-2 text-sm text-text-muted">{{ guide().headline }}</p>
          }
        </div>
      </div>
      <div class="flex items-center gap-2 text-sm">
        @if (guide().reviewCount > 0) {
          <span class="font-semibold text-text">{{ guide().ratingAverage.toFixed(1) }}</span>
          <app-rating-stars [value]="guide().ratingAverage" />
        }
        <span class="text-text-muted">{{ 'reviews.count' | transloco: { count: guide().reviewCount } }}</span>
      </div>
      <ul class="flex flex-wrap gap-1.5">
        @for (language of guide().languages; track language) {
          <li class="ui-badge bg-text/5 text-text" [attr.title]="'localGuides.languages.' + language | transloco">{{ language.toUpperCase() }}</li>
        }
        @if (guide().hasVehicle) {
          <li class="ui-badge inline-flex items-center gap-1 bg-secondary-500/15 text-secondary-800 dark:text-secondary-200">
            <span class="icon-[heroicons--truck] size-3" aria-hidden="true"></span>{{ 'localGuides.vehicleBadge' | transloco }}
          </li>
        }
      </ul>
      <p class="flex flex-wrap items-center gap-1 text-sm text-text-muted">
        <span class="icon-[heroicons--map-pin] size-4" aria-hidden="true"></span>
        @for (destination of shownDestinations(); track destination; let last = $last) {
          <span>{{ 'destinations.' + destination | transloco }}@if (!last) {,}</span>
        }
        @if (moreDestinations() > 0) {
          <span>+{{ moreDestinations() }}</span>
        }
      </p>
      <div class="mt-auto flex items-end justify-between pt-1">
        @if (guide().experienceYears; as years) {
          <span class="text-sm text-text-muted">{{ 'localGuides.experienceValue' | transloco: { years: years } }}</span>
        }
        @if (guide().pricePerDay !== null) {
          <span class="ml-auto text-right">
            <span class="text-lg font-bold text-brand-700 dark:text-brand-300">{{ price() }} {{ 'listing.currency' | transloco }}</span>
            <span class="text-sm text-text-muted"> / {{ 'listing.perDay' | transloco }}</span>
          </span>
        }
      </div>
    </a>
  `,
})
export class LocalGuideCard {
  readonly guide = input.required<LocalGuide>();
  readonly priority = input(false);
  private readonly apiBaseUrl = inject(APP_CONFIG).apiBaseUrl;
  private readonly lang = injectLang();

  protected readonly avatar = computed(() => {
    const url = this.guide().avatarUrl ?? this.guide().photos[0];
    return url ? resolveMediaUrl(this.apiBaseUrl, url) : null;
  });
  protected readonly initial = computed(() => this.guide().name.charAt(0).toUpperCase());
  protected readonly shownDestinations = computed(() =>
    this.guide().destinations.slice(0, CARD_DESTINATIONS),
  );
  protected readonly moreDestinations = computed(
    () => this.guide().destinations.length - CARD_DESTINATIONS,
  );
  protected readonly price = computed(() =>
    formatAmount(this.lang(), this.guide().pricePerDay ?? 0),
  );
}
