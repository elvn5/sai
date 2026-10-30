import type {
  AdminTourGroup,
  AdminTourInfo,
  AdminTourList,
  AppUser,
  TourParticipant,
} from '@/core/api/types';
import { TourPage } from '@/pages/tours/tour.page';
import { ToursPage } from '@/pages/tours/tours.page';
import { pageTestProviders } from '@/testing/providers';
import { HttpTestingController } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';

const tour: AdminTourInfo = {
  id: 't1',
  locale: 'ru',
  slug: 'ala-kul',
  title: 'Ала-Куль',
  location: 'Каракол',
  price: 12000,
  startDate: '2026-10-30',
  endDate: '2026-11-01',
  meetingTime: '06:00',
  meetingPoint: 'Южные ворота',
  capacity: 2,
  taken: 2,
  phone: null,
  whatsapp: null,
  telegram: '@partner_kg',
  isPublished: true,
  publishedAt: '2026-10-01T00:00:00Z',
  userId: 'org',
  organizer: { id: 'org', firstName: 'Азамат', username: 'azamat', avatarUrl: null },
  createdAt: '2026-10-01T00:00:00Z',
};

const list: AdminTourList = {
  tours: [
    {
      tour,
      phase: 'current',
      counts: {
        active: 2,
        waitlist: 1,
        cancelled: 0,
        arrived: 1,
        forms: 1,
        waivers: 2,
        due: 24000,
        paid: 12000,
      },
    },
  ],
  counts: { current: 1, upcoming: 3, past: 5 },
};

function participant(patch: Partial<TourParticipant>): TourParticipant {
  return {
    id: 'p1',
    userId: 'u1',
    account: { id: 'u1', firstName: 'Айгерим', username: 'aigerim', avatarUrl: null },
    name: null,
    phone: null,
    telegram: null,
    status: 'confirmed',
    joinedAt: '2026-10-20T10:00:00Z',
    arrivedAt: null,
    amountDue: null,
    due: 12000,
    paid: 12000,
    payments: [
      {
        id: 'pay1',
        kind: 'payment',
        amount: 12000,
        method: 'cash',
        paidOn: '2026-10-21',
        note: null,
        createdAt: '2026-10-21T00:00:00Z',
      },
    ],
    form: {
      emergencyName: 'Мама',
      emergencyPhone: '+996 555 000 111',
      experience: 'trekking',
      health: 'Аллергия на мёд',
      notes: null,
    },
    formAt: '2026-10-20T10:00:00Z',
    waiverAt: '2026-10-20T10:00:00Z',
    waiverVersion: 'v1',
    ...patch,
  };
}

function group(
  participants: TourParticipant[],
  patch: Partial<AdminTourGroup> = {},
): AdminTourGroup {
  return {
    tour,
    phase: 'current',
    organizer: tour.organizer ?? null,
    participants,
    totals: {
      active: 2,
      waitlist: 1,
      arrived: 0,
      due: 24000,
      paid: 12000,
      forms: 1,
      waivers: 1,
      expenses: 0,
      margin: 12000,
    },
    expenses: [],
    staff: [],
    ...patch,
  };
}

const users: AppUser[] = [
  {
    id: 'u1',
    telegramId: null,
    email: 'aigerim@example.com',
    username: 'aigerim',
    firstName: 'Айгерим',
    lastName: 'Садыкова',
    phone: '+996 700 123 456',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    partnerStatus: 'none' as AppUser['partnerStatus'],
    partnerRequestedAt: null,
    partnerKind: 'organizer',
  },
];

async function settle(): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve));
  TestBed.tick();
}

function button(el: HTMLElement, text: string): HTMLButtonElement {
  const found = [...el.querySelectorAll('button')].find((b) =>
    b.textContent?.trim().startsWith(text),
  );
  if (!found) throw new Error(`no button "${text}"`);
  return found;
}

describe('ToursPage', () => {
  it('shows the tours on the way, then other phases and search', async () => {
    TestBed.configureTestingModule({ imports: [ToursPage], providers: pageTestProviders() });
    const api = TestBed.inject(HttpTestingController);
    const fixture = TestBed.createComponent(ToursPage);
    TestBed.tick();
    api.expectOne('/tours/admin?phase=current').flush(list);
    await fixture.whenStable();
    const el = fixture.nativeElement as HTMLElement;
    const text = el.textContent?.replace(/\s+/g, ' ') ?? '';
    expect(text).toContain('Ала-Куль');
    expect(text).toContain('идёт сейчас');
    expect(text).toContain('Азамат');
    expect(text).toContain('2 из 2');
    expect(text).toContain('+1 в ожидании');
    expect(text).toContain('Все 9');
    expect(text).toContain('на месте сбора отмечено: 1');

    button(el, 'Предстоящие').click();
    TestBed.tick();
    api.expectOne('/tours/admin?phase=upcoming').flush({ tours: [], counts: list.counts });
    await fixture.whenStable();
    expect(el.textContent).toContain('Предстоящих туров нет');

    const search = el.querySelector('input[type=search]') as HTMLInputElement;
    search.value = 'Азамат';
    search.dispatchEvent(new Event('input'));
    TestBed.tick();
    api
      .expectOne(`/tours/admin?phase=upcoming&q=${encodeURIComponent('Азамат')}`)
      .flush({ tours: [], counts: list.counts });
    await fixture.whenStable();
    expect(el.textContent).toContain('Ничего не нашлось');
  });
});

describe('TourPage', () => {
  let api: HttpTestingController;

  async function render(g: AdminTourGroup): Promise<HTMLElement> {
    TestBed.configureTestingModule({ imports: [TourPage], providers: pageTestProviders() });
    api = TestBed.inject(HttpTestingController);
    const fixture = TestBed.createComponent(TourPage);
    fixture.componentRef.setInput('id', 't1');
    TestBed.tick();
    api.expectOne('/tours/admin/t1').flush(g);
    api.expectOne('/users').flush(users);
    await fixture.whenStable();
    return fixture.nativeElement as HTMLElement;
  }

  const waiting = participant({
    id: 'p2',
    userId: null,
    account: null,
    name: 'Бакыт',
    phone: '+996 555 222 333',
    status: 'waitlist',
    form: null,
    formAt: null,
    waiverAt: null,
    payments: [],
    paid: 0,
  });

  it('shows the group with contacts, forms and payments', async () => {
    const el = await render(group([participant({}), waiting]));
    const text = el.textContent?.replace(/\s+/g, ' ') ?? '';
    expect(text).toContain('Айгерим');
    expect(text).toContain('Айгерим Садыкова');
    expect(text).toContain('aigerim@example.com');
    expect(text).toContain('Бакыт');
    expect(text).toContain('добавлен организатором');
    expect(text).toContain('правила не приняты');
    expect(text).toContain('С местом 1');
    expect(text).toContain('Лист ожидания 1');

    button(el, 'Подробнее').click();
    TestBed.tick();
    const details = el.textContent?.replace(/\s+/g, ' ') ?? '';
    expect(details).toContain('Мама, +996 555 000 111');
    expect(details).toContain('Многодневные треки');
    expect(details).toContain('Аллергия на мёд');
    expect(details).toContain('Наличные');

    button(el, 'Лист ожидания').click();
    TestBed.tick();
    expect(el.textContent).not.toContain('Айгерим Садыкова');
  });

  it('changes a status and marks arrival', async () => {
    const el = await render(group([participant({}), waiting]));
    const selects = el.querySelectorAll('select');
    expect((selects[0] as HTMLSelectElement).value).toBe('confirmed');
    (selects[1] as HTMLSelectElement).value = 'confirmed';
    selects[1].dispatchEvent(new Event('change'));
    const req = api.expectOne('/tours/admin/t1/participants/p2');
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual({ status: 'confirmed' });
    req.flush({ message: 'No places left' }, { status: 409, statusText: 'Conflict' });
    await settle();
    expect(el.textContent).toContain('Мест нет');
    api.expectOne('/tours/admin/t1').flush(group([participant({}), waiting]));
    await settle();

    const box = el.querySelector('input[type=checkbox]') as HTMLInputElement;
    box.checked = true;
    box.dispatchEvent(new Event('change'));
    const arrival = api.expectOne('/tours/admin/t1/participants/p1');
    expect(arrival.request.body).toEqual({ arrived: true });
    arrival.flush(group([participant({ arrivedAt: '2026-10-30T00:30:00Z' }), waiting]));
    await settle();
    expect(el.textContent).toContain('Айгерим отмечен на сборе');
  });

  it('hides the tour from the catalog after confirming', async () => {
    const el = await render(group([]));
    expect(el.textContent).toContain('На тур пока никто не записался');
    button(el, 'Скрыть из каталога').click();
    TestBed.tick();
    button(el, 'Да, скрыть').click();
    const req = api.expectOne('/tours/admin/t1/published');
    expect(req.request.body).toEqual({ published: false });
    req.flush(group([], { tour: { ...tour, isPublished: false } }));
    await settle();
    expect(el.textContent).toContain('Тур скрыт из каталога');
    expect(el.textContent).toContain('Вернуть в каталог');
  });
});
