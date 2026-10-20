import { AuthStore } from '@/core/auth/auth.store';
import { splitLang } from '@/core/i18n/langs';
import { injectCurrentUrl } from '@/core/lib/current-url';
import { PageChrome } from '@/core/lib/page-chrome';
import { Notify } from '@/core/notify/notify';
import { ColorModeStore } from '@/core/theme/color-mode';
import { LanguageSwitcher } from '@/layout/language-switcher';
import { ThemeToggle } from '@/layout/theme-toggle';
import { AppLogo } from '@/ui/app-logo';
import { DestinationSearch } from '@/ui/destination-search';
import { PersonAvatar } from '@/ui/person-avatar';
import { SiteLink } from '@/ui/site-link';
import { CdkMenu, CdkMenuItem, CdkMenuTrigger } from '@angular/cdk/menu';
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

// The sections, as in the sketches: the hiking hub, then each catalog.
// Hotels, local guides, trails and travel guides are paused for now
// (server/paused-sections.ts).
const NAV_LINKS = [
  { path: '/hikes', label: 'nav.hikes', icon: 'icon-[heroicons--fire]' },
  { path: '/outings', label: 'nav.outings', icon: 'icon-[heroicons--user-group]' },
  { path: '/tours', label: 'nav.tours', icon: 'icon-[heroicons--map]' },
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
    CdkMenu,
    CdkMenuItem,
    CdkMenuTrigger,
    DestinationSearch,
    LanguageSwitcher,
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
              <button
                type="button"
                class="ml-1 rounded-full focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-500"
                [cdkMenuTriggerFor]="account"
                [attr.aria-label]="'profile.title' | transloco"
              >
                <app-person-avatar [person]="user" [size]="34" />
              </button>
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
          <nav class="motion-cascade flex flex-col gap-0.5" [attr.aria-label]="'footer.sections.nav' | transloco">
            <a
              appSiteLink="/"
              class="flex items-center gap-3 rounded-[0.625rem] px-2.5 py-3 font-medium"
              [class]="section() === '/' ? 'bg-brand-500/14 font-semibold text-brand-700 dark:text-brand-300' : ''"
              [attr.aria-current]="section() === '/' ? 'page' : null"
            >
              <span [class]="(section() === '/' ? 'icon-[heroicons--home-solid]' : 'icon-[heroicons--home] text-text-muted') + ' size-5'" aria-hidden="true"></span>
              {{ 'nav.feed' | transloco }}
            </a>
            @for (link of links; track link.path) {
              <a
                [appSiteLink]="link.path"
                class="flex items-center gap-3 rounded-[0.625rem] px-2.5 py-3 font-medium"
                [class]="section() === link.path ? 'bg-brand-500/14 font-semibold text-brand-700 dark:text-brand-300' : ''"
                [attr.aria-current]="section() === link.path ? 'page' : null"
              >
                <span [class]="link.icon + ' size-5' + (section() === link.path ? '' : ' text-text-muted')" aria-hidden="true"></span>
                {{ link.label | transloco }}
              </a>
            }
          </nav>
          <div class="mt-1.5 flex items-center justify-between border-t border-border px-1 pt-3">
            <app-language-switcher [showName]="true" />
            <div class="ui-segment w-32" role="radiogroup" [attr.aria-label]="'theme.toggle' | transloco">
              <button type="button" role="radio" [attr.aria-checked]="colorMode.mode() === 'light'" [attr.aria-label]="'theme.light' | transloco" (click)="setMode('light', $event)">
                <span class="icon-[heroicons--sun] size-4 align-[-0.2em]" aria-hidden="true"></span>
              </button>
              <button type="button" role="radio" [attr.aria-checked]="colorMode.mode() === 'dark'" [attr.aria-label]="'theme.dark' | transloco" (click)="setMode('dark', $event)">
                <span class="icon-[heroicons--moon] size-4 align-[-0.2em]" aria-hidden="true"></span>
              </button>
            </div>
          </div>
          <div class="mt-3.5 flex flex-col gap-1">
            @if (auth.user(); as user) {
              <a appSiteLink="/profile" class="flex items-center gap-3 rounded-[0.625rem] px-2.5 py-2.5 font-medium hover:bg-text/5">
                <app-person-avatar [person]="user" [size]="30" />{{ auth.displayName() }}
              </a>
              @if (auth.isPartner()) {
                <a appSiteLink="/partner" class="flex items-center gap-3 rounded-[0.625rem] px-2.5 py-2.5 font-medium hover:bg-text/5">
                  <span class="icon-[heroicons--squares-2x2] size-5 text-text-muted" aria-hidden="true"></span>{{ 'cabinet.title' | transloco }}
                </a>
              }
              <button type="button" class="flex items-center gap-3 rounded-[0.625rem] px-2.5 py-2.5 text-left font-medium hover:bg-text/5" (click)="logout()">
                <span class="icon-[heroicons--arrow-right-start-on-rectangle] size-5 text-text-muted" aria-hidden="true"></span>{{ 'profile.logout' | transloco }}
              </button>
            } @else {
              <a [appSiteLink]="loginLink()" class="ui-btn-primary ui-btn-lg w-full">
                <span class="icon-[heroicons--user-circle] size-[1.15rem]" aria-hidden="true"></span>{{ 'login.cta' | transloco }}
              </a>
            }
          </div>
        </div>
      }
    </header>

    <ng-template #account>
      <div cdkMenu class="motion-drop-in min-w-52 rounded-xl border border-border bg-surface p-1 text-sm text-text shadow-lg">
        @if (auth.user(); as user) {
          <div class="flex items-center gap-2.5 px-3 py-2.5">
            <app-person-avatar [person]="user" [size]="32" />
            <div class="min-w-0">
              <p class="truncate font-semibold">{{ auth.displayName() }}</p>
              @if (user.username) {
                <p class="truncate text-xs text-text-muted">&#64;{{ user.username }}</p>
              }
            </div>
          </div>
          <div class="my-1 h-px bg-border"></div>
        }
        <a cdkMenuItem appSiteLink="/profile" class="flex items-center gap-2.5 rounded-lg px-3 py-2 hover:bg-text/5 focus:bg-text/5 focus:outline-none">
          <span class="icon-[heroicons--user-circle] size-[1.1rem] text-text-muted" aria-hidden="true"></span>{{ 'profile.title' | transloco }}
        </a>
        @if (auth.isPartner()) {
          <a cdkMenuItem appSiteLink="/partner" class="flex items-center gap-2.5 rounded-lg px-3 py-2 hover:bg-text/5 focus:bg-text/5 focus:outline-none">
            <span class="icon-[heroicons--squares-2x2] size-[1.1rem] text-text-muted" aria-hidden="true"></span>{{ 'cabinet.title' | transloco }}
          </a>
        }
        <div class="my-1 h-px bg-border"></div>
        <button cdkMenuItem type="button" class="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-left hover:bg-text/5 focus:bg-text/5 focus:outline-none" (cdkMenuItemTriggered)="logout()">
          <span class="icon-[heroicons--arrow-right-start-on-rectangle] size-[1.1rem] text-text-muted" aria-hidden="true"></span>{{ 'profile.logout' | transloco }}
        </button>
      </div>
    </ng-template>
  `,
})
export class Header {
  protected readonly auth = inject(AuthStore);
  protected readonly colorMode = inject(ColorModeStore);
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

  protected setMode(mode: 'light' | 'dark', event: MouseEvent): void {
    if (this.colorMode.mode() === mode) return;
    const origin = event.detail > 0 ? { x: event.clientX, y: event.clientY } : undefined;
    this.colorMode.toggle(origin);
  }

  protected async logout(): Promise<void> {
    await this.auth.logout();
    this.menuOpen.set(false);
    this.notify.info({ title: 'notify.signedOut' });
  }
}
