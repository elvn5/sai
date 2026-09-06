export default defineNuxtConfig({
  compatibilityDate: '2025-01-01',
  devtools: { enabled: true },
  modules: ['@nuxt/ui', '@pinia/nuxt', '@vueuse/nuxt'],
  css: ['~/assets/css/main.css'],
  colorMode: {
    preference: 'system', // по умолчанию следуем системе; переключатель ставит явное значение
    fallback: 'light',
  },
  runtimeConfig: {
    public: {
      apiBaseUrl: process.env.NUXT_PUBLIC_API_BASE_URL ?? 'http://localhost:3001',
      // Версия задеплоенного образа для футера (задаётся в рантайме через NUXT_PUBLIC_APP_VERSION).
      appVersion: process.env.NUXT_PUBLIC_APP_VERSION ?? '',
    },
  },
  typescript: {
    strict: true,
  },
});
