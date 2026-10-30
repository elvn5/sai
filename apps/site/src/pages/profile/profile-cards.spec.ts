import { type AppUser, PartnerStatus } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { ParticipantFormCard } from '@/pages/profile/participant-form-card';
import { PasswordCard } from '@/pages/profile/password-card';
import { pageTestProviders } from '@/testing/providers';
import { HttpTestingController } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';

const user: AppUser = {
  id: 'u1',
  telegramId: null,
  email: 'aida@example.kg',
  username: null,
  firstName: 'Аида',
  lastName: null,
  phone: null,
  avatarUrl: null,
  partnerStatus: PartnerStatus.NONE,
  partnerKind: 'organizer',
  hasPassword: true,
};

async function settle(): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve));
  TestBed.tick();
}

function type(el: HTMLElement, selector: string, value: string): void {
  const input = el.querySelector(selector) as HTMLInputElement;
  input.value = value;
  input.dispatchEvent(new Event('input'));
}

function click(el: HTMLElement, text: string): void {
  const found = [...el.querySelectorAll('button')].find((b) => b.textContent?.includes(text));
  if (!found) throw new Error(`no button "${text}"`);
  found.click();
}

describe('ParticipantFormCard', () => {
  it('fills in the form in the profile', async () => {
    TestBed.configureTestingModule({ providers: pageTestProviders() });
    const api = TestBed.inject(HttpTestingController);
    const fixture = TestBed.createComponent(ParticipantFormCard);
    TestBed.tick();
    api.expectOne('/tours/participant-form').flush({ form: null, updatedAt: null });
    await settle();
    const el = fixture.nativeElement as HTMLElement;
    expect(el.textContent).toContain('participantForm.empty');

    click(el, 'participantForm.fill');
    TestBed.tick();
    type(el, 'input[id$="-name"]', 'Мама');
    type(el, 'input[id$="-phone"]', '+996 555 000111');
    (el.querySelector('input[value="hiking"]') as HTMLInputElement).dispatchEvent(
      new Event('change'),
    );
    el.querySelector('form')?.dispatchEvent(new Event('submit'));
    await settle();
    const req = api.expectOne('/tours/participant-form');
    expect(req.request.method).toBe('PUT');
    expect(req.request.body.form.emergencyName).toBe('Мама');
    req.flush({ form: req.request.body.form, updatedAt: '2026-10-31T00:00:00Z' });
    await settle();
    expect(el.textContent).toContain('+996 555 000111');
    expect(el.textContent).toContain('participantForm.levels.hiking.title');
    expect(el.querySelector('form')).toBeNull();
  });
});

describe('PasswordCard', () => {
  function render(account: AppUser): { el: HTMLElement; api: HttpTestingController } {
    TestBed.configureTestingModule({ providers: pageTestProviders() });
    const api = TestBed.inject(HttpTestingController);
    const fixture = TestBed.createComponent(PasswordCard);
    fixture.componentRef.setInput('user', account);
    TestBed.tick();
    return { el: fixture.nativeElement as HTMLElement, api };
  }

  it('changes the password with the current one', async () => {
    const { el, api } = render(user);
    click(el, 'profile.password.change');
    TestBed.tick();
    const inputs = el.querySelectorAll<HTMLInputElement>('input[type=password]');
    expect(inputs.length).toBe(3);
    const fill = (current: string, next: string, confirm: string): void => {
      for (const [i, v] of [current, next, confirm].entries()) {
        inputs[i].value = v;
        inputs[i].dispatchEvent(new Event('input'));
      }
    };

    fill('old-secret', 'new-secret', 'other');
    el.querySelector('form')?.dispatchEvent(new Event('submit'));
    await settle();
    api.expectNone('/auth/user/password/change');

    fill('wrong', 'new-secret', 'new-secret');
    el.querySelector('form')?.dispatchEvent(new Event('submit'));
    await settle();
    const wrong = api.expectOne('/auth/user/password/change');
    expect(wrong.request.body).toEqual({ currentPassword: 'wrong', password: 'new-secret' });
    wrong.flush(
      { message: 'Current password is incorrect' },
      { status: 400, statusText: 'Bad Request' },
    );
    await settle();
    expect(el.textContent).toContain('profile.password.wrongCurrent');

    fill('old-secret', 'new-secret', 'new-secret');
    el.querySelector('form')?.dispatchEvent(new Event('submit'));
    await settle();
    api.expectOne('/auth/user/password/change').flush({ user });
    await settle();
    expect(TestBed.inject(AuthStore).user()?.id).toBe('u1');
    expect(el.querySelector('form')).toBeNull();
  });

  it('sets the first password without a current one, if there is a login', () => {
    const { el } = render({ ...user, hasPassword: false });
    click(el, 'profile.password.set');
    TestBed.tick();
    expect(el.querySelectorAll('input[type=password]').length).toBe(2);

    TestBed.resetTestingModule();
    const telegramOnly = render({ ...user, hasPassword: false, email: null });
    expect(telegramOnly.el.textContent).toContain('profile.password.needContact');
    expect(telegramOnly.el.querySelector('button')).toBeNull();
  });
});
