import type { Person } from '@/core/api/types';

export function som(amount: number): string {
  return `${amount.toLocaleString('ru-RU')} сом`;
}

export function formatDate(value: string): string {
  return new Date(value).toLocaleDateString('ru-RU');
}

export function formatDateTime(value: string): string {
  return new Date(value).toLocaleString('ru-RU');
}

// A person's display name: first name, else @username, else a short id.
export function personName(
  person: Pick<Person, 'id' | 'firstName' | 'username'> | null | undefined,
): string {
  if (!person) return '—';
  return person.firstName ?? (person.username ? `@${person.username}` : person.id.slice(0, 8));
}

export function fullName(user: { firstName: string | null; lastName: string | null }): string {
  return [user.firstName, user.lastName].filter(Boolean).join(' ') || '—';
}
