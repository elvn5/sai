import { AuthStore } from '@/core/auth/auth.store';
import { ChatUnread } from '@/core/chat/chats';
import { splitLang } from '@/core/i18n/langs';
import { injectCurrentUrl } from '@/core/lib/current-url';
import { PageChrome } from '@/core/lib/page-chrome';
import { Notify } from '@/core/notify/notify';
import { LanguageSwitcher } from '@/layout/language-switcher';
import { MobileMenu } from '@/layout/mobile-menu';
import { ThemeToggle } from '@/layout/theme-toggle';
import { AppLogo } from '@/ui/app-logo';
import { DestinationSearch } from '@/ui/destination-search';
import { Menu, MenuPanel, MenuTrigger } from '@/ui/menu';
import { PersonAvatar } from '@/ui/person-avatar';
import { SiteLink } from '@/ui/site-link';
import { DOCUMENT } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  afterNextRender,
  computed,
  effect,
  inject,
  signal,
  viewChild,
} from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// The sections: tours, the route catalog and the blog. Hotels,
// local guides and travel guides are paused for now
// (server/paused-sections.ts).
const NAV_LINKS = [
  { path: '/tours', label: 'nav.tours', icon: 'icon-[heroicons--map]' },
  { path: '/trails', label: 'nav.trails', icon: 'icon-[heroicons--map-pin]' },
  { path: '/blog', label: 'nav.blog', icon: 'icon-[heroicons--newspaper]' },
] as const;

// Scrolled further than this, the header slides away while scrolling down
// and comes back on the way up.
const TUCK_AFTER = 240;

// The site header: a light bar (blurred surface) with the sections, search,
// language, mode and the account; a menu panel on phones. Rendered on the
// server for an anonymous visitor; the account part fills in after the
// session check in the browser. On the home page's hero (phones) it's
// see-through until scrolled.
@Component({
  selector: 'app-header',
  // Escape closes the search and the phone menu.
  host: { '(document:keydown.escape)': 'closePanels()' },
  imports: [
    AppLogo,
    DestinationSearch,
    LanguageSwitcher,
    Menu,
    MenuPanel,
    MenuTrigger,
    MobileMenu,
    PersonAvatar,
    SiteLink,
    ThemeToggle,
    TranslocoPipe,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <header
      class="site-header sticky top-0 z-40 transition-[translate,background-color,border-color,box-shadow,color] duration-(--dur-base) ease-standard"
      [class]="overlay() ? 'border-b border-transparent text-white max-lg:bg-transparent lg:border-border lg:bg-surface/82 lg:text-text lg:backdrop-blur-md' : 'border-b border-border bg-surface/82 text-text backdrop-blur-md'"
      [class.-translate-y-full]="tucked()"
      [class.shadow-[0_6px_20px_-12px_rgb(15_23_42/0.25)]]="scrolled() && !overlay()"
    >
      <div class="ui-container flex h-15 items-center gap-5 lg:gap-6">
        <a appSiteLink="/" class="flex shrink-0 items-center gap-2 text-[1.06rem] font-extrabold tracking-tight">
          <span class="flex size-[1.875rem] items-center justify-center rounded-[0.5625rem] bg-brand-600 text-white">
            <app-logo class="size-[1.15rem]" />
          </span>
          {{ 'brand.name' | transloco }}
        </a>

        <nav class="hidden items-center gap-0.5 text-[0.84rem] font-medium xl:flex" [attr.aria-label]="'footer.sections.nav' | transloco">
          @for (link of links; track link.path) {
            <a
              [appSiteLink]="link.path"
              class="relative rounded-lg px-2.5 py-1.5 transition-colors duration-(--dur-fast) after:absolute after:inset-x-2.5 after:-bottom-[0.81rem] after:h-0.5 after:origin-center after:scale-x-0 after:rounded-full after:bg-brand-500 after:transition-transform after:duration-(--dur-base) after:ease-spring aria-[current=page]:text-text aria-[current=page]:after:scale-x-100"
              [class]="section() === link.path ? '' : 'text-text-muted hover:text-text'"
              [attr.aria-current]="section() === link.path ? 'page' : null"
            >
              {{ link.label | transloco }}
            </a>
          }
        </nav>

        <div class="ml-auto flex items-center gap-1.5">
          <button
            type="button"
            class="ui-icon-btn"
            [class]="overlay() ? 'max-lg:text-white max-lg:hover:bg-white/10' : ''"
            [attr.aria-label]="'search.label' | transloco"
            [attr.aria-expanded]="searchOpen()"
            aria-controls="header-search"
            (click)="toggleSearch()"
          >
            <span class="icon-[heroicons--magnifying-glass] size-[1.2rem]" aria-hidden="true"></span>
          </button>
          <div class="hidden items-center gap-1.5 lg:flex">
            <app-language-switcher />
            <app-theme-toggle />
            @if (auth.user(); as user) {
              <a [appSiteLink]="auth.isPartner() ? '/partner/messages' : '/messages'" class="ui-icon-btn relative" [attr.aria-label]="'chat.title' | transloco">
                <span class="icon-[heroicons--chat-bubble-left-right] size-[1.2rem]" aria-hidden="true"></span>
                @if (chats.count() > 0) {
                  <span class="ui-count absolute -top-1 -right-1">{{ chats.count() }}<span class="sr-only"> {{ 'chat.unread' | transloco }}</span></span>
                }
              </a>
              <div appMenu #account="appMenu" class="relative ml-1">
                <button
                  type="button"
                  appMenuTrigger
                  class="flex rounded-full focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-500"
                  [attr.aria-label]="'profile.title' | transloco"
                >
                  <app-person-avatar [person]="user" [size]="34" />
                </button>
                @if (account.open()) {
                  <div appMenuPanel class="motion-drop-in absolute top-full right-0 z-50 mt-2 min-w-52 rounded-xl border border-border bg-surface p-1 text-sm text-text shadow-lg">
                    <div class="flex items-center gap-2.5 px-3 py-2.5">
                      <app-person-avatar [person]="user" [size]="32" />
                      <div class="min-w-0">
                        <p class="truncate font-semibold">{{ auth.displayName() }}</p>
                        @if (user.username) {
                          <p class="truncate text-xs text-text-muted">&#64;{{ user.username }}</p>
                        }
                      </div>
                    </div>
                    <div class="my-1 h-px bg-border" role="separator"></div>
                    <a role="menuitem" appSiteLink="/profile" class="flex items-center gap-2.5 rounded-lg px-3 py-2 hover:bg-text/5 focus:bg-text/5 focus:outline-none">
                      <span class="icon-[heroicons--user-circle] size-[1.1rem] text-text-muted" aria-hidden="true"></span>{{ 'profile.title' | transloco }}
                    </a>
                    <a role="menuitem" [appSiteLink]="auth.isPartner() ? '/partner/messages' : '/messages'" class="flex items-center gap-2.5 rounded-lg px-3 py-2 hover:bg-text/5 focus:bg-text/5 focus:outline-none">
                      <span class="icon-[heroicons--chat-bubble-left-right] size-[1.1rem] text-text-muted" aria-hidden="true"></span><span class="flex-1">{{ 'chat.title' | transloco }}</span>
                      @if (chats.count() > 0) {
                        <span class="ui-count">{{ chats.count() }}</span>
                      }
                    </a>
                    @if (auth.isPartner()) {
                      <a role="menuitem" appSiteLink="/partner" class="flex items-center gap-2.5 rounded-lg px-3 py-2 hover:bg-text/5 focus:bg-text/5 focus:outline-none">
                        <span class="icon-[heroicons--squares-2x2] size-[1.1rem] text-text-muted" aria-hidden="true"></span>{{ 'cabinet.title' | transloco }}
                      </a>
                    }
                    <div class="my-1 h-px bg-border" role="separator"></div>
                    <button role="menuitem" type="button" class="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-left hover:bg-text/5 focus:bg-text/5 focus:outline-none" (click)="logout()">
                      <span class="icon-[heroicons--arrow-right-start-on-rectangle] size-[1.1rem] text-text-muted" aria-hidden="true"></span>{{ 'profile.logout' | transloco }}
                    </button>
                  </div>
                }
              </div>
            } @else {
              <a [appSiteLink]="loginLink()" class="ui-btn-primary ml-1">
                <span class="icon-[heroicons--user-circle] size-4" aria-hidden="true"></span>{{ 'login.cta' | transloco }}
              </a>
            }
          </div>
          <button
            type="button"
            class="ui-icon-btn xl:hidden"
            [class]="overlay() ? 'text-white hover:bg-white/10' : 'text-text'"
            [attr.aria-label]="'nav.menu' | transloco"
            [attr.aria-expanded]="menuOpen()"
            aria-controls="mobile-menu"
            (click)="menuOpen.set(!menuOpen())"
          >
            <span class="icon-swap" aria-hidden="true">
              <span class="icon-[heroicons--bars-3] size-6" [attr.data-off]="menuOpen() ? '' : null"></span>
              <span class="icon-[heroicons--x-mark] size-6" [attr.data-off]="menuOpen() ? null : ''"></span>
            </span>
          </button>
        </div>
      </div>
      @if (!overlay()) {
        <div class="ui-brandline"></div>
      }

      @if (searchOpen()) {
        <div
          id="header-search"
          class="border-t border-border bg-surface text-text"
          animate.enter="motion-drop-in"
          animate.leave="motion-drop-out"
        >
          <div class="ui-container py-3">
            @defer (on immediate) {
              <app-destination-search #search variant="field" class="mx-auto max-w-xl" (chosen)="searchOpen.set(false)" />
            } @placeholder {
              <div class="mx-auto h-[2.4rem] max-w-xl"></div>
            }
          </div>
        </div>
      }

      @if (menuOpen()) {
        <div
          id="mobile-menu"
          class="max-h-[calc(100dvh-3.75rem)] overflow-y-auto border-t border-border bg-surface px-3 py-3 text-text xl:hidden"
          animate.enter="motion-drop-in"
          animate.leave="motion-drop-out"
        >
          <!-- Its own chunk, fetched while the page is idle: the panel opens
               at once, and its code stays out of the first bundle. -->
          @defer (on immediate) {
            <app-mobile-menu [links]="links" [section]="section()" [loginLink]="loginLink()" (done)="menuOpen.set(false)" />
          }
        </div>
      }
    </header>
  `,
})
export class Header {
  protected readonly auth = inject(AuthStore);
  protected readonly chats = inject(ChatUnread);
  private readonly document = inject(DOCUMENT);
  private readonly chrome = inject(PageChrome);
  private readonly notify = inject(Notify);
  protected readonly links = NAV_LINKS;
  protected readonly menuOpen = signal(false);
  protected readonly searchOpen = signal(false);
  protected readonly scrolled = signal(false);
  private readonly scrollingDown = signal(false);
  protected readonly tucked = computed(
    () => this.scrollingDown() && !this.menuOpen() && !this.searchOpen(),
  );
  private readonly url = injectCurrentUrl();
  private readonly search = viewChild<DestinationSearch>('search');

  // The section the page belongs to ("/tours" for /ky/tours/some-tour).
  protected readonly section = computed(() => {
    const path = splitLang(this.url()).path.split(/[?#]/)[0] ?? '/';
    if (path === '/') return '/';
    return NAV_LINKS.find((link) => path === link.path || path.startsWith(`${link.path}/`))?.path;
  });
  // See-through over the home page's hero, until scrolled or opened.
  protected readonly overlay = computed(
    () => this.chrome.overlayHeader() && !this.scrolled() && !this.menuOpen() && !this.searchOpen(),
  );
  // Back to this page after signing in.
  protected readonly loginLink = computed(() => {
    const { path } = splitLang(this.url());
    return path.startsWith('/login') || path === '/'
      ? '/login'
      : `/login?redirect=${encodeURIComponent(this.url())}`;
  });

  constructor() {
    const destroyRef = inject(DestroyRef);
    afterNextRender(() => {
      let last = scrollY;
      let frame = 0;
      const onScroll = (): void => {
        if (frame) return;
        frame = requestAnimationFrame(() => {
          frame = 0;
          const y = scrollY;
          this.scrolled.set(y > 8);
          if (Math.abs(y - last) > 4) this.scrollingDown.set(y > last && y > TUCK_AFTER);
          last = y;
        });
      };
      onScroll();
      addEventListener('scroll', onScroll, { passive: true });
      destroyRef.onDestroy(() => removeEventListener('scroll', onScroll));
    });

    // Things that stick under the header (a tour's tabs) follow it up
    // when it tucks away: --header-offset (styles.css).
    effect(() => {
      this.document.documentElement.toggleAttribute('data-header-tucked', this.tucked());
    });

    // Close the panels after navigating.
    effect(() => {
      this.url();
      this.menuOpen.set(false);
      this.searchOpen.set(false);
    });
    // Into the search field once it opens.
    effect(() => {
      const search = this.search();
      if (search) setTimeout(() => search.focus());
    });
  }

  protected closePanels(): void {
    this.menuOpen.set(false);
    this.searchOpen.set(false);
  }

  protected toggleSearch(): void {
    this.menuOpen.set(false);
    this.searchOpen.set(!this.searchOpen());
  }

  protected async logout(): Promise<void> {
    await this.auth.logout();
    this.menuOpen.set(false);
    this.notify.info({ title: 'notify.signedOut' });
  }
}
