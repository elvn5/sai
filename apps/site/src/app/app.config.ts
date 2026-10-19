import { routes } from '@/app/app.routes';
import { apiInterceptor } from '@/core/api/api.interceptor';
import { loadClientConfig } from '@/core/config/app-config';
import { FeatureFlags } from '@/core/feature-flags/feature-flags';
import { provideI18n } from '@/core/i18n/i18n';
import { provideImageThumbnails } from '@/core/lib/image-loader';
import { DOCUMENT, isPlatformBrowser } from '@angular/common';
import { provideHttpClient, withFetch, withInterceptors } from '@angular/common/http';
import {
  type ApplicationConfig,
  PLATFORM_ID,
  TransferState,
  inject,
  provideAppInitializer,
  provideBrowserGlobalErrorListeners,
} from '@angular/core';
import {
  provideClientHydration,
  withEventReplay,
  withHttpTransferCacheOptions,
  withIncrementalHydration,
} from '@angular/platform-browser';
import {
  Router,
  type UrlTree,
  provideRouter,
  withComponentInputBinding,
  withInMemoryScrolling,
  withViewTransitions,
} from '@angular/router';

// Whether two URLs differ only in the query string or fragment.
function samePath(from: UrlTree, to: UrlTree | undefined): boolean {
  const path = (url: UrlTree): string => url.toString().split(/[?#]/)[0] ?? '';
  return to !== undefined && path(from) === path(to);
}

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideRouter(
      routes,
      withComponentInputBinding(),
      withInMemoryScrolling({ scrollPositionRestoration: 'enabled', anchorScrolling: 'enabled' }),
      // Pages cross-fade into each other (motion.css); a filter that only
      // changes the query string swaps in place.
      withViewTransitions({
        skipInitialTransition: true,
        onViewTransitionCreated: ({ transition }) => {
          // Switched off in the developer menu (core/feature-flags).
          if (!inject(FeatureFlags).isOn('viewTransitions')) {
            transition.skipTransition();
            return;
          }
          const navigation = inject(Router).currentNavigation();
          const from = navigation?.previousNavigation?.finalUrl;
          if (from && samePath(from, navigation?.finalUrl)) {
            transition.skipTransition();
            return;
          }
          // Back (and forward) through history: the page comes in from the
          // side instead of rising (motion.css, html.vt-back).
          if (navigation?.trigger === 'popstate') {
            const root = inject(DOCUMENT).documentElement;
            root.classList.add('vt-back');
            void transition.finished.finally(() => root.classList.remove('vt-back'));
          }
        },
      }),
    ),
    provideHttpClient(withFetch(), withInterceptors([apiInterceptor])),
    // Data fetched during SSR reaches the browser in the HTML (HTTP transfer
    // cache) instead of being requested again. Browser requests carry the
    // session cookie, which by default opts them out; the server renders
    // without cookies, so what it cached is the anonymous response, served
    // only for the first render. Personal data is fetched again in the browser.
    provideClientHydration(
      withEventReplay(),
      withIncrementalHydration(),
      withHttpTransferCacheOptions({ includeRequestsWithCredentials: true }),
    ),
    provideI18n(),
    provideImageThumbnails(),
    // Client-rendered pages have no server state: load the runtime config.
    provideAppInitializer(() =>
      isPlatformBrowser(inject(PLATFORM_ID)) ? loadClientConfig(inject(TransferState)) : undefined,
    ),
  ],
};
