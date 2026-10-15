import { AuthStore } from '@/core/auth/auth.store';
import { requirePartner } from '@/core/auth/guards';
import { DOCUMENT, Injector, runInInjectionContext } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import {
  type ActivatedRouteSnapshot,
  type RouterStateSnapshot,
  provideRouter,
} from '@angular/router';

function setup(session: boolean, partner: boolean) {
  const assign = vi.fn();
  TestBed.configureTestingModule({
    providers: [
      provideRouter([]),
      {
        provide: AuthStore,
        useValue: { ensureSession: async () => session, isPartner: () => partner },
      },
      { provide: DOCUMENT, useValue: { ...document, location: { assign } } },
    ],
  });
  const route = { data: { lang: 'en' } } as unknown as ActivatedRouteSnapshot;
  const run = () =>
    runInInjectionContext(TestBed.inject(Injector), () =>
      requirePartner(route, {} as RouterStateSnapshot),
    );
  return { run, assign };
}

describe('requirePartner', () => {
  it('lets approved partners in', async () => {
    const { run, assign } = setup(true, true);
    expect(await run()).toBe(true);
    expect(assign).not.toHaveBeenCalled();
  });

  it('sends anonymous visitors to the login page, still on the Nuxt site', async () => {
    const { run, assign } = setup(false, false);
    expect(await run()).toBe(false);
    expect(assign).toHaveBeenCalledWith('/en/login');
  });

  it('sends other users to the partner tab of their profile', async () => {
    const { run, assign } = setup(true, false);
    expect(await run()).toBe(false);
    expect(assign).toHaveBeenCalledWith('/en/profile?tab=partner');
  });
});
