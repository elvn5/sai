import { splitLang } from '@/core/i18n/langs';
import { injectCurrentUrl } from '@/core/lib/current-url';
import { CabinetStore } from '@/pages/partner/cabinet-store';
import { SiteLink } from '@/ui/site-link';
import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';

const LINKS = [
  { path: '/partner', key: 'dashboard', icon: 'icon-[heroicons--squares-2x2]', exact: true },
  { path: '/partner/stats', key: 'stats', icon: 'icon-[heroicons--chart-bar]', exact: false },
  {
    path: '/partner/bookings',
    key: 'bookings',
    icon: 'icon-[mdi--ticket-confirmation-outline]',
    exact: false,
  },
  { path: '/partner/finances', key: 'finances', icon: 'icon-[heroicons--banknotes]', exact: false },
  {
    path: '/partner/listings',
    key: 'listings',
    icon: 'icon-[heroicons--building-storefront]',
    exact: false,
  },
  {
    path: '/partner/reviews',
    key: 'reviews',
    icon: 'icon-[heroicons--chat-bubble-left-right]',
    exact: false,
  },
] as const;

// The partner cabinet (/partner/**): a section menu (a sidebar on wide
// screens, a scrolling row on phones and in the Telegram Mini App) next to
// the section. Approved partners only; rendered in the browser.
@Component({
  selector: 'app-partner-layout',
  imports: [RouterOutlet, SiteLink, TranslocoPipe],
  providers: [CabinetStore],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mx-auto flex w-full max-w-6xl flex-col gap-6 px-4 py-6 lg:flex-row lg:gap-10 lg:py-10">
      <aside class="lg:w-56 lg:shrink-0">
        <div class="lg:sticky lg:top-20">
          <p class="mb-3 hidden text-xs font-semibold tracking-wide text-text-muted uppercase lg:block">{{ 'cabinet.title' | transloco }}</p>
          <nav class="-mx-4 flex gap-1 overflow-x-auto px-4 pb-1 lg:mx-0 lg:flex-col lg:overflow-visible lg:px-0" [attr.aria-label]="'cabinet.title' | transloco">
            @for (link of links; track link.key) {
              <a
                [appSiteLink]="link.path"
                [attr.aria-current]="isActive(link) ? 'page' : null"
                class="flex shrink-0 items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium whitespace-nowrap transition-colors"
                [class]="isActive(link) ? 'bg-brand-500/10 text-brand-700 dark:text-brand-300' : 'text-text-muted hover:bg-surface hover:text-text'"
              >
                <span [class]="link.icon + ' size-5 shrink-0'" aria-hidden="true"></span>
                <span class="flex-1">{{ 'cabinet.nav.' + link.key | transloco }}</span>
                @if (link.key === 'reviews' && store.unansweredCount() > 0) {
                  <span class="rounded-full bg-brand-600 px-1.5 py-0.5 text-[11px] leading-none font-bold text-white">{{ store.unansweredCount() }}</span>
                }
              </a>
            }
          </nav>
        </div>
      </aside>
      <div class="min-w-0 flex-1"><router-outlet /></div>
    </div>
  `,
})
export class PartnerLayout {
  protected readonly links = LINKS;
  protected readonly store = inject(CabinetStore);
  private readonly url = injectCurrentUrl();
  // The unprefixed path shown.
  private readonly path = computed(() => splitLang(this.url()).path);

  protected isActive(link: (typeof LINKS)[number]): boolean {
    const path = this.path();
    return link.exact ? path === link.path : path === link.path || path.startsWith(`${link.path}/`);
  }
}
