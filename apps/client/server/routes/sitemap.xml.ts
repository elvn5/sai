/**
 * Динамический sitemap.xml: статические страницы во всех локалях + опубликованные
 * посты блога (тянутся из API по внутреннему URL). Отдаётся Nitro-сервером Nuxt,
 * доступен по https://example.com/sitemap.xml (проксируется nginx на клиент).
 */
interface PostSitemapItem {
  slug: string;
  publishedAt: string | null;
}

const LOCALES = [
  { code: 'ru', prefix: '' },
  { code: 'ky', prefix: '/ky' },
  { code: 'en', prefix: '/en' },
];

// Пути относительно префикса локали. '' -> корень.
const STATIC_PATHS = ['', '/blog'];

function xmlEscape(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&apos;');
}

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig(event);
  const siteUrl = String(config.public.siteUrl ?? 'https://example.com').replace(/\/$/, '');
  const apiBase = String(config.apiInternalUrl ?? 'http://localhost:3001').replace(/\/$/, '');

  const urls: Array<{ loc: string; lastmod?: string }> = [];

  for (const locale of LOCALES) {
    for (const path of STATIC_PATHS) {
      urls.push({ loc: `${siteUrl}${locale.prefix}${path || '/'}` });
    }
  }

  // Посты по каждому языку. Ошибки API не должны ронять sitemap.
  await Promise.all(
    LOCALES.map(async (locale) => {
      try {
        const posts = await $fetch<PostSitemapItem[]>(`${apiBase}/posts`, {
          query: { language: locale.code },
        });
        for (const post of posts ?? []) {
          urls.push({
            loc: `${siteUrl}${locale.prefix}/blog/${post.slug}`,
            lastmod: post.publishedAt ? new Date(post.publishedAt).toISOString() : undefined,
          });
        }
      } catch {
        // API недоступен — отдаём хотя бы статические URL.
      }
    }),
  );

  const body =
    `<?xml version="1.0" encoding="UTF-8"?>\n` +
    `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n` +
    urls
      .map(
        (u) =>
          `  <url><loc>${xmlEscape(u.loc)}</loc>` +
          (u.lastmod ? `<lastmod>${u.lastmod}</lastmod>` : '') +
          `</url>`,
      )
      .join('\n') +
    `\n</urlset>\n`;

  setHeader(event, 'content-type', 'application/xml; charset=utf-8');
  return body;
});
