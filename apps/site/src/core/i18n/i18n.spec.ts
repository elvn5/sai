import { activateLang, localizedRoutes, provideI18n } from '@/core/i18n/i18n';
import { DOCUMENT, Injector, runInInjectionContext } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import type { ActivatedRouteSnapshot, RouterStateSnapshot } from '@angular/router';
import { TranslocoService } from '@jsverse/transloco';
import { type Observable, firstValueFrom } from 'rxjs';

describe('i18n', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({ providers: [provideI18n()] });
  });

  it('serves pages under every language prefix, Russian last and unprefixed', () => {
    const routes = localizedRoutes([{ path: 'tours' }]);
    expect(routes.map((route) => [route.path, route.data?.['lang']])).toEqual([
      ['ky', 'ky'],
      ['en', 'en'],
      ['de', 'de'],
      ['', 'ru'],
    ]);
  });

  it('loads the translations with their {placeholders} and sets <html lang>', async () => {
    const injector = TestBed.inject(Injector);
    const route = { data: { lang: 'en' } } as unknown as ActivatedRouteSnapshot;
    const result = runInInjectionContext(injector, () =>
      activateLang(route, {} as RouterStateSnapshot),
    ) as Observable<boolean>;
    expect(await firstValueFrom(result)).toBe(true);

    const transloco = TestBed.inject(TranslocoService);
    expect(TestBed.inject(DOCUMENT).documentElement.lang).toBe('en');
    expect(transloco.translate('nav.tours')).toBe('Tours');
    expect(transloco.translate('reviews.count', { count: 3 })).toContain('3');
  });
});
