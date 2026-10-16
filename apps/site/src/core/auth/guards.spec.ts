import { AuthStore } from '@/core/auth/auth.store';
import { requirePartner } from '@/core/auth/guards';
import { Injector, runInInjectionContext } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import {
  type ActivatedRouteSnapshot,
  Router,
  type RouterStateSnapshot,
  UrlTree,
  provideRouter,
} from '@angular/router';

function setup(session: boolean, partner: boolean) {
  TestBed.configureTestingModule({
    providers: [
      provideRouter([]),
      {
        provide: AuthStore,
        useValue: { ensureSession: async () => session, isPartner: () => partner },
      },
    ],
  });
  const route = { data: { lang: 'en' } } as unknown as ActivatedRouteSnapshot;
  const run = () =>
    runInInjectionContext(TestBed.inject(Injector), () =>
      requirePartner(route, { url: '/en/partner/stats' } as RouterStateSnapshot),
    );
  const serialize = (tree: unknown): string => TestBed.inject(Router).serializeUrl(tree as UrlTree);
  return { run, serialize };
}

describe('requirePartner', () => {
  it('lets approved partners in', async () => {
    const { run } = setup(true, true);
    expect(await run()).toBe(true);
  });

  it('sends anonymous visitors to the login page, which brings them back', async () => {
    const { run, serialize } = setup(false, false);
    const result = await run();
    expect(result).toBeInstanceOf(UrlTree);
    expect(serialize(result)).toBe('/en/login?redirect=%2Fen%2Fpartner%2Fstats');
  });

  it('sends other users to the partner tab of their profile', async () => {
    const { run, serialize } = setup(true, false);
    expect(serialize(await run())).toBe('/en/profile?tab=partner');
  });
});
