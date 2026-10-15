import { AuthStore } from '@/core/auth/auth.store';
import { injectCurrentUrl } from '@/core/lib/current-url';
import { ColorThemeToggle } from '@/layout/color-theme-toggle';
import { LanguageSwitcher } from '@/layout/language-switcher';
import { ThemeToggle } from '@/layout/theme-toggle';
import { AppLogo } from '@/ui/app-logo';
import { SiteLink } from '@/ui/site-link';
import { NgOptimizedImage } from '@angular/common';
import { ChangeDetectionStrategy, Component, effect, inject, signal } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

const NAV_LINKS = [
  { path: '/', label: 'nav.feed' },
  { path: '/trails', label: 'nav.trails' },
  { path: '/outings', label: 'nav.outings' },
  { path: '/hotels', label: 'nav.hotels' },
  { path: '/tours', label: 'nav.tours' },
  { path: '/local-guides', label: 'nav.localGuides' },
  { path: '/guides', label: 'nav.guides' },
  { path: '/gallery', label: 'nav.gallery' },
  { path: '/blog', label: 'nav.blog' },
  { path: '/hall-of-fame', label: 'nav.hallOfFame' },
] as const;

// The site header. Rendered on the server for an anonymous visitor (the
// HTML is cached and shared); the account part fills in after the session
// check in the browser.
@Component({
  selector: 'app-header',
  imports: [
    AppLogo,
    ColorThemeToggle,
    LanguageSwitcher,
    NgOptimizedImage,
    SiteLink,
    ThemeToggle,
    TranslocoPipe,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <header
      class="animated-gradient-border sticky top-0 z-40 bg-linear-to-r from-brand-900 via-brand-700 to-brand-500 text-white"
    >
      <div class="mx-auto flex h-16 max-w-6xl items-center justify-between gap-4 px-4">
        <a appSiteLink="/" class="flex items-center gap-2 text-lg font-bold">
          <app-logo class="size-7" />
          {{ 'brand.name' | transloco }}
        </a>

        <nav class="hidden items-center gap-5 text-sm font-medium text-white/80 lg:flex" [attr.aria-label]="'footer.sections.nav' | transloco">
          @for (link of links; track link.path) {
            <a [appSiteLink]="link.path" class="transition-colors hover:text-white">{{ link.label | transloco }}</a>
          }
        </nav>

        <div class="hidden items-center gap-1 lg:flex">
          <app-language-switcher />
          <app-color-theme-toggle />
          <app-theme-toggle />
          @if (auth.user(); as user) {
            @if (auth.isPartner()) {
              <a
                appSiteLink="/partner"
                class="flex size-9 items-center justify-center rounded-md hover:bg-white/10"
                [attr.aria-label]="'cabinet.title' | transloco"
                [title]="'cabinet.title' | transloco"
              >
                <span class="icon-[heroicons--squares-2x2] size-5" aria-hidden="true"></span>
              </a>
            }
            <a appSiteLink="/profile" class="ml-1 flex items-center gap-2">
              @if (user.avatarUrl) {
                <img
                  [ngSrc]="user.avatarUrl"
                  width="32"
                  height="32"
                  alt=""
                  class="size-8 rounded-full object-cover ring-2 ring-white/30"
                />
              }
              <span class="hidden text-sm xl:inline">{{ auth.displayName() }}</span>
            </a>
            <button type="button" class="rounded-md px-3 py-2 text-sm hover:bg-white/10" (click)="logout()">
              {{ 'profile.logout' | transloco }}
            </button>
          } @else {
            <a
              appSiteLink="/login"
              class="ml-1 rounded-md bg-white px-3 py-2 text-sm font-semibold text-brand-700 hover:bg-white/90"
            >
              {{ 'login.cta' | transloco }}
            </a>
          }
        </div>

        <button
          type="button"
          class="flex size-10 items-center justify-center rounded-md transition-colors hover:bg-white/10 lg:hidden"
          [attr.aria-label]="'nav.menu' | transloco"
          [attr.aria-expanded]="menuOpen()"
          aria-controls="mobile-menu"
          (click)="menuOpen.set(!menuOpen())"
        >
          @if (menuOpen()) {
            <span class="icon-[heroicons--x-mark] size-6" aria-hidden="true"></span>
          } @else {
            <span class="icon-[heroicons--bars-3] size-6" aria-hidden="true"></span>
          }
        </button>
      </div>

      @if (menuOpen()) {
        <div id="mobile-menu" class="border-t border-white/15 bg-brand-900/95 px-4 py-4 lg:hidden">
          <nav class="flex flex-col gap-1" [attr.aria-label]="'footer.sections.nav' | transloco">
            @for (link of links; track link.path) {
              <a
                [appSiteLink]="link.path"
                class="rounded-md px-3 py-2.5 text-base font-medium text-white/85 hover:bg-white/10 hover:text-white"
              >
                {{ link.label | transloco }}
              </a>
            }
          </nav>
          <div class="mt-3 flex items-center gap-1 border-t border-white/15 pt-3">
            <app-language-switcher />
            <app-color-theme-toggle />
            <app-theme-toggle />
          </div>
          <div class="mt-3 flex flex-col gap-1 border-t border-white/15 pt-3">
            @if (auth.user(); as user) {
              <a appSiteLink="/profile" class="flex items-center gap-2 rounded-md px-3 py-2.5 hover:bg-white/10">
                @if (user.avatarUrl) {
                  <img
                    [ngSrc]="user.avatarUrl"
                    width="32"
                    height="32"
                    alt=""
                    class="size-8 rounded-full object-cover ring-2 ring-white/30"
                  />
                }
                <span class="text-sm font-medium">{{ auth.displayName() }}</span>
              </a>
              @if (auth.isPartner()) {
                <a
                  appSiteLink="/partner"
                  class="flex items-center gap-2 rounded-md px-3 py-2.5 text-sm font-medium hover:bg-white/10"
                >
                  <span class="icon-[heroicons--squares-2x2] size-5" aria-hidden="true"></span>
                  {{ 'cabinet.title' | transloco }}
                </a>
              }
              <button type="button" class="rounded-md px-3 py-2.5 text-left text-sm hover:bg-white/10" (click)="logout()">
                {{ 'profile.logout' | transloco }}
              </button>
            } @else {
              <a
                appSiteLink="/login"
                class="rounded-md bg-white px-3 py-2.5 text-center text-sm font-semibold text-brand-700 hover:bg-white/90"
              >
                {{ 'login.cta' | transloco }}
              </a>
            }
          </div>
        </div>
      }
    </header>
  `,
})
export class Header {
  protected readonly auth = inject(AuthStore);
  protected readonly links = NAV_LINKS;
  protected readonly menuOpen = signal(false);
  private readonly url = injectCurrentUrl();

  constructor() {
    // Close the mobile menu after navigating.
    effect(() => {
      this.url();
      this.menuOpen.set(false);
    });
  }

  protected async logout(): Promise<void> {
    await this.auth.logout();
    this.menuOpen.set(false);
  }
}
