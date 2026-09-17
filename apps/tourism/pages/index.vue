<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import AppLogo from '~/components/shared/ui/AppLogo.vue';
import { useApiClient } from '~/shared/api/client';
import { type BlogPost, SeoPage } from '~/shared/api/types';

const NEWS_LIMIT = 3;

const { locale, t } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const config = useRuntimeConfig();
const apiBaseUrl = config.public.apiBaseUrl;

const postsQuery = useQuery({
  queryKey: ['blog', locale],
  queryFn: () => api.get<BlogPost[]>(`/blog?locale=${locale.value}`),
});
await postsQuery.suspense();

const latestPosts = computed(() => (postsQuery.data.value ?? []).slice(0, NEWS_LIMIT));

const pageSeoQuery = usePageSeo(SeoPage.HOME);
await pageSeoQuery.suspense();
const pageSeo = pageSeoQuery.data;

useSeoMeta({
  title: () => pageSeo.value?.title || undefined,
  description: () => pageSeo.value?.description || t('home.tagline'),
  ogTitle: () => pageSeo.value?.title || t('brand.name'),
  ogDescription: () => pageSeo.value?.description || t('home.tagline'),
  ogImage: () =>
    pageSeo.value?.ogImageUrl
      ? `${apiBaseUrl}${pageSeo.value.ogImageUrl}`
      : '/images/hero-mountain.webp',
});
</script>

<template>
  <div>
    <section class="relative isolate overflow-hidden px-4 py-24 text-center sm:py-32">
      <div
        class="absolute inset-0 -z-20 bg-cover bg-center bg-no-repeat"
        style="background-image: url('/images/hero-mountain.webp')"
      />
      <div class="absolute inset-0 -z-10 bg-gradient-to-b from-black/70 via-black/50 to-bg" />

      <div class="mx-auto max-w-4xl">
        <AppLogo class="mx-auto h-16 w-16 text-white" />
        <h1 class="mt-4 text-3xl font-extrabold text-white sm:text-4xl">{{ $t('brand.name') }}</h1>
        <p class="mx-auto mt-3 max-w-xl text-white/85">{{ $t('home.tagline') }}</p>
        <UButton class="mt-6" color="primary" size="lg" :to="localePath('/guides')">
          {{ $t('home.ctaPrimary') }}
        </UButton>
      </div>
    </section>

    <section class="mx-auto max-w-4xl px-4 pb-16">
      <div class="mb-6 flex items-center justify-between">
        <h2 class="text-xl font-bold text-text">{{ $t('home.latestNews') }}</h2>
        <NuxtLink
          :to="localePath('/blog')"
          class="text-sm text-primary-600 hover:underline dark:text-primary-400"
        >
          {{ $t('home.viewAllPosts') }}
        </NuxtLink>
      </div>

      <div v-if="latestPosts.length > 0" class="flex flex-col gap-4">
        <UCard v-for="post in latestPosts" :key="post.id">
          <NuxtLink :to="localePath(`/blog/${post.slug}`)" class="flex flex-col gap-4 sm:flex-row">
            <img
              v-if="post.coverImageUrl"
              :src="`${apiBaseUrl}${post.coverImageUrl}`"
              :alt="post.title"
              class="h-40 w-full rounded-lg object-cover sm:w-56"
            />
            <div class="flex flex-col justify-center">
              <h3 class="text-lg font-semibold text-text">{{ post.title }}</h3>
              <p v-if="post.excerpt" class="mt-2 text-text-muted">{{ post.excerpt }}</p>
            </div>
          </NuxtLink>
        </UCard>
      </div>
      <p v-else class="text-text-muted">{{ $t('blog.empty') }}</p>
    </section>
  </div>
</template>
