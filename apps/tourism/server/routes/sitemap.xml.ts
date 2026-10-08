const DEFAULT_LOCALE = 'ru';
const LOCALES = ['ru', 'ky', 'en', 'de'];
const STATIC_PATHS = ['/', '/blog', '/gallery', '/guides', '/tours', '/hotels', '/local-guides'];
const LISTING_PAGE_SIZE = 50;

interface PublishedItem {
  slug: string;
  publishedAt: string | null;
}

interface ListingItem {
  slug: string;
  updatedAt: string;
}

interface SitemapEntry {
  loc: string;
  lastmod?: string;
  priority?: string;
}

function localizedPath(locale: string, path: string): string {
  return locale === DEFAULT_LOCALE ? path : `/${locale}${path}`;
}

function escapeXml(value: string): string {
  return value.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

// Hotels and guides are paginated and language-independent, so they're
// fetched once and listed under every locale.
async function fetchAllListings(apiBaseUrl: string, path: string): Promise<ListingItem[]> {
  const items: ListingItem[] = [];
  for (let offset = 0; ; offset += LISTING_PAGE_SIZE) {
    const page = await $fetch<{ items: ListingItem[]; total: number }>(
      `${apiBaseUrl}${path}?limit=${LISTING_PAGE_SIZE}&offset=${offset}&sort=newest`,
    ).catch(() => ({ items: [], total: 0 }));
    items.push(...page.items);
    if (page.items.length < LISTING_PAGE_SIZE || items.length >= page.total) return items;
  }
}

async function collectEntries(apiBaseUrl: string): Promise<SitemapEntry[]> {
  const entries: SitemapEntry[] = [];
  const [hotels, localGuides] = await Promise.all([
    fetchAllListings(apiBaseUrl, '/hotels'),
    fetchAllListings(apiBaseUrl, '/local-guides'),
  ]);

  for (const locale of LOCALES) {
    for (const path of STATIC_PATHS) {
      entries.push({
        loc: localizedPath(locale, path),
        priority: path === '/' ? '1.0' : undefined,
      });
    }

    const [blogPosts, guides, tours] = await Promise.all([
      $fetch<PublishedItem[]>(`${apiBaseUrl}/blog?locale=${locale}`).catch(() => []),
      $fetch<PublishedItem[]>(`${apiBaseUrl}/guides?locale=${locale}`).catch(() => []),
      $fetch<PublishedItem[]>(`${apiBaseUrl}/tours?locale=${locale}`).catch(() => []),
    ]);

    for (const post of blogPosts) {
      entries.push({
        loc: localizedPath(locale, `/blog/${post.slug}`),
        lastmod: post.publishedAt ?? undefined,
      });
    }
    for (const guide of guides) {
      entries.push({
        loc: localizedPath(locale, `/guides/${guide.slug}`),
        lastmod: guide.publishedAt ?? undefined,
      });
    }
    for (const tour of tours) {
      entries.push({
        loc: localizedPath(locale, `/tours/${tour.slug}`),
        lastmod: tour.publishedAt ?? undefined,
      });
    }
    for (const hotel of hotels) {
      entries.push({
        loc: localizedPath(locale, `/hotels/${hotel.slug}`),
        lastmod: hotel.updatedAt,
      });
    }
    for (const guide of localGuides) {
      entries.push({
        loc: localizedPath(locale, `/local-guides/${guide.slug}`),
        lastmod: guide.updatedAt,
      });
    }
  }

  return entries;
}

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig();
  const siteUrl = config.public.siteUrl.replace(/\/$/, '');
  const entries = await collectEntries(config.public.apiBaseUrl);

  const body = entries
    .map((entry) => {
      const loc = `${siteUrl}${entry.loc}`;
      const lastmod = entry.lastmod
        ? `<lastmod>${new Date(entry.lastmod).toISOString()}</lastmod>`
        : '';
      const priority = entry.priority ? `<priority>${entry.priority}</priority>` : '';
      return `<url><loc>${escapeXml(loc)}</loc>${lastmod}${priority}</url>`;
    })
    .join('');

  setHeader(event, 'Content-Type', 'application/xml; charset=utf-8');
  return `<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">${body}</urlset>`;
});
