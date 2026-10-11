import type { LocalizedText, Person } from '~/shared/api/types';

// The text for locale, falling back to Russian (always present).
export function localizedText(text: LocalizedText, locale: string): string {
  return text[locale as keyof LocalizedText] ?? text.ru ?? '';
}

// How a person is named on public pages.
export function personName(person: Person): string {
  return person.firstName ?? (person.username ? `@${person.username}` : '—');
}
