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
  // The API on the private network (http://gateway:8080), for requests made
  // while rendering on the server. Never sent to the browser.
  apiInternalUrl?: string;
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

// The API base URL for requests made on the server: the internal one when
// set (no round trip through the public internet), else the public one.
export const SERVER_API_URL = new InjectionToken<string>('SERVER_API_URL', {
  providedIn: 'root',
  factory: (): string => {
    const context = inject(REQUEST_CONTEXT, { optional: true }) as RequestContext | null;
    return context?.apiInternalUrl || inject(APP_CONFIG).apiBaseUrl;
  },
});

// Reads the request context from the server's environment (SSR server only).
export function contextFromEnv(env: Record<string, string | undefined>): RequestContext {
  const apiInternalUrl = env['API_INTERNAL_URL']?.replace(/\/$/, '');
  return { config: configFromEnv(env), ...(apiInternalUrl ? { apiInternalUrl } : {}) };
}

export function configFromEnv(env: Record<string, string | undefined>): AppConfig {
  const siteUrl = (env['SITE_URL'] ?? FALLBACK.siteUrl).replace(/\/$/, '');
  return {
    apiBaseUrl: (env['API_BASE_URL'] ?? FALLBACK.apiBaseUrl).replace(/\/$/, ''),
    siteUrl,
    telegramBotUsername: env['TELEGRAM_BOT_USERNAME'] ?? '',
    indexable: siteUrl === 'https://ermekten.com',
  };
}
