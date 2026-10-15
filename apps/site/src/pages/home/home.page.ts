import { SeoPage } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { bindPageMeta, pageSeoResource } from '@/core/seo/seo.service';
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

// Placeholder home page: the template every public page follows for SEO
// (admin overrides, canonical, hreflang, Open Graph, JSON-LD). The real
// home page arrives in wave 3.
@Component({
  selector: 'app-home-page',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="mx-auto flex max-w-3xl flex-col gap-6 px-4 py-24">
      <p class="flex items-center gap-2 text-sm font-semibold tracking-wide text-brand-600 uppercase dark:text-brand-300">
        <span class="icon-[heroicons--map] size-5" aria-hidden="true"></span>
        {{ 'brand.name' | transloco }}
      </p>
      <h1 class="text-4xl font-extrabold text-text">{{ 'home.tagline' | transloco }}</h1>
    </section>
  `,
})
export class HomePage {
  constructor() {
    const transloco = inject(TranslocoService);
    const overrides = pageSeoResource(SeoPage.HOME);
    injectLang();
    bindPageMeta(() => {
      // Wait for the overrides (or their absence) so the server renders the final <head>.
      if (overrides.isLoading()) return undefined;
      const custom = overrides.hasValue() ? overrides.value() : null;
      return {
        title: custom?.title || undefined,
        description: custom?.description || transloco.translate('home.tagline'),
        image: custom?.ogImageUrl,
      };
    });
  }
}
