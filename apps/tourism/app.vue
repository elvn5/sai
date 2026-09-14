<script setup lang="ts">
import CookieConsent from '~/components/widgets/cookie-consent/CookieConsent.vue';

const { t } = useI18n();
const head = useLocaleHead();

useHead({
  htmlAttrs: computed(() => head.value.htmlAttrs),
  link: computed(() => head.value.link),
  meta: computed(() => head.value.meta),
  titleTemplate: (title) =>
    title ? `${title} — ${t('brand.name')}` : `${t('brand.name')} — ${t('home.tagline')}`,
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
