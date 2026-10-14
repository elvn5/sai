import { PaymentsPage } from '@/pages/payments/payments.page';
import type { AdminOrder, PayeeBalance } from '@/shared/api/types';
import { pageTestProviders } from '@/testing/providers';
import { HttpTestingController } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';

const balance: PayeeBalance = {
  payee: { id: 'p1', firstName: 'Айбек', username: null, avatarUrl: null },
  amount: 21600,
  orders: 2,
};
const order: AdminOrder = {
  id: 'o1',
  kind: 'tour',
  itemTitle: 'Ала-Куль',
  itemDate: '2026-10-24',
  quantity: 2,
  amount: 24000,
  fee: 2400,
  payeeAmount: 21600,
  contactName: 'Айгерим',
  contactPhone: '+996 700 123 456',
  status: 'paid',
  paidAt: '2026-10-14T08:00:00Z',
  refundNote: null,
  payoutId: null,
  createdAt: '2026-10-14T08:00:00Z',
};

describe('PaymentsPage', () => {
  let api: HttpTestingController;

  async function render(): Promise<HTMLElement> {
    TestBed.configureTestingModule({ imports: [PaymentsPage], providers: pageTestProviders() });
    api = TestBed.inject(HttpTestingController);
    const fixture = TestBed.createComponent(PaymentsPage);
    TestBed.tick();
    api.expectOne('/orders/admin').flush([order]);
    api.expectOne('/payouts/balances').flush([balance]);
    api.expectOne('/payouts').flush([]);
    await fixture.whenStable();
    return fixture.nativeElement as HTMLElement;
  }

  function button(el: HTMLElement, text: string): HTMLButtonElement {
    const found = [...el.querySelectorAll('button')].find((b) => b.textContent?.trim() === text);
    if (!found) throw new Error(`no button "${text}"`);
    return found;
  }

  it('shows what is owed and the orders', async () => {
    const el = await render();
    // ru-RU groups thousands with a non-breaking space.
    const text = el.textContent?.replace(/\s/g, ' ') ?? '';
    expect(text).toContain('Айбек');
    expect(text).toContain('21 600 сом');
    expect(text).toContain('комиссия 2 400 сом');
    expect(text).toContain('Выплат ещё не было');
  });

  it('records a payout with the note', async () => {
    const el = await render();
    vi.spyOn(window, 'prompt').mockReturnValue('  Перевод №5 ');
    button(el, 'Отметить выплату').click();
    const req = api.expectOne('/payouts');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ payeeId: 'p1', note: 'Перевод №5' });
  });

  it('explains a refund the API refuses', async () => {
    const el = await render();
    vi.spyOn(window, 'prompt').mockReturnValue('');
    button(el, 'Возврат').click();
    const req = api.expectOne('/orders/o1/refund');
    expect(req.request.body).toEqual({ note: null });
    req.flush({ message: 'conflict' }, { status: 409, statusText: 'Conflict' });
    await new Promise((resolve) => setTimeout(resolve));
    TestBed.tick();
    expect(el.textContent).toContain(
      'Возврат возможен только для оплаченных заказов до выплаты партнёру',
    );
  });
});
