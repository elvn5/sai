<template>
  <div class="bg-gray-50 dark:bg-gray-950 min-h-screen">
    <!-- Hero Section -->
    <div class="relative bg-white dark:bg-gray-900 overflow-hidden">
      <div class="max-w-7xl mx-auto">
        <div class="relative z-10 pb-8 bg-white dark:bg-gray-900 sm:pb-16 md:pb-20 lg:max-w-2xl lg:w-full lg:pb-28 xl:pb-32">
          <main class="mt-10 mx-auto max-w-7xl px-4 sm:mt-12 sm:px-6 md:mt-16 lg:mt-20 lg:px-8 xl:mt-28">
            <div class="sm:text-center lg:text-left">
              <h1 class="text-4xl tracking-tight font-extrabold text-gray-900 dark:text-white sm:text-5xl md:text-6xl">
                <span class="block xl:inline">{{ $t('home.hero.titleLine1') }}</span>
                <span class="block text-primary-600 dark:text-primary-400 xl:inline"> {{ $t('home.hero.titleLine2') }}</span>
              </h1>
              <p class="mt-3 text-base text-gray-500 dark:text-gray-400 sm:mt-5 sm:text-lg sm:max-w-xl sm:mx-auto md:mt-5 md:text-xl lg:mx-0">
                {{ $t('home.hero.subtitle') }}
              </p>
              <div class="mt-5 sm:mt-8 sm:flex sm:justify-center lg:justify-start">
                <div class="rounded-md shadow">
                  <UButton size="xl" color="primary">{{ $t('home.hero.ctaPrimary') }}</UButton>
                </div>
                <div class="mt-3 sm:mt-0 sm:ml-3">
                  <UButton size="xl" color="white" variant="solid" class="dark:bg-gray-800 dark:text-white dark:hover:bg-gray-700">{{ $t('home.hero.ctaSecondary') }}</UButton>
                </div>
              </div>
            </div>
          </main>
        </div>
      </div>
    </div>

    <!-- Feature Section -->
    <div class="py-12 bg-gray-50 dark:bg-gray-950">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="lg:text-center">
          <h2 class="text-base text-primary-600 dark:text-primary-400 font-semibold tracking-wide uppercase">{{ $t('home.features.eyebrow') }}</h2>
          <p class="mt-2 text-3xl leading-8 font-extrabold tracking-tight text-gray-900 dark:text-white sm:text-4xl">
            {{ $t('home.features.heading') }}
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useApiClient } from '~/shared/api/client';
import type { PageSeo } from '~/shared/api/types';

const { locale, t } = useI18n();
const api = useApiClient();
const config = useRuntimeConfig();

const { data: seo } = await useAsyncData<PageSeo | null>(
  () => `page-seo-home-${locale.value}`,
  () => api.get<PageSeo | null>(`/page-seo/home/${locale.value}`).catch(() => null),
  { watch: [locale] },
);

useSeoMeta({
  title: () => seo.value?.title || t('home.hero.titleLine1'),
  description: () => seo.value?.description || t('home.hero.subtitle'),
  ogTitle: () => seo.value?.title || t('home.hero.titleLine1'),
  ogDescription: () => seo.value?.description || t('home.hero.subtitle'),
  ogImage: () =>
    seo.value?.ogImageUrl ? `${config.public.apiBaseUrl}${seo.value.ogImageUrl}` : undefined,
});
</script>
