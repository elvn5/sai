// The site's languages. Russian is served without a prefix (/tours),
// the others under their code (/en/tours) — the URL scheme of the Nuxt
// site, kept as is for search engines.
export const LANGS = ['ru', 'ky', 'en', 'de'] as const;
export type Lang = (typeof LANGS)[number];
export const DEFAULT_LANG: Lang = 'ru';

// Each language in its own language, for the switcher.
export const LANG_NAMES: Record<Lang, string> = {
  ru: 'Русский',
  ky: 'Кыргызча',
  en: 'English',
  de: 'Deutsch',
};

// BCP 47 tags (hreflang) and Open Graph locales.
export const LANG_TAGS: Record<Lang, string> = {
  ru: 'ru-RU',
  ky: 'ky-KG',
  en: 'en-US',
  de: 'de-DE',
};
export const OG_LOCALES: Record<Lang, string> = {
  ru: 'ru_RU',
  ky: 'ky_KG',
  en: 'en_US',
  de: 'de_DE',
};

export function isLang(value: unknown): value is Lang {
  return typeof value === 'string' && (LANGS as readonly string[]).includes(value);
}

// The path of `path` (absolute, e.g. "/tours") in `lang`.
export function localizedPath(lang: Lang, path: string): string {
  if (lang === DEFAULT_LANG) return path;
  return path === '/' ? `/${lang}` : `/${lang}${path}`;
}

// Splits a URL ("/en/tours?page=2#top") into its language and the path
// without the prefix, query or fragment ("/tours").
export function splitLang(url: string): { lang: Lang; path: string } {
  const pathname = url.split(/[?#]/, 1)[0] || '/';
  const [, first, ...rest] = pathname.split('/');
  if (isLang(first) && first !== DEFAULT_LANG) {
    return { lang: first, path: `/${rest.join('/')}`.replace(/\/+$/, '') || '/' };
  }
  return { lang: DEFAULT_LANG, path: pathname.replace(/\/+$/, '') || '/' };
}
