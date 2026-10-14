const siteUrl = process.env.NUXT_PUBLIC_SITE_URL ?? 'http://localhost:3011';
// Only the canonical production domain should ever be indexable — staging
// serves the same build and would otherwise create duplicate-content noise.
const isProductionSite = siteUrl === 'https://ermekten.com';

export default defineNuxtConfig({
  compatibilityDate: '2025-01-01',
  devtools: { enabled: true },
  devServer: { port: 3011 },
  features: { inlineStyles: false },
  modules: ['@nuxt/ui', '@nuxtjs/i18n', '@nuxtjs/robots'],
  site: {
    url: siteUrl,
    name: 'Ermekten',
  },
  robots: {
    sitemap: '/sitemap.xml',
    // Staging serves the same build as production and must stay out of search
    // results entirely, so it gets a blanket disallow instead of the
    // page-level rules below.
    disallow: isProductionSite
      ? [
          '/login',
          '/profile',
          '/partner',
          '/partner/*',
          '/guides/new',
          '/tours/new',
          '/hotels/new',
          '/hotels/*/edit',
          '/local-guides/new',
          '/local-guides/*/edit',
        ]
      : ['/'],
  },
  css: ['~/assets/css/tokens.css'],
  ui: {
    safelistColors: ['secondary'],
  },
  colorMode: {
    preference: 'system',
    fallback: 'light',
  },
  imports: {
    dirs: ['shared/lib'],
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
    baseUrl: siteUrl,
    detectBrowserLanguage: {
      useCookie: true,
      cookieKey: 'i18n_redirected',
      redirectOn: 'root',
    },
  },
  app: {
    head: {
      link: [{ id: 'favicon', rel: 'icon', type: 'image/svg+xml', href: '/logo.svg' }],
      script: [
        { src: 'https://telegram.org/js/telegram-web-app.js', defer: true },
        {
          innerHTML:
            "try{if(localStorage.getItem('color-theme')==='azure'){document.documentElement.setAttribute('data-color-theme','azure');var f=document.getElementById('favicon');if(f)f.href='/logo-azure.svg'}}catch(e){}",
        },
      ],
    },
  },
  runtimeConfig: {
    public: {
      apiBaseUrl: process.env.NUXT_PUBLIC_API_BASE_URL ?? 'http://localhost:3001',
      telegramBotUsername: process.env.NUXT_PUBLIC_TELEGRAM_BOT_USERNAME ?? '',
      siteUrl,
    },
  },
  typescript: {
    strict: true,
  },
});
