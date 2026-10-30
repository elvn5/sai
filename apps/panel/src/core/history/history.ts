// The panel's change history: each service records the changes made with
// an admin session (services/pkg/audit) and serves them per module.

export type HistoryModule =
  | 'users'
  | 'partners'
  | 'support'
  | 'achievements'
  | 'trails'
  | 'payments'
  | 'blog'
  | 'seo';

export interface HistoryEntry {
  id: string;
  module: HistoryModule;
  method: 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  // The route pattern, e.g. "/blog/:id".
  route: string;
  path: string;
  entityId: string | null;
  // What changed, when the service named it (e.g. a partner's name).
  subject: string | null;
  // The request body, with secrets hidden and long texts cut.
  payload: unknown;
  status: number;
  // The panel has one shared password: the client tells sessions apart.
  ip: string;
  userAgent: string;
  createdAt: string;
}

export const HISTORY_MODULES: { module: HistoryModule; label: string; url: string }[] = [
  { module: 'support', label: 'Поддержка', url: '/support/admin/history' },
  { module: 'users', label: 'Пользователи', url: '/users/admin/history' },
  { module: 'partners', label: 'Партнёры', url: '/partners/admin/history' },
  { module: 'achievements', label: 'Достижения', url: '/achievements/admin/history' },
  { module: 'trails', label: 'Маршруты', url: '/trails/admin/history' },
  { module: 'payments', label: 'Платежи', url: '/orders/admin/history' },
  { module: 'blog', label: 'Блог', url: '/blog/admin/history' },
  { module: 'seo', label: 'SEO', url: '/page-seo/admin/history' },
];

export function moduleLabel(module: HistoryModule): string {
  return HISTORY_MODULES.find((m) => m.module === module)?.label ?? module;
}

// What each admin change means, by "METHOD route".
const ACTIONS: Record<string, string> = {
  'POST /partners/:userId/approve': 'Заявка одобрена',
  'POST /partners/:userId/reject': 'Заявка отклонена',
  'POST /partners': 'Партнёр добавлен',
  'PUT /partners/:userId': 'Партнёр изменён',
  'DELETE /partners/:userId': 'Статус партнёра снят',
  'PUT /support': 'Контакты поддержки изменены',
  'POST /achievements': 'Достижение создано',
  'PUT /achievements/:id': 'Достижение изменено',
  'DELETE /achievements/:id': 'Достижение удалено',
  'DELETE /achievements/awards/:id': 'Награда отозвана',
  'POST /trails': 'Маршрут создан',
  'PUT /trails/:key': 'Маршрут изменён',
  'DELETE /trails/:key': 'Маршрут удалён',
  'DELETE /conditions/:id': 'Сводка о состоянии удалена',
  'POST /orders/:id/refund': 'Возврат по заказу',
  'POST /payouts': 'Выплата создана',
  'POST /payments/admin/transactions/:id/check': 'Платёж проверен в GoPay',
  'POST /payments/admin/transactions/:id/cancel': 'Платёж отменён в GoPay',
  'POST /blog': 'Статья создана',
  'PUT /blog/:id': 'Статья изменена',
  'DELETE /blog/:id': 'Статья удалена',
  'PUT /page-seo/:page/:locale': 'SEO страницы изменено',
};

const METHODS: Record<HistoryEntry['method'], string> = {
  POST: 'Создание',
  PUT: 'Изменение',
  PATCH: 'Изменение',
  DELETE: 'Удаление',
};

export function actionLabel(entry: HistoryEntry): string {
  return ACTIONS[`${entry.method} ${entry.route}`] ?? `${METHODS[entry.method]} · ${entry.route}`;
}

export function actionTone(entry: HistoryEntry): string {
  if (
    entry.method === 'DELETE' ||
    entry.route.endsWith('/reject') ||
    entry.route.endsWith('/cancel')
  )
    return 'bg-red-100 text-red-800';
  if (entry.method === 'POST') return 'bg-green-100 text-green-800';
  return 'bg-sky-100 text-sky-800';
}

// A readable name of what changed: the one the service recorded, else a
// title, a name or a person's name from the request; else the entity's id.
export function subjectLabel(entry: HistoryEntry): string {
  if (entry.subject) return entry.subject;
  const payload = entry.payload;
  if (payload && typeof payload === 'object' && !Array.isArray(payload)) {
    const p = payload as Record<string, unknown> & { firstName?: unknown; lastName?: unknown };
    for (const key of ['title', 'name', 'slug']) {
      const value = p[key];
      if (typeof value === 'string' && value.trim()) return value;
      if (value && typeof value === 'object') {
        const { ru } = value as { ru?: unknown };
        if (typeof ru === 'string' && ru.trim()) return ru;
      }
    }
    const person = [p.firstName, p.lastName].filter((v) => typeof v === 'string' && v).join(' ');
    if (person) return person;
  }
  return entry.entityId ?? '—';
}

// Merges pages from several modules into one, newest first, cut to limit.
export function mergeHistory(pages: HistoryEntry[][], limit: number): HistoryEntry[] {
  return pages
    .flat()
    .sort((a, b) => b.createdAt.localeCompare(a.createdAt))
    .slice(0, limit);
}
