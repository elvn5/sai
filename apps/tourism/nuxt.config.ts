export default defineNuxtConfig({
  compatibilityDate: '2025-01-01',
  devtools: { enabled: true },
  devServer: { port: 3011 },
  features: { inlineStyles: false },
  modules: ['@nuxt/ui', '@nuxtjs/i18n'],
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
    baseUrl: process.env.NUXT_PUBLIC_SITE_URL ?? 'http://localhost:3011',
    // The locale is already encoded in the URL path (prefix_except_default),
    // so a redirect-tracking cookie isn't needed — one less non-essential
    // cookie on the site.
    detectBrowserLanguage: {
      useCookie: false,
      redirectOn: 'root',
    },
  },
  app: {
    head: {
      link: [{ rel: 'icon', type: 'image/svg+xml', href: '/logo.svg' }],
      script: [
        { src: 'https://telegram.org/js/telegram-web-app.js', defer: true },
        {
          innerHTML:
            "try{if(localStorage.getItem('color-theme')==='azure'){document.documentElement.setAttribute('data-color-theme','azure')}}catch(e){}",
        },
      ],
    },
  },
  runtimeConfig: {
    public: {
      apiBaseUrl: process.env.NUXT_PUBLIC_API_BASE_URL ?? 'http://localhost:3001',
      telegramBotUsername: process.env.NUXT_PUBLIC_TELEGRAM_BOT_USERNAME ?? '',
    },
  },
  typescript: {
    strict: true,
  },
});
