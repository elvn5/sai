import { type GalleryItem, SeoPage } from '@/core/api/types';
import { IMAGE_TYPES, UploadApi } from '@/core/api/upload';
import { AuthStore } from '@/core/auth/auth.store';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import { localizedPath } from '@/core/i18n/langs';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { loaded } from '@/core/lib/resource';
import { imageGallery } from '@/core/seo/json-ld';
import { SeoService, bindPageMeta, pageSeoResource } from '@/core/seo/seo.service';
import { injectConfirm } from '@/ui/confirm-dialog';
import { FieldErrors } from '@/ui/field-errors';
import { ListSkeleton } from '@/ui/list-skeleton';
import { LoadError } from '@/ui/load-error';
import { SiteLink } from '@/ui/site-link';
import { Dialog, type DialogRef } from '@angular/cdk/dialog';
import { NgOptimizedImage } from '@angular/common';
import { HttpClient, httpResource } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  type ElementRef,
  Injector,
  type TemplateRef,
  afterNextRender,
  computed,
  effect,
  inject,
  input,
  linkedSignal,
  signal,
  viewChild,
} from '@angular/core';
import { FormField, form, maxLength, submit } from '@angular/forms/signals';
import { Router } from '@angular/router';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

export const GALLERY_PAGE_SIZE = 20;
const SEARCH_DEBOUNCE_MS = 300;

interface Filters {
  location: string | null;
  search: string | null;
}

interface AddForm {
  location: string;
  caption: string;
}

// The /gallery query for a page and filters (also the crawlable links).
export function galleryQuery(page: number, filters: Filters): string {
  const params = new URLSearchParams();
  if (filters.location) params.set('location', filters.location);
  if (filters.search) params.set('search', filters.search);
  if (page > 1) params.set('page', String(page));
  const query = params.toString();
  return query ? `/gallery?${query}` : '/gallery';
}

// Photos from the community. The server renders page N (?page=N) for
// crawlers, with a real link to the next page; in the browser that link and
// scrolling load more in place. Filters live in the URL (?location=,
// ?search=): such pages are not indexed. Partners add photos; authors
// delete their own from the lightbox.
@Component({
  selector: 'app-gallery-page',
  imports: [
    FieldErrors,
    FormField,
    ListSkeleton,
    LoadError,
    NgOptimizedImage,
    SiteLink,
    TranslocoPipe,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="mx-auto max-w-6xl px-4 py-12">
      <div class="mb-6 flex items-center justify-between gap-4">
        <h1 class="text-3xl font-extrabold text-text">{{ 'gallery.heading' | transloco }}</h1>
        @if (auth.isPartner()) {
          <button type="button" class="ui-btn-primary" (click)="openAdd()">
            <span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>
            {{ 'gallery.addButton' | transloco }}
          </button>
        }
      </div>

      <div class="relative mb-6">
        <span class="icon-[heroicons--magnifying-glass] pointer-events-none absolute top-1/2 left-3 size-5 -translate-y-1/2 text-text-muted" aria-hidden="true"></span>
        <input
          type="search"
          class="ui-input pl-10"
          [value]="search() ?? ''"
          [placeholder]="'gallery.searchPlaceholder' | transloco"
          [attr.aria-label]="'gallery.searchPlaceholder' | transloco"
          (input)="onSearch($any($event.target).value)"
        />
      </div>

      @if (locationList(); as list) {
        @if (list.length > 0) {
          <nav class="mb-8 flex flex-wrap gap-2" [attr.aria-label]="'gallery.locationLabel' | transloco">
            <a class="ui-pill" [appSiteLink]="filterLink(null)" [attr.aria-current]="location() ? null : 'page'">
              {{ 'gallery.allLocations' | transloco }}
            </a>
            @for (place of list; track place) {
              <a class="ui-pill" [appSiteLink]="filterLink(place)" [attr.aria-current]="location() === place ? 'page' : null">{{ place }}</a>
            }
          </nav>
        }
      }

      @if (firstPage.error()) {
        <app-load-error (retry)="firstPage.reload()" />
      } @else if (items().length > 0) {
        <ul class="motion-reveal grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
          @for (item of items(); track item.id; let index = $index) {
            <li>
              <button
                type="button"
                class="group relative block aspect-square w-full overflow-hidden rounded-card border border-border"
                [attr.aria-label]="item.caption || item.altText || item.location || ('gallery.heading' | transloco)"
                (click)="openLightbox(index)"
              >
                <img
                  [ngSrc]="media(item.imageUrl)"
                  fill
                  [priority]="index < 2"
                  sizes="(min-width: 1024px) 25vw, (min-width: 640px) 33vw, 50vw"
                  [alt]="item.altText ?? ''"
                  class="motion-zoom object-cover"
                />
                @if (item.location) {
                  <span class="absolute inset-x-0 bottom-0 truncate bg-linear-to-t from-black/70 to-transparent px-2 py-1.5 text-left text-xs text-white">
                    <span class="icon-[heroicons--map-pin] mr-1 size-3" aria-hidden="true"></span>{{ item.location }}
                  </span>
                }
              </button>
            </li>
          }
        </ul>
      } @else if (firstPage.hasValue()) {
        <p class="text-text-muted">{{ 'gallery.empty' | transloco }}</p>
      } @else {
        <app-list-skeleton [count]="8" columns="grid-cols-2 sm:grid-cols-3 lg:grid-cols-4" />
      }

      <nav class="mt-8 flex items-center justify-center gap-3" [attr.aria-label]="'pagination.more' | transloco">
        @if (pageNumber() > 1) {
          <a class="ui-btn-ghost" [appSiteLink]="pageLink(pageNumber() - 1)" rel="prev">
            <span class="icon-[heroicons--chevron-left] size-4" aria-hidden="true"></span>
            {{ 'pagination.previous' | transloco }}
          </a>
        }
        @if (hasMore()) {
          <a #more class="ui-btn-soft" [href]="localized(pageLink(nextPage()))" rel="next" (click)="loadMoreFromLink($event)">
            {{ (loadingMore() ? 'gallery.loadingMore' : 'pagination.more') | transloco }}
          </a>
        }
      </nav>
    </section>

    <ng-template #lightbox>
      @if (current(); as item) {
        <div
          class="flex h-full w-full flex-col bg-black/95 text-white"
          tabindex="-1"
          (keydown.arrowright)="showNext()"
          (keydown.arrowleft)="showPrevious()"
        >
          <div class="flex justify-end p-4">
            <button type="button" class="flex size-10 items-center justify-center rounded-full bg-white/10 hover:bg-white/20" [attr.aria-label]="'gallery.close' | transloco" (click)="closeLightbox()">
              <span class="icon-[heroicons--x-mark] size-6" aria-hidden="true"></span>
            </button>
          </div>
          <div class="relative flex min-h-0 flex-1 items-center justify-center px-14">
            @if (lightboxIndex() > 0) {
              <button type="button" class="absolute left-2 flex size-12 items-center justify-center rounded-full bg-white/10 hover:bg-white/20 sm:left-4" [attr.aria-label]="'gallery.previous' | transloco" (click)="showPrevious()">
                <span class="icon-[heroicons--chevron-left] size-7" aria-hidden="true"></span>
              </button>
            }
            <img [src]="media(item.imageUrl)" [alt]="item.altText ?? ''" class="max-h-[78vh] max-w-full object-contain" />
            @if (lightboxIndex() < items().length - 1 || hasMore()) {
              <button type="button" class="absolute right-2 flex size-12 items-center justify-center rounded-full bg-white/10 hover:bg-white/20 sm:right-4" [attr.aria-label]="'gallery.next' | transloco" (click)="showNext()">
                <span class="icon-[heroicons--chevron-right] size-7" aria-hidden="true"></span>
              </button>
            }
          </div>
          <div class="flex flex-col items-center gap-2 px-4 pt-3 pb-6 text-center">
            @if (item.location) {
              <p class="flex items-center gap-1 text-sm text-white/80">
                <span class="icon-[heroicons--map-pin] size-4" aria-hidden="true"></span>{{ item.location }}
              </p>
            }
            @if (item.caption) {
              <p class="max-w-lg text-sm text-white/90">{{ item.caption }}</p>
            }
            @if (isOwn(item)) {
              <button type="button" class="rounded-md bg-white/10 px-3 py-1.5 text-sm hover:bg-white/20" [disabled]="deleting()" (click)="remove(item)">
                {{ 'gallery.deleteButton' | transloco }}
              </button>
            }
          </div>
        </div>
      }
    </ng-template>

    <ng-template #addDialog>
      <form class="ui-dialog flex flex-col gap-4" novalidate (submit)="addPhoto($event)">
        <h2 class="text-lg font-semibold text-text">{{ 'gallery.addTitle' | transloco }}</h2>
        <div class="ui-field">
          <label for="gallery-file">{{ 'gallery.uploadLabel' | transloco }}</label>
          <input id="gallery-file" type="file" accept="image/jpeg,image/png,image/webp" class="ui-file" (change)="onFile($event)" />
        </div>
        <label class="ui-field">
          {{ 'gallery.locationLabel' | transloco }}
          <input class="ui-input" [formField]="addForm.location" />
          <span class="ui-hint">{{ 'gallery.locationHint' | transloco }}</span>
          <app-field-errors [state]="addForm.location()" />
        </label>
        <label class="ui-field">
          {{ 'gallery.captionLabel' | transloco }}
          <input class="ui-input" [formField]="addForm.caption" />
          <app-field-errors [state]="addForm.caption()" />
        </label>
        @if (addError(); as key) {
          <p class="ui-error" role="alert">{{ key | transloco }}</p>
        }
        <div class="flex justify-end gap-2">
          <button type="button" class="ui-btn-ghost" (click)="addRef?.close()">{{ 'gallery.close' | transloco }}</button>
          <button type="submit" class="ui-btn-primary" [disabled]="!file() || addForm().submitting()">
            {{ (addForm().submitting() ? 'upload.uploading' : 'gallery.submit') | transloco }}
          </button>
        </div>
      </form>
    </ng-template>
  `,
})
export class GalleryPage {
  readonly page = input<string | undefined>(undefined);
  readonly location = input<string | undefined>(undefined);
  readonly search = input<string | undefined>(undefined);

  protected readonly auth = inject(AuthStore);
  private readonly http = inject(HttpClient);
  private readonly uploads = inject(UploadApi);
  private readonly dialog = inject(Dialog);
  private readonly router = inject(Router);
  private readonly transloco = inject(TranslocoService);
  private readonly confirm = injectConfirm();
  private readonly apiBaseUrl = inject(APP_CONFIG).apiBaseUrl;
  private readonly injector = inject(Injector);
  private readonly lang = injectLang();

  protected readonly pageNumber = computed(() =>
    Math.max(1, Number.parseInt(this.page() ?? '1', 10) || 1),
  );
  private readonly filters = computed<Filters>(() => ({
    location: this.location() || null,
    search: this.search() || null,
  }));

  protected readonly locations = httpResource<string[]>(() => '/gallery/locations');
  protected readonly locationList = computed(() => loaded(this.locations));
  protected readonly firstPage = httpResource<GalleryItem[]>(() => {
    const { location, search } = this.filters();
    return {
      url: '/gallery',
      params: {
        limit: GALLERY_PAGE_SIZE,
        offset: (this.pageNumber() - 1) * GALLERY_PAGE_SIZE,
        ...(location ? { location } : {}),
        ...(search ? { search } : {}),
      },
    };
  });

  // Pages loaded in place after the first one; reset with the query.
  private readonly more = linkedSignal<unknown, GalleryItem[]>({
    source: () => [this.pageNumber(), this.filters()],
    computation: () => [],
  });
  private readonly exhausted = linkedSignal<unknown, boolean>({
    source: () => [this.pageNumber(), this.filters()],
    computation: () => false,
  });
  protected readonly loadingMore = signal(false);
  protected readonly items = computed(() => [...(loaded(this.firstPage) ?? []), ...this.more()]);
  protected readonly hasMore = computed(
    () => !this.exhausted() && (loaded(this.firstPage)?.length ?? 0) === GALLERY_PAGE_SIZE,
  );
  protected readonly nextPage = computed(
    () => this.pageNumber() + 1 + Math.floor(this.more().length / GALLERY_PAGE_SIZE),
  );

  private readonly moreLink = viewChild<ElementRef<HTMLElement>>('more');
  private readonly lightboxTemplate = viewChild.required<TemplateRef<unknown>>('lightbox');
  private readonly addTemplate = viewChild.required<TemplateRef<unknown>>('addDialog');
  private lightboxRef: DialogRef | null = null;
  protected addRef: DialogRef | null = null;
  protected readonly lightboxIndex = signal(0);
  protected readonly current = computed(() => this.items()[this.lightboxIndex()] ?? null);
  protected readonly deleting = signal(false);

  protected readonly file = signal<File | null>(null);
  protected readonly addError = signal<string | null>(null);
  private readonly addModel = signal<AddForm>({ location: '', caption: '' });
  protected readonly addForm = form(this.addModel, (path) => {
    maxLength(path.location, 255);
    maxLength(path.caption, 500);
  });

  private searchTimer: ReturnType<typeof setTimeout> | undefined;

  constructor() {
    const seo = inject(SeoService);
    const overrides = pageSeoResource(SeoPage.GALLERY);
    bindPageMeta(() => {
      if (overrides.isLoading() || this.firstPage.isLoading()) return undefined;
      const custom = overrides.hasValue() ? overrides.value() : null;
      const heading = this.transloco.translate('gallery.heading');
      const filtered = Boolean(this.filters().location || this.filters().search);
      const page = this.pageNumber();
      const title = custom?.title || heading;
      return {
        title:
          page > 1 ? `${title} — ${this.transloco.translate('pagination.page', { page })}` : title,
        description: custom?.description || this.transloco.translate('gallery.metaDescription'),
        image: custom?.ogImageUrl ?? loaded(this.firstPage)?.[0]?.imageUrl,
        query: page > 1 && !filtered ? { page } : undefined,
        robots: filtered ? 'noindex, follow' : undefined,
        jsonLd: [
          imageGallery(
            title,
            seo.pageUrl('/gallery'),
            (loaded(this.firstPage) ?? []).map((item) => ({
              url: this.media(item.imageUrl),
              caption: item.caption,
              location: item.location,
            })),
          ),
          seo.breadcrumbs([{ name: heading, path: '/gallery' }]),
        ],
      };
    });

    // Scrolling near the end loads the next page (browser only).
    const destroyRef = inject(DestroyRef);
    afterNextRender(() => {
      const observer = new IntersectionObserver(
        (entries) => {
          if (entries.some((entry) => entry.isIntersecting)) void this.loadMore();
        },
        { rootMargin: '400px' },
      );
      effect(
        (onCleanup) => {
          const element = this.moreLink()?.nativeElement;
          if (!element) return;
          observer.observe(element);
          onCleanup(() => observer.unobserve(element));
        },
        { injector: this.injector },
      );
      destroyRef.onDestroy(() => observer.disconnect());
    });
    destroyRef.onDestroy(() => clearTimeout(this.searchTimer));
  }

  protected media(url: string): string {
    return resolveMediaUrl(this.apiBaseUrl, url);
  }

  protected filterLink(place: string | null): string {
    return galleryQuery(1, { location: place, search: this.filters().search });
  }

  protected localized(path: string): string {
    return localizedPath(this.lang(), path);
  }

  protected pageLink(page: number): string {
    return galleryQuery(page, this.filters());
  }

  protected onSearch(value: string): void {
    clearTimeout(this.searchTimer);
    this.searchTimer = setTimeout(() => {
      const search = value.trim() || null;
      void this.router.navigate([], {
        queryParams: { search, page: null },
        queryParamsHandling: 'merge',
        replaceUrl: true,
      });
    }, SEARCH_DEBOUNCE_MS);
  }

  // The "show more" link is a real link for crawlers; people stay here.
  protected loadMoreFromLink(event: MouseEvent): void {
    if (event.button !== 0 || event.ctrlKey || event.metaKey || event.shiftKey) return;
    event.preventDefault();
    void this.loadMore();
  }

  protected async loadMore(): Promise<void> {
    if (this.loadingMore() || !this.hasMore()) return;
    this.loadingMore.set(true);
    const { location, search } = this.filters();
    try {
      const batch = await firstValueFrom(
        this.http.get<GalleryItem[]>('/gallery', {
          params: {
            limit: GALLERY_PAGE_SIZE,
            offset: (this.pageNumber() - 1) * GALLERY_PAGE_SIZE + this.items().length,
            ...(location ? { location } : {}),
            ...(search ? { search } : {}),
          },
        }),
      );
      this.more.update((list) => [...list, ...batch]);
      if (batch.length < GALLERY_PAGE_SIZE) this.exhausted.set(true);
    } finally {
      this.loadingMore.set(false);
    }
  }

  protected openLightbox(index: number): void {
    this.lightboxIndex.set(index);
    this.lightboxRef = this.dialog.open(this.lightboxTemplate(), {
      ariaLabel: this.transloco.translate('gallery.heading'),
      width: '100vw',
      height: '100vh',
      maxWidth: '100vw',
      panelClass: 'gallery-lightbox',
    });
    this.lightboxRef.closed.subscribe(() => {
      this.lightboxRef = null;
    });
  }

  protected closeLightbox(): void {
    this.lightboxRef?.close();
  }

  protected async showNext(): Promise<void> {
    const next = this.lightboxIndex() + 1;
    if (next >= this.items().length) await this.loadMore();
    if (next < this.items().length) this.lightboxIndex.set(next);
  }

  protected showPrevious(): void {
    this.lightboxIndex.update((index) => Math.max(0, index - 1));
  }

  protected isOwn(item: GalleryItem): boolean {
    const user = this.auth.user();
    return Boolean(user && item.userId === user.id);
  }

  protected async remove(item: GalleryItem): Promise<void> {
    if (!(await this.confirm({ title: 'gallery.deleteConfirm', danger: true }))) return;
    this.deleting.set(true);
    try {
      await firstValueFrom(this.http.delete(`/gallery/${item.id}`));
      this.closeLightbox();
      this.refresh();
    } finally {
      this.deleting.set(false);
    }
  }

  protected openAdd(): void {
    this.file.set(null);
    this.addError.set(null);
    this.addModel.set({ location: '', caption: '' });
    this.addForm().reset();
    this.addRef = this.dialog.open(this.addTemplate(), {
      ariaLabel: this.transloco.translate('gallery.addTitle'),
    });
    this.addRef.closed.subscribe(() => {
      this.addRef = null;
    });
  }

  protected onFile(event: Event): void {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0] ?? null;
    if (file && !IMAGE_TYPES.has(file.type)) {
      this.addError.set('gallery.unsupportedFileType');
      this.file.set(null);
      input.value = '';
      return;
    }
    this.addError.set(null);
    this.file.set(file);
  }

  protected async addPhoto(event: Event): Promise<void> {
    event.preventDefault();
    const file = this.file();
    if (!file) return;
    await submit(this.addForm, async () => {
      const { location, caption } = this.addModel();
      try {
        const imageUrl = await this.uploads.uploadToStorage(file, 'gallery');
        await firstValueFrom(
          this.http.post<GalleryItem>('/gallery', {
            imageUrl,
            location: location.trim() || null,
            caption: caption.trim() || null,
          }),
        );
        this.addRef?.close();
        this.refresh();
      } catch {
        this.addError.set('gallery.uploadError');
      }
      return undefined;
    });
  }

  private refresh(): void {
    this.more.set([]);
    this.exhausted.set(false);
    this.firstPage.reload();
    this.locations.reload();
  }
}
