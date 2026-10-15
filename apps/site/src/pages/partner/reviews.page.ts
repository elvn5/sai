import type { Paginated, PartnerReview } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { formatInBishkek } from '@/core/lib/format';
import { contentLang, listingPath } from '@/core/lib/partner-cabinet';
import { loaded } from '@/core/lib/resource';
import { bindPageMeta } from '@/core/seo/seo.service';
import { CabinetStore } from '@/pages/partner/cabinet-store';
import { ReviewReplyForm } from '@/pages/partner/review-reply-form';
import { PhotoGrid } from '@/ui/photo-grid';
import { RatingStars } from '@/ui/rating-stars';
import { SiteLink } from '@/ui/site-link';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

const LIMIT = 50;

// /partner/reviews: reviews of the partner's listings, all or those still
// without an answer, each with the owner's reply form.
@Component({
  selector: 'app-partner-reviews-page',
  imports: [PhotoGrid, RatingStars, ReviewReplyForm, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-6">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 class="text-2xl font-extrabold text-text sm:text-3xl">{{ 'cabinet.nav.reviews' | transloco }}</h1>
          <p class="mt-1 text-sm text-text-muted">{{ 'cabinet.reviews.subtitle' | transloco }}</p>
        </div>
        <div class="flex rounded-lg border border-border bg-surface p-0.5" role="group" [attr.aria-label]="'cabinet.nav.reviews' | transloco">
          @for (option of filters; track option.key) {
            <button type="button" class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors" [class]="unanswered() === option.value ? 'bg-brand-600 text-white' : 'text-text-muted hover:text-text'" [attr.aria-pressed]="unanswered() === option.value" (click)="unanswered.set(option.value)">
              {{ option.key | transloco }}
            </button>
          }
        </div>
      </div>

      @if (list().length > 0) {
        <ul class="flex flex-col gap-3">
          @for (review of list(); track review.id) {
            <li>
              <article class="ui-card flex flex-col gap-3 px-4 py-4">
                <a [appSiteLink]="path(review.listing)" [lang]="lang(review.listing.locale)" class="self-start text-xs font-semibold tracking-wide text-text-muted uppercase hover:underline">{{ review.listing.title }}</a>
                <div class="flex items-center justify-between gap-3">
                  <div>
                    <p class="font-medium text-text">{{ author(review) }}</p>
                    <p class="text-xs text-text-muted">{{ date(review.createdAt) }}</p>
                  </div>
                  <app-rating-stars [value]="review.rating" />
                </div>
                <p class="whitespace-pre-line text-text">{{ review.text }}</p>
                @if (review.photos.length > 0) {
                  <app-photo-grid class="max-w-md" [photos]="review.photos" />
                }
                <app-review-reply-form [review]="review" (changed)="refresh()" />
              </article>
            </li>
          }
        </ul>
      } @else if (reviews.hasValue()) {
        <div class="flex flex-col items-center gap-2 rounded-card border border-dashed border-border px-6 py-12 text-center">
          <span class="icon-[heroicons--chat-bubble-left-right] size-10 text-text-muted" aria-hidden="true"></span>
          <p class="font-medium text-text">{{ (unanswered() ? 'cabinet.reviews.allAnswered' : 'cabinet.reviews.empty') | transloco }}</p>
          <p class="max-w-sm text-sm text-text-muted">{{ 'cabinet.reviews.why' | transloco }}</p>
        </div>
      }
    </div>
  `,
})
export class PartnerReviewsPage {
  private readonly store = inject(CabinetStore);
  private readonly transloco = inject(TranslocoService);
  private readonly currentLang = injectLang();
  protected readonly filters = [
    { key: 'cabinet.reviews.all', value: false },
    { key: 'cabinet.reviews.unanswered', value: true },
  ];
  protected readonly unanswered = signal(false);

  protected readonly reviews = httpResource<Paginated<PartnerReview>>(() => ({
    url: '/reviews/mine',
    params: { limit: LIMIT, ...(this.unanswered() ? { unanswered: true } : {}) },
  }));
  protected readonly list = computed(() => loaded(this.reviews)?.items ?? []);
  protected readonly path = listingPath;
  protected readonly lang = contentLang;

  constructor() {
    bindPageMeta(() => ({
      title: this.transloco.translate('cabinet.nav.reviews'),
      description: this.transloco.translate('cabinet.reviews.subtitle'),
      robots: 'noindex',
    }));
  }

  protected author(review: PartnerReview): string {
    return (
      review.author.firstName ??
      review.author.username ??
      this.transloco.translate('reviews.anonymous')
    );
  }

  protected date(iso: string): string {
    return formatInBishkek(this.currentLang(), iso, {
      day: 'numeric',
      month: 'long',
      year: 'numeric',
    });
  }

  protected refresh(): void {
    this.reviews.reload();
    this.store.refreshReviews();
  }
}
