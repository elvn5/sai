import { localizedPath, splitLang } from '@/core/i18n/langs';

// Sections taken off the site for now (the focus is on tours), with the
// section each one's old addresses go to. Their pages, links and sitemaps
// are gone; the addresses redirect temporarily (302) so search engines
// keep them for when the sections return.
const PAUSED_SECTIONS: Record<string, string> = {
  hotels: '/tours',
  'local-guides': '/tours',
  guides: '/tours',
  payments: '/tours',
  // The hiking hub only linked to sections the menu now has itself.
  hikes: '/tours',
};

// Sections removed for good, with where their addresses moved (301):
// outings merged into tours (travelers sign up for a tour now).
const MOVED_SECTIONS: Record<string, string> = {
  outings: '/tours',
};

function redirect(url: string, sections: Record<string, string>): string | null {
  const { lang, path } = splitLang(url);
  const target = sections[path.split(/[/?#]/)[1] ?? ''];
  return target ? localizedPath(lang, target) : null;
}

// Where an address in a paused section goes, in the same language; null
// for every other address.
export function pausedSectionRedirect(url: string): string | null {
  return redirect(url, PAUSED_SECTIONS);
}

// Where an address in a removed section goes for good, in the same
// language; null for every other address.
export function movedSectionRedirect(url: string): string | null {
  return redirect(url, MOVED_SECTIONS);
}
