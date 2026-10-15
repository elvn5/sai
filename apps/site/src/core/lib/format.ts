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

const NUMBER_LOCALES: Record<string, string> = {
  ru: 'ru-RU',
  ky: 'ky-KG',
  en: 'en-US',
  de: 'de-DE',
};

// A price in som with thousands separators ("12 500").
export function formatAmount(lang: string, amount: number): string {
  return new Intl.NumberFormat(NUMBER_LOCALES[lang] ?? lang).format(amount);
}

// "3 июн. 2026 г. – 7 июн. 2026 г." for calendar dates (YYYY-MM-DD), read
// as dates, not instants: no time zone can shift them a day.
export function formatDateRange(lang: string, start: string, end: string): string {
  const format = (date: string): string =>
    new Intl.DateTimeFormat(NUMBER_LOCALES[lang] ?? lang, {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
      timeZone: 'UTC',
    }).format(new Date(`${date.slice(0, 10)}T00:00:00Z`));
  return start === end ? format(start) : `${format(start)} – ${format(end)}`;
}

// Today in Kyrgyzstan as YYYY-MM-DD (tours are bookable until they start).
export function todayInBishkek(now: Date = new Date()): string {
  return now.toLocaleDateString('sv-SE', { timeZone: 'Asia/Bishkek' });
}
