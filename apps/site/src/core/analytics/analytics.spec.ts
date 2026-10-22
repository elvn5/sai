import { ANALYTICS_LOADER, Analytics } from '@/core/analytics/analytics';
import { CONSENT_KEY, CookieConsentStore } from '@/core/analytics/consent';
import type { AnalyticsBackend } from '@/core/analytics/firebase';
import { APP_CONFIG, type AppConfig } from '@/core/config/app-config';
import { Component, PLATFORM_ID } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';
import { TranslocoService } from '@jsverse/transloco';

@Component({ template: '' })
class Blank {}

const FIREBASE = { apiKey: 'key', projectId: 'p', appId: '1:2:web:3', measurementId: 'G-TEST' };

interface Logged {
  name: string;
  params: Record<string, unknown>;
}

function setup(options: { firebase?: boolean; platform?: 'browser' | 'server' } = {}) {
  const logged: Logged[] = [];
  const enabled: boolean[] = [];
  const backend: AnalyticsBackend = {
    log: (name, params) => logged.push({ name, params }),
    setEnabled: (on) => enabled.push(on),
  };
  const loader = vi.fn(async () => backend);
  TestBed.configureTestingModule({
    providers: [
      provideRouter([{ path: '**', component: Blank }]),
      { provide: PLATFORM_ID, useValue: options.platform ?? 'browser' },
      {
        provide: APP_CONFIG,
        useValue: { firebase: options.firebase === false ? null : FIREBASE } as AppConfig,
      },
      { provide: TranslocoService, useValue: { getActiveLang: () => 'ky' } },
      { provide: ANALYTICS_LOADER, useValue: loader },
    ],
  });
  return {
    analytics: TestBed.inject(Analytics),
    consent: TestBed.inject(CookieConsentStore),
    router: TestBed.inject(Router),
    loader,
    logged,
    enabled,
  };
}

// Lets effects, the loader and whenStable settle.
async function settle(): Promise<void> {
  TestBed.tick();
  for (let i = 0; i < 5; i++) await new Promise((resolve) => setTimeout(resolve));
}

describe('Analytics', () => {
  beforeEach(() => localStorage.clear());

  it('sends nothing before the visitor allows analytics', async () => {
    const { analytics, loader, logged } = setup();
    analytics.init();
    analytics.track('view_tour', { tour_id: 't1' });
    await settle();
    expect(loader).not.toHaveBeenCalled();
    expect(logged).toEqual([]);
  });

  it('loads Firebase on consent, then reports pages and events with the language', async () => {
    const { analytics, consent, router, loader, logged } = setup();
    await router.navigateByUrl('/en/tours/son-kol');
    analytics.init();
    consent.set('all');
    await settle();
    expect(loader).toHaveBeenCalledWith(FIREBASE);
    expect(logged[0]).toMatchObject({
      name: 'page_view',
      params: { content_path: '/tours/son-kol', lang: 'ky' },
    });

    await router.navigateByUrl('/outings/o1');
    await settle();
    analytics.track('join_outing', { outing_id: 'o1' });
    await settle();
    expect(logged.slice(1)).toEqual([
      expect.objectContaining({
        name: 'page_view',
        params: expect.objectContaining({ content_path: '/outings/o1' }),
      }),
      { name: 'join_outing', params: { outing_id: 'o1', lang: 'ky' } },
    ]);
    expect(loader).toHaveBeenCalledTimes(1);
  });

  it('starts at once for a visitor who allowed it before', async () => {
    localStorage.setItem(CONSENT_KEY, 'all');
    const { analytics, logged } = setup();
    analytics.track('login', { method: 'password' });
    await settle();
    expect(logged).toEqual([{ name: 'login', params: { method: 'password', lang: 'ky' } }]);
  });

  it('stops and drops its cookies when the visitor changes their mind', async () => {
    localStorage.setItem(CONSENT_KEY, 'all');
    const { analytics, consent, enabled, logged } = setup();
    analytics.init();
    await settle();
    document.cookie = '_ga=GA1.1.123; path=/';
    consent.set('necessary');
    await settle();
    analytics.track('view_outing', { outing_id: 'o1' });
    await settle();
    expect(enabled).toEqual([false]);
    expect(document.cookie).not.toContain('_ga=');
    expect(logged.map((e) => e.name)).toEqual(['page_view']);
  });

  it('is off without the Firebase config, and on the server', async () => {
    localStorage.setItem(CONSENT_KEY, 'all');
    for (const options of [{ firebase: false }, { platform: 'server' as const }]) {
      TestBed.resetTestingModule();
      const { analytics, loader } = setup(options);
      analytics.init();
      analytics.track('sign_up', { method: 'email' });
      await settle();
      expect(loader).not.toHaveBeenCalled();
    }
  });
});

describe('CookieConsentStore', () => {
  beforeEach(() => localStorage.clear());

  it("doesn't take the old notice's answer as consent to analytics", () => {
    localStorage.setItem(CONSENT_KEY, 'accepted');
    const consent = TestBed.inject(CookieConsentStore);
    consent.init();
    expect(consent.answered()).toBe(true);
    expect(consent.choice()).toBeNull();
    expect(consent.analyticsAllowed()).toBe(false);
  });

  it('keeps the answer and asks again on request', () => {
    const consent = TestBed.inject(CookieConsentStore);
    consent.init();
    consent.set('all');
    expect(localStorage.getItem(CONSENT_KEY)).toBe('all');
    consent.reopen();
    expect(consent.choice()).toBeNull();
  });
});
