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
    expect(xml).toContain('<loc>https://ermekten.com/sitemaps/tours.xml</loc>');
    expect(xml).toContain('<loc>https://ermekten.com/sitemaps/organizers.xml</loc>');
    expect(xml).not.toContain('hotels.xml');
  });

  it('lists language-independent pages in every language with hreflang', () => {
    const xml = renderUrlset(SITE, [{ path: '/hikes', lastmod: '2026-10-01T10:00:00Z' }]);
    expect(xml.match(/<url>/g)).toHaveLength(4);
    expect(xml).toContain('<loc>https://ermekten.com/en/hikes</loc>');
    expect(xml).toContain('<lastmod>2026-10-01T10:00:00.000Z</lastmod>');
    expect(xml).toContain('hreflang="ky" href="https://ermekten.com/ky/hikes"');
    expect(xml).toContain('hreflang="x-default" href="https://ermekten.com/hikes"');
  });

  it('lists per-language content once, under its own language', () => {
    const xml = renderUrlset(SITE, [{ path: '/blog/hello & bye', lang: 'de', lastmod: null }]);
    expect(xml.match(/<url>/g)).toHaveLength(1);
    expect(xml).toContain('<loc>https://ermekten.com/de/blog/hello &amp; bye</loc>');
    expect(xml).not.toContain('hreflang');
    expect(xml).not.toContain('lastmod');
  });

  it('asks for tours in each language', async () => {
    const getJson: GetJson = async (path) => [
      { slug: `tour-${path.split('=')[1]}`, publishedAt: null },
    ];
    const tours = await collectEntries('tours', getJson);
    expect(tours).toContainEqual({ path: '/tours/tour-ky', lastmod: null, lang: 'ky' });
    expect(tours).toHaveLength(4);
  });

  it('lists the organizers of tours once, with their latest tour', async () => {
    const getJson: GetJson = async (path) =>
      path.endsWith('=ru')
        ? [
            { slug: 'a', publishedAt: '2026-09-01T00:00:00Z', userId: 'u1' },
            { slug: 'b', publishedAt: null, userId: null },
          ]
        : [{ slug: 'c', publishedAt: '2026-10-01T00:00:00Z', userId: 'u1' }];
    expect(await collectEntries('organizers', getJson)).toEqual([
      { path: '/organizers/u1', lastmod: '2026-10-01T00:00:00Z' },
    ]);
  });

  it('caches rendered sitemaps but never a failure', async () => {
    let now = 0;
    let calls = 0;
    let fail = true;
    const getJson: GetJson = async () => {
      calls++;
      if (fail) throw new Error('API down');
      return [{ slug: 'ala-kul', publishedAt: null }];
    };
    const cache = new SitemapCache(SITE, getJson, 1000, () => now);

    await expect(cache.get('tours')).rejects.toThrow('API down');
    fail = false;
    expect(await cache.get('tours')).toContain('/tours/ala-kul');
    // One request per language each time the sitemap is built.
    await cache.get('tours');
    expect(calls).toBe(8);
    now = 2000;
    await cache.get('tours');
    expect(calls).toBe(12);
  });
});
