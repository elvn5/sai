<script setup lang="ts">
import AppLogo from '~/components/shared/ui/AppLogo.vue';
import { useApiClient } from '~/shared/api/client';
import type { BlogPost } from '~/shared/api/types';

const NEWS_LIMIT = 3;

const { locale } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const config = useRuntimeConfig();
const apiBaseUrl = config.public.apiBaseUrl;

const { data: posts } = await useAsyncData<BlogPost[]>(
  () => `home-news-${locale.value}`,
  () => api.get<BlogPost[]>(`/blog?locale=${locale.value}`),
  { watch: [locale] },
);

const latestPosts = computed(() => (posts.value ?? []).slice(0, NEWS_LIMIT));
</script>

<template>
  <div>
    <section class="mx-auto max-w-4xl px-4 py-16 text-center">
      <AppLogo class="mx-auto h-16 w-16 text-primary-600 dark:text-primary-400" />
      <h1 class="mt-4 text-3xl font-extrabold text-text sm:text-4xl">{{ $t('brand.name') }}</h1>
      <p class="mx-auto mt-3 max-w-xl text-text-muted">{{ $t('home.tagline') }}</p>
      <UButton class="mt-6" color="primary" size="lg" :to="localePath('/guides')">
        {{ $t('home.ctaPrimary') }}
      </UButton>
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
