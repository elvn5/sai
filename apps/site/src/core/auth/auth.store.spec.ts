import { PartnerStatus } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { TelegramWebAppStore } from '@/core/telegram/telegram-web-app';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';

const USER = {
  id: 'u1',
  telegramId: null,
  email: 'a@b.kg',
  username: null,
  firstName: 'Aida',
  lastName: null,
  phone: null,
  avatarUrl: null,
  partnerStatus: PartnerStatus.APPROVED,
  partnerKind: 'organizer',
};

function setup(initData?: string): { auth: AuthStore; http: HttpTestingController } {
  TestBed.configureTestingModule({
    providers: [
      provideHttpClient(),
      provideHttpClientTesting(),
      {
        provide: TelegramWebAppStore,
        useValue: { ready: Promise.resolve(), initData: () => initData },
      },
    ],
  });
  return { auth: TestBed.inject(AuthStore), http: TestBed.inject(HttpTestingController) };
}

const tick = (): Promise<void> => new Promise((resolve) => setTimeout(resolve));

describe('AuthStore', () => {
  it('checks the cookie session once per page load', async () => {
    const { auth, http } = setup();
    const first = auth.ensureSession();
    const second = auth.ensureSession();
    await tick();
    http.expectOne('/auth/user/me').flush({ user: USER });

    expect(await first).toBe(true);
    expect(await second).toBe(true);
    expect(auth.isPartner()).toBe(true);
    expect(auth.displayName()).toBe('Aida');
    http.verify();
  });

  it('treats a failed check as anonymous', async () => {
    const { auth, http } = setup();
    const result = auth.ensureSession();
    await tick();
    http.expectOne('/auth/user/me').flush(null, { status: 401, statusText: 'Unauthorized' });
    expect(await result).toBe(false);
    expect(auth.checked()).toBe(true);
    expect(auth.user()).toBeNull();
  });

  it('reports a sign-in made after the first check', async () => {
    const { auth, http } = setup();
    const first = auth.ensureSession();
    await tick();
    http.expectOne('/auth/user/me').flush(null, { status: 401, statusText: 'Unauthorized' });
    expect(await first).toBe(false);

    const login = auth.login({ identifier: 'a@b.kg', password: 'secret-123' });
    http.expectOne('/auth/user/login').flush({ user: USER });
    await login;
    // No second /auth/user/me: the store already knows.
    expect(await auth.ensureSession()).toBe(true);
    http.verify();
  });

  it('signs in with Telegram initData inside the Mini App', async () => {
    const { auth, http } = setup('query_id=1&hash=abc');
    const result = auth.ensureSession();
    await tick();
    const req = http.expectOne('/auth/telegram/miniapp');
    expect(req.request.body).toEqual({ initData: 'query_id=1&hash=abc' });
    req.flush({ user: USER });
    expect(await result).toBe(true);
    http.verify();
  });

  it('forgets the user on logout', async () => {
    const { auth, http } = setup();
    const done = auth.logout();
    http.expectOne('/auth/user/logout').flush({});
    await done;
    expect(auth.user()).toBeNull();
  });
});
