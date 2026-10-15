import type { LocalizedText, Person } from '@/core/api/types';

// Text the admin wrote per language, falling back to Russian.
export function localizedText(text: LocalizedText, lang: string): string {
  return text[lang as keyof LocalizedText] ?? text.ru ?? '';
}

// How a person is named on public pages.
export function personName(person: Person): string {
  return person.firstName ?? (person.username ? `@${person.username}` : '—');
}

export function fileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
