/**
 * Возвращает набор <link rel="alternate" hreflang="…"> для текущего маршрута
 * во всех локалях + x-default. Использовать на страницах, которые существуют
 * во всех языках (главная, список блога). Для отдельного поста не применять —
 * перевод может отсутствовать.
 */
export function useSeoAlternates() {
  const { locales } = useI18n();
  const switchLocalePath = useSwitchLocalePath();
  const runtimeConfig = useRuntimeConfig();
  const siteUrl = String(runtimeConfig.public.siteUrl ?? 'https://example.com').replace(/\/$/, '');

  return computed(() => {
    const list = locales.value as Array<{ code: string; language: string }>;
    const links = list
      .map((l) => {
        const path = switchLocalePath(l.code);
        return path
          ? { rel: 'alternate', hreflang: l.language, href: siteUrl + path }
          : null;
      })
      .filter((v): v is { rel: string; hreflang: string; href: string } => v !== null);

    const defaultPath = switchLocalePath('ru') || '/';
    links.push({ rel: 'alternate', hreflang: 'x-default', href: siteUrl + defaultPath });
    return links;
  });
}
