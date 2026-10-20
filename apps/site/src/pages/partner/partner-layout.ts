import { splitLang } from '@/core/i18n/langs';
import { injectCurrentUrl } from '@/core/lib/current-url';
import { CabinetStore } from '@/pages/partner/cabinet-store';
import { AppLogo } from '@/ui/app-logo';
import { SiteLink } from '@/ui/site-link';
import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';

const LINKS = [
  { path: '/partner', key: 'dashboard', icon: 'icon-[heroicons--squares-2x2]', exact: true },
  { path: '/partner/stats', key: 'stats', icon: 'icon-[heroicons--chart-bar]', exact: false },
  {
    path: '/partner/listings',
    key: 'listings',
    icon: 'icon-[heroicons--rectangle-stack]',
    exact: false,
  },
  {
    path: '/partner/reviews',
    key: 'reviews',
    icon: 'icon-[heroicons--chat-bubble-left-ellipsis]',
    exact: false,
  },
] as const;

// The partner cabinet (/partner/**): a section menu (a sidebar on wide
// screens, a scrolling row on phones and in the Telegram Mini App) next to
// the section. Approved partners only; rendered in the browser.
@Component({
  selector: 'app-partner-layout',
  imports: [AppLogo, RouterOutlet, SiteLink, TranslocoPipe],
  providers: [CabinetStore],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="ui-container flex flex-col gap-5 py-6 lg:flex-row lg:gap-7 lg:py-8">
      <aside class="lg:w-56 lg:shrink-0">
        <div class="lg:sticky lg:top-20 lg:rounded-card lg:border lg:border-border lg:bg-surface lg:p-3">
          <p class="mb-4 hidden items-center gap-2 px-1 pt-1 text-[1.06rem] font-extrabold tracking-tight text-text lg:flex">
            <span class="flex size-[1.875rem] items-center justify-center rounded-[0.5625rem] bg-brand-600 text-white"><app-logo class="size-[1.15rem]" /></span>
            {{ 'cabinet.short' | transloco }}
          </p>
          <nav class="-mx-4 flex gap-1 overflow-x-auto px-4 pb-1 lg:mx-0 lg:flex-col lg:overflow-visible lg:px-0" [attr.aria-label]="'cabinet.title' | transloco">
            @for (link of links; track link.key) {
              <a
                [appSiteLink]="link.path"
                [attr.aria-current]="isActive(link) ? 'page' : null"
                class="flex shrink-0 items-center gap-2 rounded-[0.5625rem] px-2.5 py-2 text-sm font-medium whitespace-nowrap transition-colors duration-(--dur-fast)"
                [class]="isActive(link) ? 'bg-brand-500/10 text-brand-700 dark:text-brand-300' : 'text-text-muted hover:bg-text/5 hover:text-text max-lg:border max-lg:border-border'"
              >
                <span [class]="link.icon + ' size-5 shrink-0'" aria-hidden="true"></span>
                <span class="flex-1">{{ 'cabinet.nav.' + link.key | transloco }}</span>
                @if (link.key === 'reviews' && store.unansweredCount() > 0) {
                  <span class="ui-count">{{ store.unansweredCount() }}</span>
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
