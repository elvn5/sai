const LOCALES = ['ru', 'ky', 'en', 'de'] as const;
const DEFAULT_LOCALE = 'ru';

interface PublishedItem {
  slug: string;
  publishedAt: string | null;
}

function localizedPath(locale: string, path: string): string {
  return locale === DEFAULT_LOCALE ? path : `/${locale}${path}`;
}

export default defineSitemapEventHandler(async () => {
  const config = useRuntimeConfig();
  const apiBaseUrl = config.public.apiBaseUrl;

  const urls: { loc: string; lastmod?: string; priority?: 1 }[] = [];

  for (const locale of LOCALES) {
    urls.push({ loc: localizedPath(locale, '/'), priority: 1 });
    urls.push({ loc: localizedPath(locale, '/guides') });
    urls.push({ loc: localizedPath(locale, '/tours') });
    urls.push({ loc: localizedPath(locale, '/blog') });
    urls.push({ loc: localizedPath(locale, '/gallery') });

    const [blogPosts, guides, tours] = await Promise.all([
      $fetch<PublishedItem[]>(`${apiBaseUrl}/blog?locale=${locale}`).catch(() => []),
      $fetch<PublishedItem[]>(`${apiBaseUrl}/guides?locale=${locale}`).catch(() => []),
      $fetch<PublishedItem[]>(`${apiBaseUrl}/tours?locale=${locale}`).catch(() => []),
    ]);

    for (const post of blogPosts) {
      urls.push({
        loc: localizedPath(locale, `/blog/${post.slug}`),
        lastmod: post.publishedAt ?? undefined,
      });
    }
    for (const guide of guides) {
      urls.push({
        loc: localizedPath(locale, `/guides/${guide.slug}`),
        lastmod: guide.publishedAt ?? undefined,
      });
    }
    for (const tour of tours) {
      urls.push({
        loc: localizedPath(locale, `/tours/${tour.slug}`),
        lastmod: tour.publishedAt ?? undefined,
      });
    }
  }

  return urls;
});
