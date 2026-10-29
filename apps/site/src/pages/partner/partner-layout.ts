import { AuthStore } from '@/core/auth/auth.store';
import { ChatUnread } from '@/core/chat/chats';
import { splitLang } from '@/core/i18n/langs';
import { injectCurrentUrl } from '@/core/lib/current-url';
import { CabinetStore } from '@/pages/partner/cabinet-store';
import { AppLogo } from '@/ui/app-logo';
import { SiteLink } from '@/ui/site-link';
import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';

interface CabinetLink {
  path: string;
  key: string;
  icon: string;
  exact: boolean;
}

const LINKS: CabinetLink[] = [
  { path: '/partner', key: 'dashboard', icon: 'icon-[heroicons--squares-2x2]', exact: true },
  { path: '/partner/stats', key: 'stats', icon: 'icon-[heroicons--chart-bar]', exact: false },
  { path: '/partner/clients', key: 'clients', icon: 'icon-[heroicons--users]', exact: false },
  { path: '/partner/leads', key: 'leads', icon: 'icon-[heroicons--funnel]', exact: false },
  { path: '/partner/tasks', key: 'tasks', icon: 'icon-[heroicons--check-circle]', exact: false },
  {
    path: '/partner/messages',
    key: 'messages',
    icon: 'icon-[heroicons--chat-bubble-left-right]',
    exact: false,
  },
  {
    path: '/partner/broadcasts',
    key: 'broadcasts',
    icon: 'icon-[heroicons--megaphone]',
    exact: false,
  },
  {
    path: '/partner/reports',
    key: 'reports',
    icon: 'icon-[heroicons--presentation-chart-line]',
    exact: false,
  },
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
];

// A driver's cabinet: trips and vehicles instead of tours and listings.
const DRIVER_LINKS: CabinetLink[] = [
  { path: '/partner', key: 'dashboard', icon: 'icon-[heroicons--squares-2x2]', exact: true },
  { path: '/partner/trips', key: 'trips', icon: 'icon-[heroicons--map-pin]', exact: false },
  { path: '/partner/fleet', key: 'fleet', icon: 'icon-[heroicons--truck]', exact: false },
  { path: '/partner/clients', key: 'clients', icon: 'icon-[heroicons--users]', exact: false },
  { path: '/partner/tasks', key: 'tasks', icon: 'icon-[heroicons--check-circle]', exact: false },
  {
    path: '/partner/messages',
    key: 'messages',
    icon: 'icon-[heroicons--chat-bubble-left-right]',
    exact: false,
  },
  {
    path: '/partner/broadcasts',
    key: 'broadcasts',
    icon: 'icon-[heroicons--megaphone]',
    exact: false,
  },
  { path: '/partner/costs', key: 'costs', icon: 'icon-[heroicons--receipt-percent]', exact: false },
  {
    path: '/partner/earnings',
    key: 'earnings',
    icon: 'icon-[heroicons--presentation-chart-line]',
    exact: false,
  },
  {
    path: '/partner/driver-profile',
    key: 'driverProfile',
    icon: 'icon-[heroicons--identification]',
    exact: false,
  },
];

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
            @for (link of links(); track link.key) {
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
                @if (link.key === 'messages' && chats.count() > 0) {
                  <span class="ui-count">{{ chats.count() }}<span class="sr-only"> {{ 'chat.unread' | transloco }}</span></span>
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
  private readonly auth = inject(AuthStore);
  protected readonly links = computed(() => (this.auth.isDriver() ? DRIVER_LINKS : LINKS));
  protected readonly store = inject(CabinetStore);
  protected readonly chats = inject(ChatUnread);
  private readonly url = injectCurrentUrl();
  // The unprefixed path shown.
  private readonly path = computed(() => splitLang(this.url()).path);

  protected isActive(link: CabinetLink): boolean {
    const path = this.path();
    return link.exact ? path === link.path : path === link.path || path.startsWith(`${link.path}/`);
  }
}
