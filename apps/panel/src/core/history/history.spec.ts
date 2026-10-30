import { type HistoryEntry, actionLabel, mergeHistory, subjectLabel } from '@/core/history/history';

function entry(over: Partial<HistoryEntry>): HistoryEntry {
  return {
    id: '1',
    module: 'blog',
    method: 'PUT',
    route: '/blog/:id',
    path: '/blog/p1',
    entityId: 'p1',
    subject: null,
    payload: null,
    status: 200,
    ip: '10.0.0.1',
    userAgent: '',
    createdAt: '2026-10-30T08:00:00.000Z',
    ...over,
  };
}

describe('history', () => {
  it('names known actions and falls back to the route', () => {
    expect(actionLabel(entry({}))).toBe('Статья изменена');
    expect(actionLabel(entry({ method: 'DELETE', route: '/blog/:id/x' }))).toBe(
      'Удаление · /blog/:id/x',
    );
  });

  it('names the subject from the request', () => {
    expect(subjectLabel(entry({ payload: { title: 'Ала-Куль' } }))).toBe('Ала-Куль');
    expect(subjectLabel(entry({ payload: { title: { ru: 'Покоритель' } } }))).toBe('Покоритель');
    expect(subjectLabel(entry({ payload: { firstName: 'Айбек', lastName: 'Осмонов' } }))).toBe(
      'Айбек Осмонов',
    );
    expect(subjectLabel(entry({ payload: null }))).toBe('p1');
    expect(subjectLabel(entry({ subject: 'Айбек', payload: { title: 'x' } }))).toBe('Айбек');
  });

  it('merges modules newest first', () => {
    const a = entry({ id: 'a', createdAt: '2026-10-30T08:00:00.000Z' });
    const b = entry({ id: 'b', createdAt: '2026-10-30T09:00:00.000Z' });
    const c = entry({ id: 'c', createdAt: '2026-10-29T09:00:00.000Z' });
    expect(mergeHistory([[a, c], [b]], 2).map((e) => e.id)).toEqual(['b', 'a']);
  });
});
