import { APP_CONFIG, type AppConfig } from '@/core/config/app-config';
import { TelegramWebAppStore } from '@/core/telegram/telegram-web-app';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { type EnvironmentProviders, type Provider, RESPONSE_INIT } from '@angular/core';
import { provideRouter } from '@angular/router';
import { provideTransloco } from '@jsverse/transloco';

export const TEST_CONFIG: AppConfig = {
  apiBaseUrl: 'https://api.test',
  siteUrl: 'https://ermekten.com',
  telegramBotUsername: '',
  indexable: true,
};

// What page specs need: a fake API (HttpTestingController), a router, the
// runtime config, a response to read the status from, and Transloco
// without translations (templates show the keys).
export function pageTestProviders(
  response: ResponseInit = {},
): (Provider | EnvironmentProviders)[] {
  return [
    provideHttpClient(),
    provideHttpClientTesting(),
    provideRouter([]),
    provideTransloco({
      config: {
        availableLangs: ['ru', 'ky', 'en', 'de'],
        defaultLang: 'ru',
        missingHandler: { logMissingKey: false },
      },
    }),
    { provide: APP_CONFIG, useValue: TEST_CONFIG },
    { provide: RESPONSE_INIT, useValue: response },
    {
      provide: TelegramWebAppStore,
      useValue: { ready: Promise.resolve(), initData: () => undefined, isTma: () => false },
    },
  ];
}
