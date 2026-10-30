import { type AppUser, PartnerStatus } from '@/core/api/types';
import { PartnersPage } from '@/pages/partners/partners.page';
import { pageTestProviders } from '@/testing/providers';
import { HttpTestingController } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';

function user(
  id: string,
  firstName: string,
  status: PartnerStatus,
  kind: AppUser['partnerKind'] = 'organizer',
): AppUser {
  return {
    id,
    telegramId: null,
    email: `${id}@x.kg`,
    username: null,
    firstName,
    lastName: null,
    phone: null,
    createdAt: '2026-10-01T08:00:00.000Z',
    updatedAt: '2026-10-20T08:00:00.000Z',
    partnerStatus: status,
    partnerRequestedAt: '2026-10-10T08:00:00.000Z',
    partnerKind: kind,
  };
}

describe('PartnersPage', () => {
  let api: HttpTestingController;

  async function render(): Promise<HTMLElement> {
    TestBed.configureTestingModule({ imports: [PartnersPage], providers: pageTestProviders() });
    api = TestBed.inject(HttpTestingController);
    const fixture = TestBed.createComponent(PartnersPage);
    TestBed.tick();
    api
      .expectOne('/partners')
      .flush([
        user('a', 'Айбек', PartnerStatus.APPROVED, 'driver'),
        user('b', 'Бермет', PartnerStatus.PENDING),
        user('c', 'Чынара', PartnerStatus.REJECTED),
      ]);
    await fixture.whenStable();
    return fixture.nativeElement as HTMLElement;
  }

  async function settle(): Promise<void> {
    await new Promise((resolve) => setTimeout(resolve));
    TestBed.tick();
  }

  function button(root: HTMLElement, text: string): HTMLButtonElement {
    const found = [...root.querySelectorAll('button')].find((b) =>
      b.textContent?.trim().startsWith(text),
    );
    if (!found) throw new Error(`no button "${text}"`);
    return found;
  }

  function overlay(): HTMLElement {
    return document.querySelector('.cdk-overlay-container') as HTMLElement;
  }

  it('sorts accounts into applications, partners and rejected', async () => {
    const el = await render();
    expect(el.querySelector('tbody')?.textContent).toContain('Бермет');
    expect(el.querySelector('tbody')?.textContent).not.toContain('Айбек');
    button(el, 'Партнёры').click();
    await settle();
    expect(el.querySelector('tbody')?.textContent).toContain('Айбек');
    expect(el.querySelector('tbody')?.textContent).toContain('Водитель');
  });

  it('edits a partner', async () => {
    const el = await render();
    button(el, 'Партнёры').click();
    await settle();
    el.querySelector<HTMLButtonElement>('button[aria-label="Изменить"]')?.click();
    await settle();
    const lastName = overlay().querySelectorAll<HTMLInputElement>(
      'input.ui-input',
    )[1] as HTMLInputElement;
    lastName.value = 'Осмонов';
    lastName.dispatchEvent(new Event('input'));
    overlay()
      .querySelector('form')
      ?.dispatchEvent(new Event('submit', { cancelable: true }));
    await settle();
    const req = api.expectOne('/partners/a');
    expect(req.request.method).toBe('PUT');
    expect(req.request.body).toEqual({
      firstName: 'Айбек',
      lastName: 'Осмонов',
      phone: null,
      kind: 'driver',
      status: 'approved',
    });
  });

  it('takes partnership away after confirming', async () => {
    const el = await render();
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    el.querySelector<HTMLButtonElement>('button[aria-label="Снять статус партнёра"]')?.click();
    const req = api.expectOne('/partners/b');
    expect(req.request.method).toBe('DELETE');
  });

  it('makes an existing account a partner', async () => {
    const el = await render();
    button(el, 'Добавить партнёра').click();
    await settle();
    api
      .expectOne('/users')
      .flush([user('d', 'Данияр', PartnerStatus.NONE), user('a', 'Айбек', PartnerStatus.APPROVED)]);
    await settle();
    expect(overlay().textContent).not.toContain('Айбек');
    button(overlay(), 'Данияр').click();
    const kind = overlay().querySelector<HTMLSelectElement>('select') as HTMLSelectElement;
    kind.value = 'driver';
    kind.dispatchEvent(new Event('change'));
    overlay()
      .querySelector('form')
      ?.dispatchEvent(new Event('submit', { cancelable: true }));
    const req = api.expectOne((r) => r.url === '/partners' && r.method === 'POST');
    expect(req.request.body).toEqual({ userId: 'd', kind: 'driver' });
  });
});
