<script setup lang="ts">
import CookieConsent from '~/components/widgets/cookie-consent/CookieConsent.vue';

const { t } = useI18n();
const head = useLocaleHead();
const siteConfig = useSiteConfig();

useHead({
  htmlAttrs: computed(() => head.value.htmlAttrs),
  link: computed(() => head.value.link),
  meta: computed(() => head.value.meta),
  titleTemplate: (title) =>
    title ? `${title} — ${t('brand.name')}` : `${t('brand.name')} — ${t('home.tagline')}`,
  script: [
    {
      type: 'application/ld+json',
      innerHTML: () =>
        JSON.stringify({
          '@context': 'https://schema.org',
          '@type': 'Organization',
          name: t('brand.name'),
          url: siteConfig.url,
          logo: `${siteConfig.url}/logo.svg`,
        }),
    },
  ],
});

useSeoMeta({
  ogSiteName: () => t('brand.name'),
  twitterCard: 'summary_large_image',
  themeColor: '#991b1b',
});
</script>

<template>
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
  <CookieConsent />
</template>
