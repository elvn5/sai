import { Analytics } from '@/core/analytics/analytics';
import { AuthStore } from '@/core/auth/auth.store';
import { injectLang } from '@/core/i18n/current-lang';
import { splitLang } from '@/core/i18n/langs';
import { injectCurrentUrl } from '@/core/lib/current-url';
import { PageChrome } from '@/core/lib/page-chrome';
import { SeoService } from '@/core/seo/seo.service';
import { TelegramWebAppStore } from '@/core/telegram/telegram-web-app';
import { ColorModeStore } from '@/core/theme/color-mode';
import { ColorThemeStore } from '@/core/theme/color-theme';
import { BottomNav } from '@/layout/bottom-nav';
import { CookieConsent } from '@/layout/cookie-consent';
import { Footer } from '@/layout/footer';
import { Header } from '@/layout/header';
import { NetworkBanner } from '@/layout/network-banner';
import { ToastOutlet } from '@/layout/toast-outlet';
import { Location } from '@angular/common';
import { ChangeDetectionStrategy, Component, afterNextRender, effect, inject } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';

// The page frame: header, page, footer. Inside the Telegram Mini App,
// Telegram's own chrome replaces them — no header or footer, and its back
// button on inner pages.
@Component({
  selector: 'app-root',
  imports: [
    BottomNav,
    CookieConsent,
    Footer,
    Header,
    NetworkBanner,
    RouterOutlet,
    ToastOutlet,
    TranslocoPipe,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <a
      href="#main"
      class="sr-only z-50 rounded-md bg-surface px-4 py-2 text-text focus:not-sr-only focus:fixed focus:top-2 focus:left-2"
    >
      {{ 'a11y.skipToContent' | transloco }}
    </a>
    <div class="flex min-h-screen flex-col bg-bg text-text">
      @if (!telegram.isTma()) {
        <app-header [class.max-lg:hidden]="chrome.immersive()" />
        @defer (on idle) {
          <app-network-banner />
        }
      }
      <main id="main" class="min-h-[70vh] flex-1">
        <router-outlet />
      </main>
      @if (!telegram.isTma()) {
        <!-- Rendered on the server; its code loads (and hydrates) only when
             scrolled into view, out of every page's first bundle. -->
        @defer (on viewport; hydrate on viewport) {
          <app-footer [class.max-lg:pb-16]="!chrome.ownBottomBar()" />
        } @placeholder {
          <div class="min-h-80"></div>
        }
        <!-- Fixed overlays: nothing to lay out, so they load once the page is idle. -->
        @if (!chrome.ownBottomBar()) {
          @defer (on idle) {
            <app-bottom-nav />
          }
        }
      }
    </div>
    @defer (on idle) {
      <app-cookie-consent />
    }
    @defer (on idle) {
      <app-toast-outlet />
    }
  `,
})
export class App {
  protected readonly telegram = inject(TelegramWebAppStore);
  protected readonly chrome = inject(PageChrome);

  constructor() {
    const seo = inject(SeoService);
    const lang = injectLang();
    effect(() => seo.setSiteJsonLd(lang()));

    const colorMode = inject(ColorModeStore);
    const colorTheme = inject(ColorThemeStore);
    const auth = inject(AuthStore);
    const analytics = inject(Analytics);
    afterNextRender(() => {
      colorMode.init();
      colorTheme.init();
      void this.telegram.init();
      void auth.ensureSession();
      analytics.init();
    });

    const url = injectCurrentUrl();
    const location = inject(Location);
    effect(() => {
      if (!this.telegram.isTma()) return;
      const atHome = splitLang(url()).path === '/';
      this.telegram.setBackButton(atHome ? null : () => location.back());
    });
  }
}
