export default defineNuxtConfig({
  compatibilityDate: '2025-01-01',
  devtools: { enabled: true },
  modules: ['@nuxt/ui', '@vueuse/nuxt', '@nuxtjs/i18n'],
  css: ['~/assets/css/main.css'],
  colorMode: {
    preference: 'system', // по умолчанию следуем системе; переключатель ставит явное значение
    fallback: 'light',
  },
  i18n: {
    strategy: 'prefix_except_default',
    defaultLocale: 'ru',
    // Абсолютный базовый URL сайта — нужен для canonical/hreflang/sitemap.
    baseUrl: process.env.NUXT_PUBLIC_SITE_URL ?? 'https://example.com',
    locales: [
      { code: 'ru', language: 'ru-RU', name: 'Русский', file: 'ru.json' },
      { code: 'ky', language: 'ky-KG', name: 'Кыргызча', file: 'ky.json' },
      { code: 'en', language: 'en-US', name: 'English', file: 'en.json' },
    ],
    detectBrowserLanguage: {
      useCookie: true,
      cookieKey: 'i18n_locale',
      redirectOn: 'root',
      alwaysRedirect: false,
    },
  },
  runtimeConfig: {
    // Серверный (SSR) URL API — внутри Docker это http://api:3001.
    apiInternalUrl: process.env.NUXT_API_INTERNAL_URL ?? 'http://localhost:3001',
    public: {
      apiBaseUrl: process.env.NUXT_PUBLIC_API_BASE_URL ?? 'http://localhost:3001',
      // Версия задеплоенного образа для футера (задаётся в рантайме через NUXT_PUBLIC_APP_VERSION).
      appVersion: process.env.NUXT_PUBLIC_APP_VERSION ?? '',
      botUrl: process.env.NUXT_PUBLIC_BOT_URL ?? 'https://t.me/acme_bot',
      // Канонический адрес сайта (для SEO-тегов и sitemap).
      siteUrl: process.env.NUXT_PUBLIC_SITE_URL ?? 'https://example.com',
    },
  },
  typescript: {
    strict: true,
  },
});
