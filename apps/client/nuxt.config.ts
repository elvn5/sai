export default defineNuxtConfig({
  compatibilityDate: '2025-01-01',
  devtools: { enabled: true },
  modules: [
    '@nuxt/ui',
    '@pinia/nuxt',
    '@vueuse/nuxt',
    '@nuxtjs/i18n',
    '@nuxtjs/sitemap',
    '@nuxtjs/robots',
  ],
  css: [],
  colorMode: {
    preference: 'system',
    fallback: 'light',
  },
  site: {
    url: process.env.NUXT_PUBLIC_SITE_URL ?? 'http://localhost:3002',
    name: 'ClientApp',
    defaultLocale: 'ru',
  },
  robots: {
    disallow: ['/login', '/ky/login', '/en/login', '/de/login'],
  },
  sitemap: {
    sources: ['/api/__sitemap__/urls'],
  },
  i18n: {
    locales: [
      { code: 'ru', language: 'ru-RU', name: 'Русский', file: 'ru.json' },
      { code: 'ky', language: 'ky-KG', name: 'Кыргызча', file: 'ky.json' },
      { code: 'en', language: 'en-US', name: 'English', file: 'en.json' },
      { code: 'de', language: 'de-DE', name: 'Deutsch', file: 'de.json' },
    ],
    defaultLocale: 'ru',
    strategy: 'prefix_except_default',
    lazy: true,
    baseUrl: process.env.NUXT_PUBLIC_SITE_URL ?? 'http://localhost:3002',
    detectBrowserLanguage: {
      useCookie: true,
      cookieKey: 'i18n_redirected',
      redirectOn: 'root',
    },
  },
  runtimeConfig: {
    public: {
      apiBaseUrl: process.env.NUXT_PUBLIC_API_BASE_URL ?? 'http://localhost:3001',
    },
  },
  typescript: {
    strict: true,
  },
});
