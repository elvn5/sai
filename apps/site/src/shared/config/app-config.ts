import {
  InjectionToken,
  REQUEST_CONTEXT,
  TransferState,
  inject,
  makeStateKey,
} from '@angular/core';

// Settings that differ between staging and production. The SSR server
// reads them from its environment on every request (src/server.ts) and
// hands them to the browser through TransferState, so one image serves
// every environment — nothing is baked in at build time.
export interface AppConfig {
  // The public API (services/gateway).
  apiBaseUrl: string;
  // This site's canonical origin, no trailing slash: canonical URLs, hreflang.
  siteUrl: string;
  // Bot username (no @) for the Telegram Login Widget.
  telegramBotUsername: string;
  // Only ermekten.com is indexable; staging and previews are noindex.
  indexable: boolean;
}

// What the SSR server passes to Angular as the request context.
export interface RequestContext {
  config: AppConfig;
}

const CONFIG_KEY = makeStateKey<AppConfig>('app-config');

// Local `ng serve` without a server context (unit tests).
const FALLBACK: AppConfig = {
  apiBaseUrl: 'http://localhost:3001',
  siteUrl: 'http://localhost:4000',
  telegramBotUsername: '',
  indexable: false,
};

export const APP_CONFIG = new InjectionToken<AppConfig>('APP_CONFIG', {
  providedIn: 'root',
  factory: (): AppConfig => {
    const state = inject(TransferState);
    const context = inject(REQUEST_CONTEXT, { optional: true }) as RequestContext | null;
    if (context?.config) {
      state.set(CONFIG_KEY, context.config);
      return context.config;
    }
    return state.get(CONFIG_KEY, FALLBACK);
  },
});

// Reads the config from the server's environment (SSR server only).
export function configFromEnv(env: Record<string, string | undefined>): AppConfig {
  const siteUrl = (env['SITE_URL'] ?? FALLBACK.siteUrl).replace(/\/$/, '');
  return {
    apiBaseUrl: env['API_BASE_URL'] ?? FALLBACK.apiBaseUrl,
    siteUrl,
    telegramBotUsername: env['TELEGRAM_BOT_USERNAME'] ?? '',
    indexable: siteUrl === 'https://ermekten.com',
  };
}
