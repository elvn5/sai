import type { Review, ReviewTargetType, ReviewsPage } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { injectLang } from '@/core/i18n/current-lang';
import { injectConfirm } from '@/ui/confirm-dialog';
import { FieldErrors } from '@/ui/field-errors';
import { REVIEWS_PAGE_SIZE } from '@/ui/listing/reviews';
import { PersonAvatar } from '@/ui/person-avatar';
import { PhotoGrid } from '@/ui/photo-grid';
import { PhotoUploader } from '@/ui/photo-uploader';
import { RatingInput } from '@/ui/rating-input';
import { RatingStars } from '@/ui/rating-stars';
import { SiteLink } from '@/ui/site-link';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  linkedSignal,
  output,
  signal,
} from '@angular/core';
import {
  FormField,
  form,
  maxLength,
  min,
  submit,
  validateStandardSchema,
} from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';
import { z } from 'zod/mini';

const MAX_PHOTOS = 6;

interface ReviewForm {
  rating: number;
  text: string;
  photos: string[];
}

// A listing's reviews: the rating summary, the list (first page from the
// page, the rest loaded here), writing one (signed-in visitors other than
// the owner, once) and deleting your own.
@Component({
  selector: 'app-reviews-section',
  // A box of its own: deferred blocks observe their root element.
  host: { class: 'block' },
  imports: [
    FieldErrors,
    FormField,
    PersonAvatar,
    PhotoGrid,
    PhotoUploader,
    RatingInput,
    RatingStars,
    SiteLink,
    TranslocoPipe,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section id="reviews" class="scroll-mt-24" aria-labelledby="reviews-heading">
      <h2 id="reviews-heading" class="text-2xl font-bold text-text">{{ 'reviews.heading' | transloco }}</h2>

      @if (summary(); as stats) {
        @if (stats.count > 0) {
          <div class="mt-4 flex flex-col gap-6 sm:flex-row sm:items-center">
            <div class="flex flex-col items-center sm:w-40">
              <span class="text-5xl font-extrabold text-text">{{ stats.average.toFixed(1) }}</span>
              <app-rating-stars [value]="stats.average" size="md" />
              <span class="mt-1 text-sm text-text-muted">{{ 'reviews.count' | transloco: { count: stats.count } }}</span>
            </div>
            <dl class="flex flex-1 flex-col gap-1.5">
              @for (star of [5, 4, 3, 2, 1]; track star) {
                <div class="flex items-center gap-2 text-sm">
                  <dt class="flex w-8 items-center gap-0.5 text-text-muted">
                    {{ star }}<span class="icon-[heroicons--star-solid] size-4 text-amber-400" aria-hidden="true"></span>
                  </dt>
                  <dd class="flex flex-1 items-center gap-2">
                    <span class="h-2 flex-1 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-700" aria-hidden="true">
                      <span class="block h-full rounded-full bg-amber-400" [style.width]="share(star)"></span>
                    </span>
                    <span class="w-6 text-right text-text-muted">{{ stats.distribution[star] ?? 0 }}</span>
                  </dd>
                </div>
              }
            </dl>
          </div>
        }
      }

      @if (canWrite()) {
        <form class="ui-card mt-6 flex flex-col gap-4 p-4" novalidate (submit)="send($event)">
          <h3 class="font-semibold text-text">{{ 'reviews.writeTitle' | transloco }}</h3>
          <div class="ui-field">
            <span id="review-rating-label">{{ 'reviews.ratingLabel' | transloco }}</span>
            <app-rating-input [label]="'reviews.ratingLabel' | transloco" [formField]="reviewForm.rating" />
            <app-field-errors [state]="reviewForm.rating()" />
          </div>
          <label class="ui-field">
            {{ 'reviews.textLabel' | transloco }}
            <textarea class="ui-input" rows="4" [placeholder]="'reviews.textPlaceholder' | transloco" [formField]="reviewForm.text"></textarea>
            <app-field-errors [state]="reviewForm.text()" />
          </label>
          <div class="ui-field">
            <span>{{ 'reviews.photosLabel' | transloco }}</span>
            <span class="ui-hint">{{ 'reviews.photosHint' | transloco }}</span>
            <app-photo-uploader folder="reviews" [limit]="maxPhotos" [formField]="reviewForm.photos" />
          </div>
          @if (sendError(); as key) {
            <p class="ui-error" role="alert">{{ key | transloco }}</p>
          }
          <button type="submit" class="ui-btn-primary self-start" [disabled]="reviewForm().submitting()">{{ 'reviews.submit' | transloco }}</button>
        </form>
      } @else if (auth.checked() && !auth.user()) {
        <p class="mt-6 text-sm text-text-muted">
          <a appSiteLink="/login" class="text-brand-700 hover:underline dark:text-brand-300">{{ 'login.cta' | transloco }}</a>
          {{ 'reviews.loginToReview' | transloco }}
        </p>
      }

      @if (reviews().length > 0) {
        <ul class="mt-6 flex flex-col divide-y divide-border">
          @for (review of reviews(); track review.id) {
            <li class="flex flex-col gap-2 py-5">
              <div class="flex items-center gap-3">
                <app-person-avatar [person]="review.author" />
                <div class="flex-1">
                  <p class="font-medium text-text">{{ authorName(review) }}</p>
                  <p class="text-xs text-text-muted"><time [attr.datetime]="review.createdAt">{{ date(review.createdAt) }}</time></p>
                </div>
                <app-rating-stars [value]="review.rating" />
              </div>
              <p class="whitespace-pre-line text-text">{{ review.text }}</p>
              @if (review.photos.length > 0) {
                <app-photo-grid class="max-w-md" [photos]="review.photos" />
              }
              @if (review.reply; as reply) {
                <div class="ml-4 rounded-lg border-l-4 border-brand-500 bg-brand-500/5 px-4 py-3">
                  <p class="text-xs font-semibold text-brand-700 dark:text-brand-300">{{ 'reviews.ownerReply' | transloco }} · {{ date(reply.repliedAt) }}</p>
                  <p class="mt-1 text-sm whitespace-pre-line text-text">{{ reply.text }}</p>
                </div>
              }
              @if (review.author.id === auth.user()?.id) {
                <button type="button" class="ui-btn-ghost ui-btn-xs self-start" [disabled]="deleting()" (click)="remove(review)">{{ 'reviews.delete' | transloco }}</button>
              }
            </li>
          }
        </ul>
      } @else if (page()) {
        <p class="mt-4 text-text-muted">{{ 'reviews.empty' | transloco }}</p>
      }

      @if (hasMore()) {
        <button type="button" class="ui-btn-soft mt-2" [disabled]="loadingMore()" (click)="loadMore()">
          {{ (loadingMore() ? 'gallery.loadingMore' : 'listing.loadMore') | transloco }}
        </button>
      }
    </section>
  `,
})
export class ReviewsSection {
  readonly targetType = input.required<ReviewTargetType>();
  readonly targetId = input.required<string>();
  readonly ownerId = input<string | null>(null);
  // The first page (from reviewsResource).
  readonly page = input<ReviewsPage | undefined>(undefined);
  // A review was added or removed: the listing's rating changed.
  readonly changed = output();

  protected readonly auth = inject(AuthStore);
  private readonly http = inject(HttpClient);
  private readonly transloco = inject(TranslocoService);
  private readonly confirm = injectConfirm();
  private readonly lang = injectLang();
  protected readonly maxPhotos = MAX_PHOTOS;

  private readonly more = linkedSignal<ReviewsPage | undefined, Review[]>({
    source: this.page,
    computation: () => [],
  });
  protected readonly reviews = computed(() => [...(this.page()?.items ?? []), ...this.more()]);
  protected readonly summary = computed(() => this.page()?.summary ?? null);
  protected readonly hasMore = computed(() => this.reviews().length < (this.page()?.total ?? 0));
  protected readonly loadingMore = signal(false);
  protected readonly deleting = signal(false);
  protected readonly sendError = signal<string | null>(null);

  protected readonly canWrite = computed(() => {
    const user = this.auth.user();
    if (!user || user.id === this.ownerId()) return false;
    return !this.reviews().some((review) => review.author.id === user.id);
  });

  private readonly model = signal<ReviewForm>({ rating: 0, text: '', photos: [] });
  protected readonly reviewForm = form(this.model, (path) => {
    validateStandardSchema(
      path,
      z.object({
        text: z
          .string()
          .check(z.trim(), z.minLength(10, this.transloco.translate('reviews.textTooShort'))),
      }),
    );
    min(path.rating, 1, { message: this.transloco.translate('reviews.ratingRequired') });
    maxLength(path.text, 3000);
  });

  protected share(star: number): string {
    const stats = this.summary();
    const count = stats?.distribution[String(star)] ?? 0;
    return stats && stats.count > 0 ? `${(count / stats.count) * 100}%` : '0%';
  }

  protected authorName(review: Review): string {
    return (
      review.author.firstName ??
      review.author.username ??
      this.transloco.translate('reviews.anonymous')
    );
  }

  protected date(value: string): string {
    return new Date(value).toLocaleDateString(this.lang(), { year: 'numeric', month: 'long' });
  }

  protected async loadMore(): Promise<void> {
    this.loadingMore.set(true);
    try {
      const next = await firstValueFrom(
        this.http.get<ReviewsPage>('/reviews', {
          params: {
            targetType: this.targetType(),
            targetId: this.targetId(),
            limit: REVIEWS_PAGE_SIZE,
            offset: this.reviews().length,
          },
        }),
      );
      this.more.update((list) => [...list, ...next.items]);
    } finally {
      this.loadingMore.set(false);
    }
  }

  protected async send(event: Event): Promise<void> {
    event.preventDefault();
    this.sendError.set(null);
    await submit(this.reviewForm, async () => {
      const { rating, text, photos } = this.model();
      try {
        await firstValueFrom(
          this.http.post('/reviews', {
            targetType: this.targetType(),
            targetId: this.targetId(),
            rating,
            text: text.trim(),
            photos,
          }),
        );
        this.model.set({ rating: 0, text: '', photos: [] });
        this.reviewForm().reset();
        this.changed.emit();
      } catch (error) {
        this.sendError.set(
          error instanceof HttpErrorResponse && error.status === 409
            ? 'reviews.alreadyReviewed'
            : 'reviews.submitError',
        );
      }
      return undefined;
    });
  }

  protected async remove(review: Review): Promise<void> {
    if (!(await this.confirm({ title: 'reviews.deleteConfirm', danger: true }))) return;
    this.deleting.set(true);
    try {
      await firstValueFrom(this.http.delete(`/reviews/${review.id}`));
      this.changed.emit();
    } finally {
      this.deleting.set(false);
    }
  }
}
