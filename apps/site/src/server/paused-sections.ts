import { localizedPath, splitLang } from '@/core/i18n/langs';

// Sections taken off the site for now (the focus is on outings and tours),
// with the section each one's old addresses go to. Their pages, links and
// sitemaps are gone; the addresses redirect temporarily (302) so search
// engines keep them for when the sections return.
const PAUSED_SECTIONS: Record<string, string> = {
  hotels: '/tours',
  'local-guides': '/tours',
  guides: '/tours',
  trails: '/outings',
  payments: '/tours',
  // The hiking hub only linked to outings and tours, which the menu now
  // has itself.
  hikes: '/outings',
};

// Where an address in a paused section goes, in the same language; null
// for every other address.
export function pausedSectionRedirect(url: string): string | null {
  const { lang, path } = splitLang(url);
  const target = PAUSED_SECTIONS[path.split('/')[1] ?? ''];
  return target ? localizedPath(lang, target) : null;
}
