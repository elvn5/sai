import {
  type GetJson,
  SitemapCache,
  collectEntries,
  renderIndex,
  renderUrlset,
} from '@/server/sitemap';

const SITE = 'https://ermekten.com';

describe('sitemap', () => {
  it('indexes one sitemap per content type', () => {
    const xml = renderIndex(SITE);
    expect(xml).toContain('<loc>https://ermekten.com/sitemaps/pages.xml</loc>');
    expect(xml).toContain('<loc>https://ermekten.com/sitemaps/outings.xml</loc>');
  });

  it('lists language-independent pages in every language with hreflang', () => {
    const xml = renderUrlset(SITE, [{ path: '/hotels/ak-say', lastmod: '2026-10-01T10:00:00Z' }]);
    expect(xml.match(/<url>/g)).toHaveLength(4);
    expect(xml).toContain('<loc>https://ermekten.com/en/hotels/ak-say</loc>');
    expect(xml).toContain('<lastmod>2026-10-01T10:00:00.000Z</lastmod>');
    expect(xml).toContain('hreflang="ky" href="https://ermekten.com/ky/hotels/ak-say"');
    expect(xml).toContain('hreflang="x-default" href="https://ermekten.com/hotels/ak-say"');
  });

  it('lists per-language content once, under its own language', () => {
    const xml = renderUrlset(SITE, [{ path: '/blog/hello & bye', lang: 'de', lastmod: null }]);
    expect(xml.match(/<url>/g)).toHaveLength(1);
    expect(xml).toContain('<loc>https://ermekten.com/de/blog/hello &amp; bye</loc>');
    expect(xml).not.toContain('hreflang');
    expect(xml).not.toContain('lastmod');
  });

  it('pages through hotels and asks for tours in each language', async () => {
    const calls: string[] = [];
    const getJson: GetJson = async (path) => {
      calls.push(path);
      if (path.startsWith('/hotels')) {
        const offset = Number(new URLSearchParams(path.split('?')[1]).get('offset'));
        const items = Array.from({ length: offset === 0 ? 50 : 1 }, (_, i) => ({
          slug: `h${offset + i}`,
          updatedAt: '2026-10-01T00:00:00Z',
        }));
        return { items, total: 51 };
      }
      return [{ slug: `tour-${path.split('=')[1]}`, publishedAt: null }];
    };

    expect(await collectEntries('hotels', getJson)).toHaveLength(51);
    expect(calls).toEqual([
      '/hotels?limit=50&offset=0&sort=newest',
      '/hotels?limit=50&offset=50&sort=newest',
    ]);
    const tours = await collectEntries('tours', getJson);
    expect(tours).toContainEqual({ path: '/tours/tour-ky', lastmod: null, lang: 'ky' });
    expect(tours).toHaveLength(4);
  });

  it('caches rendered sitemaps but never a failure', async () => {
    let now = 0;
    let calls = 0;
    let fail = true;
    const getJson: GetJson = async () => {
      calls++;
      if (fail) throw new Error('API down');
      return [{ slug: 'ala-kul', updatedAt: '2026-10-01T00:00:00Z' }];
    };
    const cache = new SitemapCache(SITE, getJson, 1000, () => now);

    await expect(cache.get('trails')).rejects.toThrow('API down');
    fail = false;
    expect(await cache.get('trails')).toContain('/trails/ala-kul');
    await cache.get('trails');
    expect(calls).toBe(2);
    now = 2000;
    await cache.get('trails');
    expect(calls).toBe(3);
  });
});
