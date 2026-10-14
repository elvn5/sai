import { ConfigStore } from '@/shared/config/app-config';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import type { EnvironmentProviders, Provider } from '@angular/core';
import { provideRouter } from '@angular/router';

// What page specs need: a fake API (HttpTestingController), a router and
// the runtime config.
export function pageTestProviders(): (Provider | EnvironmentProviders)[] {
  return [
    provideHttpClient(),
    provideHttpClientTesting(),
    provideRouter([]),
    { provide: ConfigStore, useValue: { config: { apiBaseUrl: 'http://api.test' } } },
  ];
}
