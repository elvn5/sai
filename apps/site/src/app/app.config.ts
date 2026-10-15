import { routes } from '@/app/app.routes';
import { apiInterceptor } from '@/core/api/api.interceptor';
import { loadClientConfig } from '@/core/config/app-config';
import { provideI18n } from '@/core/i18n/i18n';
import { isPlatformBrowser } from '@angular/common';
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
import { provideRouter, withComponentInputBinding, withInMemoryScrolling } from '@angular/router';

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideRouter(
      routes,
      withComponentInputBinding(),
      withInMemoryScrolling({ scrollPositionRestoration: 'enabled', anchorScrolling: 'enabled' }),
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
    // Client-rendered pages have no server state: load the runtime config.
    provideAppInitializer(() =>
      isPlatformBrowser(inject(PLATFORM_ID)) ? loadClientConfig(inject(TransferState)) : undefined,
    ),
  ],
};
