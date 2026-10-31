import { type GetJson, SitemapCache, collectEntries, renderUrlset } from '@/server/sitemap';

const SITE = 'https://ermekten.com';

describe('sitemap', () => {
  it('lists every page in one sitemap: sections, blog posts and trails', async () => {
    const getJson: GetJson = async (path) => {
      if (path === '/trails') return [{ slug: 'ala-kul', updatedAt: '2026-10-28T10:00:00Z' }];
      if (path === '/blog?locale=ru') return [{ slug: 'first-hike', publishedAt: null }];
      if (path === '/tours?locale=ru') {
        return [{ slug: 'song-kol', publishedAt: null, userId: 'u1' }];
      }
      return [];
    };
    const xml = await new SitemapCache(SITE, getJson).get('all');
    expect(xml).toContain('<urlset');
    expect(xml).not.toContain('<sitemapindex');
    expect(xml).toContain('<loc>https://ermekten.com/blog</loc>');
    expect(xml).toContain('<loc>https://ermekten.com/blog/first-hike</loc>');
    expect(xml).toContain('<loc>https://ermekten.com/trails/ala-kul</loc>');
    expect(xml).toContain('<loc>https://ermekten.com/tours/song-kol</loc>');
    expect(xml).toContain('<loc>https://ermekten.com/organizers/u1</loc>');
    expect(xml).not.toContain('/hotels');
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

  it('links a blog post to its translations under the same slug', async () => {
    const getJson: GetJson = async (path) =>
      path.endsWith('=ru') || path.endsWith('=en')
        ? [{ slug: 'winter', publishedAt: null }]
        : path.endsWith('=de')
          ? [{ slug: 'nur-deutsch', publishedAt: null }]
          : [];
    const posts = await collectEntries('blog', getJson);
    expect(posts).toContainEqual({
      path: '/blog/winter',
      lastmod: null,
      lang: 'en',
      translations: ['ru', 'en'],
    });
    expect(posts).toContainEqual({ path: '/blog/nur-deutsch', lastmod: null, lang: 'de' });
    const xml = renderUrlset(SITE, posts);
    expect(xml.match(/<url>/g)).toHaveLength(3);
    expect(xml).toContain(
      '<url><loc>https://ermekten.com/en/blog/winter</loc><xhtml:link rel="alternate" hreflang="ru" href="https://ermekten.com/blog/winter"/><xhtml:link rel="alternate" hreflang="en" href="https://ermekten.com/en/blog/winter"/><xhtml:link rel="alternate" hreflang="x-default" href="https://ermekten.com/blog/winter"/></url>',
    );
    expect(xml).not.toContain('hreflang="ky"');
    expect(xml).toContain('<url><loc>https://ermekten.com/de/blog/nur-deutsch</loc></url>');
  });

  it('asks for tours in each language', async () => {
    const getJson: GetJson = async (path) => [
      { slug: `tour-${path.split('=')[1]}`, publishedAt: null },
    ];
    const tours = await collectEntries('tours', getJson);
    expect(tours).toContainEqual({ path: '/tours/tour-ky', lastmod: null, lang: 'ky' });
    expect(tours).toHaveLength(4);
  });

  it('includes published trails in Russian only and adds the catalog', async () => {
    const getJson = vi.fn(async () => [{ slug: 'own-route', updatedAt: '2026-10-24T10:00:00Z' }]);
    const trails = await collectEntries('trails', getJson);
    expect(getJson).toHaveBeenCalledWith('/trails');
    expect(trails).toEqual([
      { path: '/trails/own-route', lastmod: '2026-10-24T10:00:00Z', lang: 'ru' },
    ]);
    const xml = renderUrlset(SITE, trails);
    expect(xml).toContain('<loc>https://ermekten.com/trails/own-route</loc>');
    expect(xml).not.toContain('/ky/trails/own-route');
    expect(await collectEntries('pages', getJson)).toContainEqual({ path: '/trails' });
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
