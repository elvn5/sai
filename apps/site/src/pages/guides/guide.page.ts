import type { Guide } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import { fileSize } from '@/core/lib/format';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { loaded } from '@/core/lib/resource';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { article } from '@/core/seo/json-ld';
import { SeoService, bindPageMeta } from '@/core/seo/seo.service';
import { equipmentIcon, equipmentLabelKey } from '@/pages/guides/guide-options';
import { NotFoundPage } from '@/pages/not-found/not-found.page';
import { injectConfirm } from '@/ui/confirm-dialog';
import { LoadError } from '@/ui/load-error';
import { SiteLink } from '@/ui/site-link';
import { NgOptimizedImage } from '@angular/common';
import { HttpClient, HttpErrorResponse, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, input, signal } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';

// One guide (written per language: no hreflang alternates). Its author can
// delete it.
@Component({
  selector: 'app-guide-page',
  imports: [LoadError, NgOptimizedImage, NotFoundPage, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (notFound()) {
      <app-not-found-page />
    } @else if (guide.error()) {
      <div class="mx-auto max-w-3xl px-4 py-12"><app-load-error (retry)="guide.reload()" /></div>
    } @else if (guide.value(); as item) {
      <article class="mx-auto max-w-3xl px-4 py-12">
        <a appSiteLink="/guides" class="text-sm text-brand-700 hover:underline dark:text-brand-300">
          &larr; {{ 'guides.heading' | transloco }}
        </a>
        <div class="mt-4 flex flex-wrap gap-2">
          <span class="ui-badge bg-brand-500/10 text-brand-700 dark:text-brand-300">
            {{ 'guides.difficulty.' + item.difficulty | transloco }}
          </span>
          <span class="ui-badge bg-secondary-500/15 text-secondary-800 dark:text-secondary-200">
            {{ 'guides.season.' + item.season | transloco }}
          </span>
        </div>
        <h1 class="mt-4 text-3xl font-extrabold text-text sm:text-4xl">{{ item.title }}</h1>
        @if (item.coverImageUrl) {
          <div class="relative mt-6 aspect-video overflow-hidden rounded-lg">
            <img [ngSrc]="media(item.coverImageUrl)" fill priority sizes="(min-width: 768px) 50vw, 100vw" alt="" class="object-cover" />
          </div>
        }

        @if (item.equipmentChecklist.length > 0) {
          <section class="ui-card mt-8 p-4">
            <h2 class="mb-3 text-sm font-semibold tracking-wide text-text-muted uppercase">{{ 'guides.equipment' | transloco }}</h2>
            <ul class="flex flex-wrap gap-3">
              @for (entry of item.equipmentChecklist; track entry) {
                <li class="flex items-center gap-2 rounded-full border border-border bg-bg px-3 py-1.5 text-sm text-text">
                  @if (icon(entry); as iconClass) {
                    <span [class]="iconClass + ' size-4 text-brand-600 dark:text-brand-300'" aria-hidden="true"></span>
                  } @else {
                    <span class="size-1.5 shrink-0 rounded-full bg-brand-600 dark:bg-brand-300" aria-hidden="true"></span>
                  }
                  @if (labelKey(entry); as key) {
                    {{ key | transloco }}
                  } @else {
                    {{ entry }}
                  }
                </li>
              }
            </ul>
          </section>
        }

        <div class="prose prose-neutral dark:prose-invert mt-8 max-w-none" [innerHTML]="item.content"></div>

        @if (item.attachments.length > 0) {
          <section class="ui-card mt-8 p-4">
            <h2 class="mb-3 text-sm font-semibold tracking-wide text-text-muted uppercase">{{ 'guides.filesLabel' | transloco }}</h2>
            <ul class="flex flex-col gap-2">
              @for (file of item.attachments; track file.url) {
                <li>
                  <a
                    [href]="media(file.url)"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="flex items-center gap-2 text-sm text-brand-700 hover:underline dark:text-brand-300"
                  >
                    <span class="icon-[heroicons--document-arrow-down] size-4 shrink-0" aria-hidden="true"></span>
                    {{ file.name }}
                    <span class="text-text-muted">({{ size(file.size) }})</span>
                  </a>
                </li>
              }
            </ul>
          </section>
        }

        @if (isAuthor()) {
          <button type="button" class="ui-btn-soft mt-8" [disabled]="deleting()" (click)="remove(item)">
            {{ 'guides.deleteButton' | transloco }}
          </button>
        }
      </article>
    }
  `,
})
export class GuidePage {
  readonly slug = input.required<string>();
  private readonly config = inject(APP_CONFIG);
  private readonly auth = inject(AuthStore);
  private readonly http = inject(HttpClient);
  private readonly transloco = inject(TranslocoService);
  private readonly confirm = injectConfirm();
  private readonly navigate = injectSiteNavigate();
  private readonly lang = injectLang();
  protected readonly guide = httpResource<Guide>(
    () => `/guides/${this.lang()}/${encodeURIComponent(this.slug())}`,
  );
  protected readonly notFound = computed(() => {
    const error = this.guide.error();
    return error instanceof HttpErrorResponse && error.status === 404;
  });
  protected readonly isAuthor = computed(() => {
    const user = this.auth.user();
    return Boolean(user && loaded(this.guide)?.userId === user.id);
  });
  protected readonly deleting = signal(false);
  protected readonly icon = equipmentIcon;
  protected readonly labelKey = equipmentLabelKey;
  protected readonly size = fileSize;

  constructor() {
    const seo = inject(SeoService);
    bindPageMeta(() => {
      const item = this.guide.hasValue() ? this.guide.value() : undefined;
      if (!item) return undefined;
      const path = `/guides/${item.slug}`;
      const description: string =
        item.seoDescription || item.excerpt || this.transloco.translate('guides.metaDescription');
      const image = item.coverImageUrl ? this.media(item.coverImageUrl) : null;
      return {
        title: item.seoTitle || item.title,
        description,
        image,
        type: 'article',
        alternates: null,
        jsonLd: [
          article({
            type: 'Article',
            url: seo.pageUrl(path),
            headline: item.title,
            description,
            image,
            datePublished: item.publishedAt,
            dateModified: item.updatedAt,
            inLanguage: seo.langTag(),
            byOrganization: item.userId === null,
            siteUrl: this.config.siteUrl,
          }),
          seo.breadcrumbs([
            { name: this.transloco.translate('guides.heading'), path: '/guides' },
            { name: item.title, path },
          ]),
        ],
      };
    });
  }

  protected media(url: string): string {
    return resolveMediaUrl(this.config.apiBaseUrl, url);
  }

  protected async remove(item: Guide): Promise<void> {
    if (!(await this.confirm({ title: 'guides.deleteConfirm', danger: true }))) return;
    this.deleting.set(true);
    try {
      await firstValueFrom(this.http.delete(`/guides/${item.id}`));
      await this.navigate('/guides');
    } finally {
      this.deleting.set(false);
    }
  }
}
