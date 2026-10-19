import { normalizeHash } from '@/core/feature-flags/dev-menu-password';
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
  // The R2 bucket's public URL, no trailing slash ('' when unset): images
  // there get thumbnails from the API (core/lib/image-loader.ts).
  mediaPublicUrl: string;
  // Only ermekten.com is indexable; staging and previews are noindex.
  indexable: boolean;
  // Shown at the foot of every page: APP_VERSION, else package.json's.
  version: string;
  // SHA-256 (hex) of the developer menu's password ('' — no menu).
  devMenuPasswordHash: string;
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
  mediaPublicUrl: '',
  indexable: false,
  version: '',
  devMenuPasswordHash: '',
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

// Where the SSR server publishes the config for pages it doesn't render
// (RenderMode.Client: the browser gets an empty shell, without TransferState).
export const CONFIG_PATH = '/app-config.json';

// Before bootstrap in the browser: when the page came without server-rendered
// state (a Client-mode route), fetch the config from the SSR server. Plain
// fetch: HttpClient's interceptor would send the path to the API.
export async function loadClientConfig(
  state: TransferState,
  fetchJson: (path: string) => Promise<unknown> = async (path) => (await fetch(path)).json(),
): Promise<void> {
  if (state.hasKey(CONFIG_KEY)) return;
  try {
    state.set(CONFIG_KEY, (await fetchJson(CONFIG_PATH)) as AppConfig);
  } catch {
    // Stays on the fallback; the page still renders.
  }
}

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
    mediaPublicUrl: (env['R2_PUBLIC_URL'] ?? '').replace(/\/$/, ''),
    indexable: siteUrl === 'https://ermekten.com',
    version: env['APP_VERSION'] ?? '',
    devMenuPasswordHash: normalizeHash(env['DEV_MENU_PASSWORD_HASH']),
  };
}
