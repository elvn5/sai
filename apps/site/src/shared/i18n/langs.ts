// The site's languages. Russian is served without a prefix (/tours),
// the others under their code (/en/tours) — the URL scheme of the Nuxt
// site, kept as is for search engines.
export const LANGS = ['ru', 'ky', 'en', 'de'] as const;
export type Lang = (typeof LANGS)[number];
export const DEFAULT_LANG: Lang = 'ru';

export function isLang(value: unknown): value is Lang {
  return typeof value === 'string' && (LANGS as readonly string[]).includes(value);
}

// The path of `path` (absolute, e.g. "/tours") in `lang`.
export function localizedPath(lang: Lang, path: string): string {
  if (lang === DEFAULT_LANG) return path;
  return path === '/' ? `/${lang}` : `/${lang}${path}`;
}
