import { type Guide, SeoPage } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { APP_CONFIG } from '@/core/config/app-config';
import { injectLang } from '@/core/i18n/current-lang';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { SeoService, bindPageMeta, pageSeoResource } from '@/core/seo/seo.service';
import { ListSkeleton } from '@/ui/list-skeleton';
import { LoadError } from '@/ui/load-error';
import { SiteLink } from '@/ui/site-link';
import { NgOptimizedImage } from '@angular/common';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

// Guides in the current language.
@Component({
  selector: 'app-guides-list-page',
  imports: [ListSkeleton, LoadError, NgOptimizedImage, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="mx-auto max-w-4xl px-4 py-12">
      <div class="mb-8 flex items-center justify-between gap-4">
        <h1 class="text-3xl font-extrabold text-text">{{ 'guides.heading' | transloco }}</h1>
        @if (auth.isPartner()) {
          <a appSiteLink="/guides/new" class="ui-btn-primary">
            <span class="icon-[heroicons--plus] size-4" aria-hidden="true"></span>
            {{ 'guides.addButton' | transloco }}
          </a>
        }
      </div>

      @if (guides.error()) {
        <app-load-error (retry)="guides.reload()" />
      } @else if (guides.value(); as list) {
        @if (list.length > 0) {
          <ul class="motion-reveal flex flex-col gap-4">
            @for (guide of list; track guide.id; let first = $first) {
              <li class="ui-card overflow-hidden motion-lift">
                <a [appSiteLink]="'/guides/' + guide.slug" class="flex flex-col gap-4 p-4 sm:flex-row">
                  @if (guide.coverImageUrl) {
                    <div class="relative aspect-video w-full shrink-0 overflow-hidden rounded-lg sm:w-56">
                      <img
                        [ngSrc]="media(guide.coverImageUrl)"
                        fill
                        [priority]="first"
                        sizes="(min-width: 640px) 25vw, 100vw"
                        alt=""
                        class="object-cover"
                      />
                    </div>
                  }
                  <div class="flex flex-1 flex-col justify-center">
                    <div class="mb-2 flex flex-wrap gap-2">
                      <span class="ui-badge bg-brand-500/10 text-brand-700 dark:text-brand-300">
                        {{ 'guides.difficulty.' + guide.difficulty | transloco }}
                      </span>
                      <span class="ui-badge bg-secondary-500/15 text-secondary-800 dark:text-secondary-200">
                        {{ 'guides.season.' + guide.season | transloco }}
                      </span>
                    </div>
                    <h2 class="text-xl font-semibold text-text">{{ guide.title }}</h2>
                    @if (guide.excerpt) {
                      <p class="mt-2 text-text-muted">{{ guide.excerpt }}</p>
                    }
                  </div>
                </a>
              </li>
            }
          </ul>
        } @else {
          <p class="text-text-muted">{{ 'guides.empty' | transloco }}</p>
        }
      } @else {
        <app-list-skeleton variant="rows" />
      }
    </section>
  `,
})
export class GuidesListPage {
  protected readonly auth = inject(AuthStore);
  private readonly apiBaseUrl = inject(APP_CONFIG).apiBaseUrl;
  private readonly lang = injectLang();
  protected readonly guides = httpResource<Guide[]>(() => `/guides?locale=${this.lang()}`);

  constructor() {
    const transloco = inject(TranslocoService);
    const seo = inject(SeoService);
    const overrides = pageSeoResource(SeoPage.GUIDES);
    bindPageMeta(() => {
      if (overrides.isLoading()) return undefined;
      const custom = overrides.hasValue() ? overrides.value() : null;
      const heading = transloco.translate('guides.heading');
      return {
        title: custom?.title || heading,
        description: custom?.description || transloco.translate('guides.metaDescription'),
        image: custom?.ogImageUrl,
        jsonLd: [seo.breadcrumbs([{ name: heading, path: '/guides' }])],
      };
    });
  }

  protected media(url: string): string {
    return resolveMediaUrl(this.apiBaseUrl, url);
  }
}
