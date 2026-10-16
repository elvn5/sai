import { SeoService, bindPageMeta } from '@/core/seo/seo.service';
import { SiteLink } from '@/ui/site-link';
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

const SECTIONS = [
  { path: '/trails', key: 'trails', icon: 'icon-[heroicons--map]' },
  { path: '/outings', key: 'outings', icon: 'icon-[mdi--hiking]' },
  { path: '/tours', key: 'tours', icon: 'icon-[mdi--compass-outline]' },
  { path: '/hotels', key: 'hotels', icon: 'icon-[heroicons--building-office-2]' },
  { path: '/local-guides', key: 'localGuides', icon: 'icon-[heroicons--user-group]' },
  { path: '/guides', key: 'guides', icon: 'icon-[heroicons--book-open]' },
] as const;

// /hikes: everything for a trip in the mountains, one card per section.
@Component({
  selector: 'app-hikes-page',
  imports: [SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="mx-auto max-w-5xl px-4 py-12">
      <h1 class="text-3xl font-extrabold text-text">{{ 'hikes.title' | transloco }}</h1>
      <p class="mt-2 max-w-2xl text-text-muted">{{ 'hikes.subtitle' | transloco }}</p>

      <ul class="mt-8 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        @for (section of sections; track section.key) {
          <li>
            <a
              [appSiteLink]="section.path"
              class="group flex h-full flex-col gap-3 rounded-card border border-border bg-surface p-5 shadow-sm transition duration-200 ease-out hover:-translate-y-1 hover:border-brand-400 hover:shadow-xl focus-visible:-translate-y-1 focus-visible:shadow-xl motion-reduce:transform-none motion-reduce:transition-none"
            >
              <span class="flex size-12 items-center justify-center rounded-full bg-brand-500/10 transition-colors group-hover:bg-brand-500/20">
                <span [class]="section.icon + ' size-6 text-brand-600 dark:text-brand-300'" aria-hidden="true"></span>
              </span>
              <span class="text-lg font-semibold text-text">{{ 'nav.' + section.key | transloco }}</span>
              <span class="flex-1 text-sm text-text-muted">{{ 'hikes.sections.' + section.key | transloco }}</span>
              <span class="inline-flex items-center gap-1 text-sm font-medium text-brand-700 dark:text-brand-300">
                {{ 'hikes.open' | transloco }}
                <span class="icon-[heroicons--arrow-right] size-4 transition-transform group-hover:translate-x-0.5 motion-reduce:transform-none" aria-hidden="true"></span>
              </span>
            </a>
          </li>
        }
      </ul>
    </section>
  `,
})
export class HikesPage {
  protected readonly sections = SECTIONS;

  constructor() {
    const transloco = inject(TranslocoService);
    const seo = inject(SeoService);
    bindPageMeta(() => {
      const title = transloco.translate('hikes.title');
      return {
        title,
        description: transloco.translate('hikes.subtitle'),
        jsonLd: [seo.breadcrumbs([{ name: title, path: '/hikes' }])],
      };
    });
  }
}
