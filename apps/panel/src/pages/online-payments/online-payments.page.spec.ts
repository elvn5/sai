import type { AdminOrder, PaymentGateway, Transaction } from '@/core/api/types';
import { OnlinePaymentPage } from '@/pages/online-payments/online-payment.page';
import { OnlinePaymentsPage } from '@/pages/online-payments/online-payments.page';
import { pageTestProviders } from '@/testing/providers';
import { HttpTestingController } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';

const gateway: PaymentGateway = { provider: 'gopay', online: true, canCancel: true };
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
  status: 'pending',
  paidAt: null,
  refundNote: null,
  payoutId: null,
  createdAt: '2026-10-14T08:00:00Z',
};
function transaction(
  patch: Partial<Transaction> = {},
  orderPatch: Partial<AdminOrder> = {},
): Transaction {
  return {
    orderId: 'o1',
    provider: 'gopay',
    paymentId: 'pay1',
    providerOrderId: 'o1compact',
    status: 'CREATED',
    amount: '24000.00',
    testing: true,
    checkoutUrl: 'https://pay.gopay.kg/p/pay1/',
    gatewayCreatedAt: '2026-10-14T08:00:00Z',
    expiresAt: '2026-10-14T08:30:00Z',
    committedAt: null,
    bankOpDate: null,
    lastError: null,
    checkedAt: '2026-10-14T08:00:00Z',
    createdAt: '2026-10-14T08:00:00Z',
    updatedAt: '2026-10-14T08:00:00Z',
    order: { ...order, ...orderPatch },
    events: [],
    ...patch,
  };
}

async function settle(): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve));
  TestBed.tick();
}

function button(el: HTMLElement, text: string): HTMLButtonElement {
  const found = [...el.querySelectorAll('button')].find((b) => b.textContent?.trim() === text);
  if (!found) throw new Error(`no button "${text}"`);
  return found;
}

describe('OnlinePaymentsPage', () => {
  it('lists GoPay payments and filters them', async () => {
    TestBed.configureTestingModule({
      imports: [OnlinePaymentsPage],
      providers: pageTestProviders(),
    });
    const api = TestBed.inject(HttpTestingController);
    const fixture = TestBed.createComponent(OnlinePaymentsPage);
    TestBed.tick();
    api.expectOne('/payments/admin/gateway').flush(gateway);
    api
      .expectOne('/payments/admin/transactions')
      .flush([transaction({ status: 'COMMITTED' }, { status: 'paid' })]);
    await fixture.whenStable();
    const el = fixture.nativeElement as HTMLElement;
    const text = el.textContent?.replace(/\s/g, ' ') ?? '';
    expect(text).toContain('Ала-Куль');
    expect(text).toContain('Оплачен');
    expect(text).toContain('тест');
    expect(text).toContain('оплачено 24 000 сом');

    const select = el.querySelector('select') as HTMLSelectElement;
    select.value = 'EXPIRED';
    select.dispatchEvent(new Event('change'));
    const from = el.querySelector('input[type=date]') as HTMLInputElement;
    from.value = '2026-10-01';
    from.dispatchEvent(new Event('change'));
    TestBed.tick();
    api.expectOne('/payments/admin/transactions?status=EXPIRED&from=2026-10-01').flush([]);
    await fixture.whenStable();
    expect(el.textContent).toContain('Платежей нет');
  });
});

describe('OnlinePaymentPage', () => {
  let api: HttpTestingController;

  async function render(t: Transaction): Promise<HTMLElement> {
    TestBed.configureTestingModule({
      imports: [OnlinePaymentPage],
      providers: pageTestProviders(),
    });
    api = TestBed.inject(HttpTestingController);
    const fixture = TestBed.createComponent(OnlinePaymentPage);
    fixture.componentRef.setInput('id', 'o1');
    TestBed.tick();
    api.expectOne('/payments/admin/gateway').flush(gateway);
    api.expectOne('/payments/admin/transactions/o1').flush(t);
    await fixture.whenStable();
    return fixture.nativeElement as HTMLElement;
  }

  it('checks a payment at GoPay', async () => {
    const el = await render(transaction());
    button(el, 'Проверить в GoPay').click();
    const req = api.expectOne('/payments/admin/transactions/o1/check');
    expect(req.request.method).toBe('POST');
    req.flush(transaction({ status: 'COMMITTED' }, { status: 'paid' }));
    await settle();
    expect(el.textContent).toContain('Статус обновлён');
    expect(el.textContent).toContain('Оплачен');
    expect(el.textContent).toContain('Отметить возврат');
  });

  it('cancels an unpaid payment after confirming', async () => {
    const el = await render(transaction());
    button(el, 'Отменить платёж').click();
    TestBed.tick();
    button(el, 'Да, отменить').click();
    api
      .expectOne('/payments/admin/transactions/o1/cancel')
      .flush({ message: 'conflict' }, { status: 409, statusText: 'Conflict' });
    await settle();
    expect(el.textContent).toContain('Платёж уже нельзя отменить');
    api
      .expectOne('/payments/admin/transactions/o1')
      .flush(transaction({ status: 'COMMITTED' }, { status: 'paid' }));
  });

  it('marks a refund with a note', async () => {
    const el = await render(transaction({ status: 'COMMITTED' }, { status: 'paid' }));
    expect(el.textContent).not.toContain('Отменить платёж');
    button(el, 'Отметить возврат').click();
    TestBed.tick();
    const note = el.querySelector('input.ui-input') as HTMLInputElement;
    note.value = ' Операция 77 ';
    note.dispatchEvent(new Event('input'));
    const buttons = [...el.querySelectorAll('button')].filter(
      (b) => b.textContent?.trim() === 'Отметить возврат',
    );
    buttons[buttons.length - 1].click();
    const req = api.expectOne('/orders/o1/refund');
    expect(req.request.body).toEqual({ note: 'Операция 77' });
    req.flush({});
    await settle();
    expect(el.textContent).toContain('Возврат отмечен');
    api
      .expectOne('/payments/admin/transactions/o1')
      .flush(transaction({ status: 'COMMITTED' }, { status: 'refunded' }));
  });
});
