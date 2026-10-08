const INTL_LOCALES: Record<string, string> = {
  ru: 'ru-RU',
  ky: 'ky-KG',
  en: 'en-US',
  de: 'de-DE',
};

// Thousands-separated number for prices in som, e.g. "12 500".
export function formatAmount(locale: string, amount: number): string {
  return new Intl.NumberFormat(INTL_LOCALES[locale] ?? locale).format(amount);
}
