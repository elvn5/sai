import type { ListingPost, Paginated, ReviewTargetType } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { injectLang } from '@/core/i18n/current-lang';
import { loaded } from '@/core/lib/resource';
import { injectConfirm } from '@/ui/confirm-dialog';
import { PhotoGrid } from '@/ui/photo-grid';
import { PhotoUploader } from '@/ui/photo-uploader';
import { HttpClient, httpResource } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  linkedSignal,
  signal,
} from '@angular/core';
import { FormField, form, maxLength, submit } from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

const PAGE_SIZE = 5;
const MAX_PHOTOS = 10;

interface PostForm {
  text: string;
  photos: string[];
}

// News and photo updates from a listing's owner, like a small feed; the
// owner writes and deletes them here.
@Component({
  selector: 'app-posts-feed',
  // A box of its own: deferred blocks observe their root element.
  host: { class: 'block' },
  imports: [FormField, PhotoGrid, PhotoUploader, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (isOwner() || posts().length > 0) {
      <section aria-labelledby="posts-heading">
        <h2 id="posts-heading" class="text-2xl font-bold text-text">{{ 'posts.heading' | transloco }}</h2>

        @if (isOwner()) {
          <form class="ui-card mt-4 flex flex-col gap-3 p-4" novalidate (submit)="send($event)">
            <label class="sr-only" for="post-text">{{ 'posts.composerPlaceholder' | transloco }}</label>
            <textarea id="post-text" class="ui-input" rows="3" [placeholder]="'posts.composerPlaceholder' | transloco" [formField]="postForm.text"></textarea>
            <app-photo-uploader folder="posts" [limit]="maxPhotos" [formField]="postForm.photos" />
            @if (sendError(); as key) {
              <p class="ui-error" role="alert">{{ key | transloco }}</p>
            }
            <button type="submit" class="ui-btn-primary self-start" [disabled]="postForm().submitting()">{{ 'posts.submit' | transloco }}</button>
          </form>
        }

        <ul class="mt-4 flex flex-col gap-4">
          @for (post of posts(); track post.id) {
            <li class="ui-card p-4">
              <p class="text-xs text-text-muted"><time [attr.datetime]="post.createdAt">{{ date(post.createdAt) }}</time></p>
              @if (post.text) {
                <p class="mt-2 whitespace-pre-line text-text">{{ post.text }}</p>
              }
              @if (post.photos.length > 0) {
                <app-photo-grid class="mt-3 block" [photos]="post.photos" />
              }
              @if (isOwner()) {
                <button type="button" class="ui-btn-ghost ui-btn-xs mt-2" [disabled]="deleting()" (click)="remove(post)">{{ 'posts.delete' | transloco }}</button>
              }
            </li>
          }
        </ul>
        @if (isOwner() && posts().length === 0 && first.hasValue()) {
          <p class="mt-4 text-sm text-text-muted">{{ 'posts.emptyOwner' | transloco }}</p>
        }
        @if (hasMore()) {
          <button type="button" class="ui-btn-soft mt-4" [disabled]="loadingMore()" (click)="loadMore()">
            {{ (loadingMore() ? 'gallery.loadingMore' : 'listing.loadMore') | transloco }}
          </button>
        }
      </section>
    }
  `,
})
export class PostsFeed {
  readonly targetType = input.required<ReviewTargetType>();
  readonly targetId = input.required<string>();
  readonly ownerId = input<string | null>(null);

  private readonly auth = inject(AuthStore);
  private readonly http = inject(HttpClient);
  private readonly transloco = inject(TranslocoService);
  private readonly confirm = injectConfirm();
  private readonly lang = injectLang();
  protected readonly maxPhotos = MAX_PHOTOS;

  protected readonly first = httpResource<Paginated<ListingPost>>(() => ({
    url: '/listing-posts',
    params: {
      targetType: this.targetType(),
      targetId: this.targetId(),
      limit: PAGE_SIZE,
      offset: 0,
    },
  }));
  private readonly more = linkedSignal<unknown, ListingPost[]>({
    source: () => loaded(this.first),
    computation: () => [],
  });
  protected readonly posts = computed(() => [...(loaded(this.first)?.items ?? []), ...this.more()]);
  protected readonly hasMore = computed(
    () => this.posts().length < (loaded(this.first)?.total ?? 0),
  );
  protected readonly isOwner = computed(() => {
    const user = this.auth.user();
    return Boolean(user && user.id === this.ownerId());
  });
  protected readonly loadingMore = signal(false);
  protected readonly deleting = signal(false);
  protected readonly sendError = signal<string | null>(null);

  private readonly model = signal<PostForm>({ text: '', photos: [] });
  protected readonly postForm = form(this.model, (path) => {
    maxLength(path.text, 3000);
  });

  protected date(value: string): string {
    return new Date(value).toLocaleDateString(this.lang(), {
      day: 'numeric',
      month: 'long',
      year: 'numeric',
    });
  }

  protected async loadMore(): Promise<void> {
    this.loadingMore.set(true);
    try {
      const next = await firstValueFrom(
        this.http.get<Paginated<ListingPost>>('/listing-posts', {
          params: {
            targetType: this.targetType(),
            targetId: this.targetId(),
            limit: PAGE_SIZE,
            offset: this.posts().length,
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
    const { text, photos } = this.model();
    if (!text.trim() && photos.length === 0) {
      this.sendError.set('posts.emptyError');
      return;
    }
    await submit(this.postForm, async () => {
      try {
        await firstValueFrom(
          this.http.post('/listing-posts', {
            targetType: this.targetType(),
            targetId: this.targetId(),
            text: text.trim(),
            photos,
          }),
        );
        this.model.set({ text: '', photos: [] });
        this.postForm().reset();
        this.first.reload();
      } catch {
        this.sendError.set('posts.submitError');
      }
      return undefined;
    });
  }

  protected async remove(post: ListingPost): Promise<void> {
    if (!(await this.confirm({ title: 'posts.deleteConfirm', danger: true }))) return;
    this.deleting.set(true);
    try {
      await firstValueFrom(this.http.delete(`/listing-posts/${post.id}`));
      this.first.reload();
    } finally {
      this.deleting.set(false);
    }
  }
}
