import { splitLang } from '@/core/i18n/langs';
import { injectCurrentUrl } from '@/core/lib/current-url';
import { bindPageMeta } from '@/core/seo/seo.service';
import { MountainScene } from '@/ui/mountain-scene';
import { SiteLink } from '@/ui/site-link';
import { ChangeDetectionStrategy, Component, RESPONSE_INIT, computed, inject } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

const PARENTS = [
  { path: '/tours', label: 'nav.tours' },
  { path: '/blog', label: 'nav.blog' },
] as const;

// Unknown URLs answer with a real 404 (not a 200 "soft 404"), unindexed.
@Component({
  selector: 'app-not-found-page',
  imports: [MountainScene, SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="relative isolate flex min-h-[70vh] items-center justify-center overflow-hidden px-4 py-20 text-center">
      <div class="absolute inset-x-0 bottom-0 -z-10 h-56 opacity-50 [mask-image:linear-gradient(to_top,#000,transparent)]">
        <app-mountain-scene palette="peaks" [lake]="false" />
      </div>
      <div>
        <p class="motion-enter text-7xl leading-none font-extrabold tracking-tighter text-brand-600 sm:text-8xl dark:text-brand-300">404</p>
        <h1 class="motion-enter motion-enter-1 mt-2 text-xl font-bold text-text">{{ 'notFound.title' | transloco }}</h1>
        <p class="motion-enter motion-enter-2 mt-1.5 text-text-muted">{{ 'notFound.text' | transloco }}</p>
        <div class="motion-enter motion-enter-3 mt-5 flex flex-wrap justify-center gap-2">
          <a appSiteLink="/" class="ui-btn-primary ui-btn-lg"><span class="icon-[heroicons--home] size-[1.1rem]" aria-hidden="true"></span>{{ 'notFound.home' | transloco }}</a>
          @if (section(); as parent) {
            <a [appSiteLink]="parent.path" class="ui-btn-soft ui-btn-lg"><span class="icon-[heroicons--arrow-uturn-left] size-[1.1rem]" aria-hidden="true"></span>{{ parent.label | transloco }}</a>
          }
        </div>
      </div>
    </section>
  `,
})
export class NotFoundPage {
  private readonly url = injectCurrentUrl();
  // The section the missing page would have been in (/tours/… → "All tours").
  protected readonly section = computed(() => {
    const path = splitLang(this.url()).path;
    return PARENTS.find((parent) => path.startsWith(`${parent.path}/`)) ?? null;
  });

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
