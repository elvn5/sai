import { DEFAULT_LANG, LANGS, type Lang, localizedPath } from '@/core/i18n/langs';
import { publishedLegalPaths } from '@/core/legal/documents';

// sitemap.xml for the SSR server: an index (/sitemap.xml) of one sitemap per
// content type (/sitemaps/<type>.xml), with lastmod and hreflang alternates
// (docs/angular-migration-plan.md, section 4).

// Hotels, local guides, trails and travel guides are paused for now
// (server/paused-sections.ts): no sitemaps of theirs.
export const SITEMAP_TYPES = ['pages', 'tours', 'blog', 'organizers'] as const;
export type SitemapType = (typeof SITEMAP_TYPES)[number];

export function isSitemapType(value: string): value is SitemapType {
  return (SITEMAP_TYPES as readonly string[]).includes(value);
}

// Section pages, the same in every language (plus the finished legal
// documents).
const SECTION_PATHS = [
  '/',
  '/tours',
  '/outings',
  '/blog',
  '/gallery',
  '/hall-of-fame',
  ...publishedLegalPaths(),
];

// A GET of the API returning parsed JSON; throws when the API fails.
export type GetJson = (path: string) => Promise<unknown>;

export interface SitemapEntry {
  // Unprefixed path ("/tours/song-kol").
  path: string;
  lastmod?: string | null;
  // One language only (its own slug), or every language (hreflang).
  lang?: Lang;
}

interface Published {
  slug: string;
  publishedAt: string | null;
  userId?: string | null;
}

// Tours and blog posts are written per language, each with its own slug.
async function perLanguage(getJson: GetJson, section: string): Promise<SitemapEntry[]> {
  const lists = await Promise.all(
    LANGS.map(async (lang) => ({
      lang,
      items: (await getJson(`/${section}?locale=${lang}`)) as Published[],
    })),
  );
  return lists.flatMap(({ lang, items }) =>
    items.map((item) => ({ path: `/${section}/${item.slug}`, lastmod: item.publishedAt, lang })),
  );
}

// Organizers' pages, one per partner running a published tour in any
// language (outing organizers aren't listed: their outings are gone in
// days). The same page in every language; lastmod is their latest tour.
async function organizers(getJson: GetJson): Promise<SitemapEntry[]> {
  const lists = await Promise.all(
    LANGS.map(async (lang) => (await getJson(`/tours?locale=${lang}`)) as Published[]),
  );
  const latest = new Map<string, string | null>();
  for (const tour of lists.flat()) {
    if (!tour.userId) continue;
    const seen = latest.get(tour.userId);
    const newer = !seen || (tour.publishedAt !== null && tour.publishedAt > seen);
    latest.set(tour.userId, newer ? tour.publishedAt : seen);
  }
  return [...latest].map(([id, lastmod]) => ({ path: `/organizers/${id}`, lastmod }));
}

export async function collectEntries(type: SitemapType, getJson: GetJson): Promise<SitemapEntry[]> {
  switch (type) {
    case 'pages':
      return SECTION_PATHS.map((path) => ({ path }));
    case 'tours':
    case 'blog':
      return perLanguage(getJson, type);
    case 'organizers':
      return organizers(getJson);
  }
}

function escapeXml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function lastmodTag(lastmod: string | null | undefined): string {
  if (!lastmod) return '';
  const date = new Date(lastmod);
  return Number.isNaN(date.getTime()) ? '' : `<lastmod>${date.toISOString()}</lastmod>`;
}

function link(hreflang: string, href: string): string {
  return `<xhtml:link rel="alternate" hreflang="${hreflang}" href="${escapeXml(href)}"/>`;
}

const XML_DECLARATION = '<?xml version="1.0" encoding="UTF-8"?>';
const SITEMAP_NS = 'http://www.sitemaps.org/schemas/sitemap/0.9';
const XHTML_NS = 'http://www.w3.org/1999/xhtml';

export function renderUrlset(siteUrl: string, entries: SitemapEntry[]): string {
  const urls = entries.flatMap((entry) => {
    const lastmod = lastmodTag(entry.lastmod);
    if (entry.lang) {
      const loc = `${siteUrl}${localizedPath(entry.lang, entry.path)}`;
      return [`<url><loc>${escapeXml(loc)}</loc>${lastmod}</url>`];
    }
    // Every language links to all of them (and itself), as Google asks.
    const alternates = [
      ...LANGS.map((lang) => link(lang, `${siteUrl}${localizedPath(lang, entry.path)}`)),
      link('x-default', `${siteUrl}${localizedPath(DEFAULT_LANG, entry.path)}`),
    ].join('');
    return LANGS.map((lang) => {
      const loc = `${siteUrl}${localizedPath(lang, entry.path)}`;
      return `<url><loc>${escapeXml(loc)}</loc>${lastmod}${alternates}</url>`;
    });
  });
  return `${XML_DECLARATION}<urlset xmlns="${SITEMAP_NS}" xmlns:xhtml="${XHTML_NS}">${urls.join('')}</urlset>`;
}

export function renderIndex(siteUrl: string): string {
  const sitemaps = SITEMAP_TYPES.map(
    (type) => `<sitemap><loc>${escapeXml(`${siteUrl}/sitemaps/${type}.xml`)}</loc></sitemap>`,
  ).join('');
  return `${XML_DECLARATION}<sitemapindex xmlns="${SITEMAP_NS}">${sitemaps}</sitemapindex>`;
}

// Keeps each rendered sitemap for `ttlMs`, so crawlers don't hit the API on
// every request. A failed API call is not cached and fails the request: a
// short outage must not look like every page was removed.
export class SitemapCache {
  private readonly entries = new Map<SitemapType, { xml: string; expires: number }>();

  constructor(
    private readonly siteUrl: string,
    private readonly getJson: GetJson,
    private readonly ttlMs = 10 * 60 * 1000,
    private readonly now: () => number = Date.now,
  ) {}

  async get(type: SitemapType): Promise<string> {
    const cached = this.entries.get(type);
    if (cached && cached.expires > this.now()) return cached.xml;
    const xml = renderUrlset(this.siteUrl, await collectEntries(type, this.getJson));
    this.entries.set(type, { xml, expires: this.now() + this.ttlMs });
    return xml;
  }
}
