import { type BlogPost, SeoPage } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { SeoService, bindPageMeta, pageSeoResource } from '@/core/seo/seo.service';
import { LoadError } from '@/ui/load-error';
import { SiteLink } from '@/ui/site-link';
import { NgOptimizedImage } from '@angular/common';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

// The blog in the current language: newest posts first (the API's order).
@Component({
  selector: 'app-blog-list-page',
  imports: [LoadError, NgOptimizedImage, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="mx-auto max-w-4xl px-4 py-12">
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
          <ul class="flex flex-col gap-4">
            @for (post of list; track post.id; let first = $first) {
              <li class="ui-card overflow-hidden transition-shadow hover:shadow-md">
                <a [appSiteLink]="'/blog/' + post.slug" class="flex flex-col gap-4 p-4 sm:flex-row">
                  @if (post.coverImageUrl) {
                    <div class="relative aspect-video w-full shrink-0 overflow-hidden rounded-lg sm:w-56">
                      <img
                        [ngSrc]="media(post.coverImageUrl)"
                        fill
                        [priority]="first"
                        sizes="(min-width: 640px) 25vw, 100vw"
                        alt=""
                        class="object-cover"
                      />
                    </div>
                  }
                  <div class="flex flex-col justify-center">
                    <h2 class="text-xl font-semibold text-text">{{ post.title }}</h2>
                    @if (post.excerpt) {
                      <p class="mt-2 text-text-muted">{{ post.excerpt }}</p>
                    }
                  </div>
                </a>
              </li>
            }
          </ul>
        } @else {
          <p class="text-text-muted">{{ 'blog.empty' | transloco }}</p>
        }
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

  protected media(url: string): string {
    return resolveMediaUrl(this.apiBaseUrl, url);
  }
}
