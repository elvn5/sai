import { type BlogPost } from '@/core/api/types';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import { type Lang, isLang } from '@/core/i18n/langs';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { loaded } from '@/core/lib/resource';
import { article } from '@/core/seo/json-ld';
import { SeoService, bindPageMeta } from '@/core/seo/seo.service';
import { NotFoundPage } from '@/pages/not-found/not-found.page';
import { LoadError } from '@/ui/load-error';
import { PostBadges } from '@/ui/post-badges';
import { SiteLink } from '@/ui/site-link';
import { NgOptimizedImage } from '@angular/common';
import { HttpErrorResponse, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

// The post's path in each language it is published in; null (no hreflang)
// when it exists in its own language only.
export function postAlternates(
  path: string,
  translations: string[] | undefined,
): Partial<Record<Lang, string>> | null {
  const langs = (translations ?? []).filter(isLang);
  return langs.length > 1 ? Object.fromEntries(langs.map((lang) => [lang, path])) : null;
}

// One blog post. Posts are written per language; those under the same slug
// are one article's translations, linked to each other with hreflang.
@Component({
  selector: 'app-blog-post-page',
  imports: [LoadError, NgOptimizedImage, NotFoundPage, PostBadges, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (notFound()) {
      <app-not-found-page />
    } @else if (post.error()) {
      <div class="mx-auto max-w-3xl px-4 py-12"><app-load-error (retry)="post.reload()" /></div>
    } @else if (post.value(); as item) {
      <article class="mx-auto max-w-3xl px-4 py-12">
        <a appSiteLink="/blog" class="text-sm text-brand-700 hover:underline dark:text-brand-300">
          &larr; {{ 'blog.heading' | transloco }}
        </a>
        <h1 class="mt-4 text-3xl font-extrabold text-text sm:text-4xl">{{ item.title }}</h1>
        @if (item.publishedAt) {
          <time class="mt-2 block text-sm text-text-muted" [attr.datetime]="item.publishedAt">
            {{ published() }}
          </time>
        }
        <div class="group mt-4"><app-post-badges [post]="item" /></div>
        @if (item.coverImageUrl) {
          <div class="relative mt-6 aspect-video overflow-hidden rounded-lg">
            <img [ngSrc]="media(item.coverImageUrl)" fill priority sizes="(min-width: 768px) 50vw, 100vw" [alt]="item.title" class="object-cover" />
          </div>
        }
        <div class="prose prose-neutral dark:prose-invert mt-8 max-w-none" [innerHTML]="item.content"></div>
      </article>
    }
  `,
})
export class BlogPostPage {
  readonly slug = input.required<string>();
  private readonly apiBaseUrl = inject(APP_CONFIG).apiBaseUrl;
  private readonly config = inject(APP_CONFIG);
  private readonly lang = injectLang();
  protected readonly post = httpResource<BlogPost>(
    () => `/blog/${this.lang()}/${encodeURIComponent(this.slug())}`,
  );
  protected readonly notFound = computed(() => {
    const error = this.post.error();
    return error instanceof HttpErrorResponse && error.status === 404;
  });
  protected readonly published = computed(() => {
    const date = loaded(this.post)?.publishedAt;
    return date ? new Date(date).toLocaleDateString(this.lang(), { dateStyle: 'long' }) : '';
  });

  constructor() {
    const transloco = inject(TranslocoService);
    const seo = inject(SeoService);
    bindPageMeta(() => {
      const item = this.post.hasValue() ? this.post.value() : undefined;
      if (!item) return undefined;
      const path = `/blog/${item.slug}`;
      const description: string =
        item.seoDescription || item.excerpt || transloco.translate('blog.metaDescription');
      const image = item.coverImageUrl ? this.media(item.coverImageUrl) : null;
      return {
        title: item.seoTitle || item.title,
        description,
        image,
        type: 'article',
        alternates: postAlternates(path, item.translations),
        jsonLd: [
          article({
            type: 'BlogPosting',
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
            { name: transloco.translate('blog.heading'), path: '/blog' },
            { name: item.title, path },
          ]),
        ],
      };
    });
  }

  protected media(url: string): string {
    return resolveMediaUrl(this.apiBaseUrl, url);
  }
}
