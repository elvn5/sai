export default defineSitemapEventHandler(async () => {
  const apiBase = process.env.NUXT_PUBLIC_API_BASE_URL ?? 'http://localhost:3001';
  const locales = ['ru', 'ky', 'en', 'de'];

  const results = await Promise.all(
    locales.map((locale) =>
      $fetch<Array<{ locale: string; slug: string; updatedAt: string }>>(`${apiBase}/blog`, {
        query: { locale },
      }).catch(() => []),
    ),
  );

  return results.flat().map((post) => ({
    loc: post.locale === 'ru' ? `/blog/${post.slug}` : `/${post.locale}/blog/${post.slug}`,
    lastmod: post.updatedAt,
  }));
});
