import type { CrmGroup, CrmParticipant } from '@/core/api/types';
import { clientName, groupRows } from '@/pages/partner/crm';

function participant(fields: Partial<CrmParticipant>): CrmParticipant {
  return {
    id: 'p',
    clientId: 'c',
    userId: null,
    account: null,
    name: null,
    phone: null,
    telegram: null,
    status: 'new',
    joinedAt: '2026-10-01T00:00:00Z',
    arrivedAt: null,
    amountDue: null,
    due: 5000,
    paid: 0,
    payments: [],
    form: null,
    formAt: null,
    waiverAt: null,
    waiverVersion: null,
    ...fields,
  };
}

describe('CRM helpers', () => {
  it('names a client by the typed name, else the account', () => {
    const account = { id: 'u', firstName: 'Aibek', username: 'aibek', avatarUrl: null };
    expect(clientName({ name: 'Айбек', account })).toBe('Айбек');
    expect(clientName({ name: null, account })).toBe('Aibek');
    expect(clientName({ name: null, account: null })).toBe('—');
  });

  it('exports the group without the cancelled, with debts', () => {
    const group = {
      participants: [
        participant({
          name: 'Айбек',
          phone: '+996 555 1',
          paid: 2000,
          arrivedAt: '2026-10-17T00:00:00Z',
          form: {
            emergencyName: 'Мама',
            emergencyPhone: '+996 555 2',
            experience: 'hiking',
            health: 'астма',
            notes: null,
          },
          waiverAt: '2026-10-10T00:00:00Z',
        }),
        participant({
          account: { id: 'u', firstName: 'Ann', username: 'ann_kg', avatarUrl: null },
          status: 'waitlist',
        }),
        participant({ name: 'Bob', status: 'cancelled' }),
      ],
    } as CrmGroup;
    const rows = groupRows(group, {
      headers: ['h'],
      status: (s) => `[${s}]`,
      experience: (e) => `<${e}>`,
      yes: 'yes',
    });
    expect(rows).toEqual([
      ['h'],
      [
        1,
        'Айбек',
        '+996 555 1',
        null,
        '[new]',
        'yes',
        5000,
        2000,
        3000,
        'Мама +996 555 2',
        '<hiking>',
        'астма',
        'yes',
      ],
      [2, 'Ann', null, '@ann_kg', '[waitlist]', '', 5000, 0, 5000, null, null, null, ''],
    ]);
  });
});
