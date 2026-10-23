import { type BlogPost, SeoPage } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import { formatInBishkek } from '@/core/lib/format';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { SeoService, bindPageMeta, pageSeoResource } from '@/core/seo/seo.service';
import { ListSkeleton } from '@/ui/list-skeleton';
import { LoadError } from '@/ui/load-error';
import { PostBadges } from '@/ui/post-badges';
import { SiteLink } from '@/ui/site-link';
import { NgOptimizedImage } from '@angular/common';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

// The blog in the current language: newest posts first (the API's order).
@Component({
  selector: 'app-blog-list-page',
  imports: [ListSkeleton, LoadError, NgOptimizedImage, PostBadges, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="ui-container py-12">
      <div class="mb-8 flex items-center justify-between gap-4">
        <h1 class="text-3xl font-extrabold text-text">{{ 'blog.heading' | transloco }}</h1>
        @if (auth.isPartner()) {
          <a appSiteLink="/blog/new" class="ui-btn-primary">
            <span class="icon-[heroicons--pencil-square] size-4" aria-hidden="true"></span>
            {{ 'blog.form.newTitle' | transloco }}
          </a>
        }
      </div>

      @if (posts.error()) {
        <app-load-error (retry)="posts.reload()" />
      } @else if (posts.value(); as list) {
        @if (list.length > 0) {
          <ul class="motion-reveal grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            @for (post of list; track post.id; let first = $first) {
              <li>
                <a [appSiteLink]="'/blog/' + post.slug" class="group flex h-full flex-col overflow-hidden rounded-card border border-border bg-surface shadow-[0_1px_2px_rgb(15_23_42/0.06)] motion-lift dark:shadow-none">
                  <div class="relative aspect-[16/10] overflow-hidden bg-linear-to-br from-brand-500/20 via-surface to-secondary-500/20">
                    @if (post.coverImageUrl) {
                      <img
                        [ngSrc]="media(post.coverImageUrl)"
                        fill
                        [priority]="first"
                        sizes="(min-width: 1024px) 30vw, (min-width: 640px) 50vw, 100vw"
                        alt=""
                        class="motion-zoom object-cover"
                      />
                    } @else {
                      <span class="icon-[mdi--image-filter-hdr] absolute inset-0 m-auto size-14 text-brand-600/40 dark:text-brand-300/40" aria-hidden="true"></span>
                    }
                    @if (post.publishedAt) {
                      <time class="ui-chip-on-image absolute top-2 left-2" [attr.datetime]="post.publishedAt">
                        <span class="icon-[heroicons--calendar-days] size-3.5" aria-hidden="true"></span>{{ date(post.publishedAt) }}
                      </time>
                    }
                  </div>
                  <div class="flex flex-1 flex-col gap-2 px-4 pt-3.5 pb-4">
                    <h2 class="text-[1.0625rem] leading-snug font-semibold text-text group-hover:text-brand-700 dark:group-hover:text-brand-300">{{ post.title }}</h2>
                    @if (post.excerpt) {
                      <p class="line-clamp-3 text-sm text-text-muted">{{ post.excerpt }}</p>
                    }
                    <div class="mt-auto pt-1.5"><app-post-badges [post]="post" /></div>
                  </div>
                </a>
              </li>
            }
          </ul>
        } @else {
          <p class="text-text-muted">{{ 'blog.empty' | transloco }}</p>
        }
      } @else {
        <app-list-skeleton variant="cards" />
      }
    </section>
  `,
})
export class BlogListPage {
  protected readonly auth = inject(AuthStore);
  private readonly apiBaseUrl = inject(APP_CONFIG).apiBaseUrl;
  private readonly lang = injectLang();
  protected readonly posts = httpResource<BlogPost[]>(() => `/blog?locale=${this.lang()}`);

  constructor() {
    const transloco = inject(TranslocoService);
    const seo = inject(SeoService);
    const overrides = pageSeoResource(SeoPage.BLOG);
    bindPageMeta(() => {
      if (overrides.isLoading()) return undefined;
      const custom = overrides.hasValue() ? overrides.value() : null;
      const heading = transloco.translate('blog.heading');
      return {
        title: custom?.title || heading,
        description: custom?.description || transloco.translate('blog.metaDescription'),
        image: custom?.ogImageUrl,
        jsonLd: [seo.breadcrumbs([{ name: heading, path: '/blog' }])],
      };
    });
  }

  protected date(iso: string): string {
    return formatInBishkek(this.lang(), iso, { day: 'numeric', month: 'long' });
  }

  protected media(url: string): string {
    return resolveMediaUrl(this.apiBaseUrl, url);
  }
}
