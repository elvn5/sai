import { bindPageMeta } from '@/core/seo/seo.service';
import { SiteLink } from '@/ui/site-link';
import { ChangeDetectionStrategy, Component, RESPONSE_INIT, inject } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

// Unknown URLs answer with a real 404 (not a 200 "soft 404"), unindexed.
@Component({
  selector: 'app-not-found-page',
  imports: [SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="mx-auto flex max-w-xl flex-col items-center gap-4 px-4 py-24 text-center">
      <span class="icon-[heroicons--map] size-12 text-text-muted" aria-hidden="true"></span>
      <p class="text-5xl font-extrabold text-brand-600 dark:text-brand-300">404</p>
      <h1 class="text-2xl font-bold text-text">{{ 'notFound.title' | transloco }}</h1>
      <p class="text-text-muted">{{ 'notFound.text' | transloco }}</p>
      <a appSiteLink="/" class="rounded-md bg-brand-600 px-4 py-2 font-semibold text-white hover:bg-brand-700">
        {{ 'notFound.home' | transloco }}
      </a>
    </section>
  `,
})
export class NotFoundPage {
  constructor() {
    const response = inject(RESPONSE_INIT, { optional: true });
    if (response) response.status = 404;
    const transloco = inject(TranslocoService);
    bindPageMeta(() => ({
      title: transloco.translate('notFound.title'),
      description: transloco.translate('notFound.text'),
      alternates: null,
      robots: 'noindex',
    }));
  }
}
