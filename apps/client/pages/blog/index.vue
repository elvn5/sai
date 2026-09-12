<template>
  <div class="mx-auto min-h-screen max-w-4xl px-4 py-12 sm:px-6 lg:px-8">
    <h1 class="mb-8 text-3xl font-extrabold text-gray-900 dark:text-white">
      {{ $t('blog.heading') }}
    </h1>

    <div v-if="posts && posts.length > 0" class="flex flex-col gap-8">
      <NuxtLink
        v-for="post in posts"
        :key="post.id"
        :to="localePath(`/blog/${post.slug}`)"
        class="flex flex-col gap-4 rounded-lg border border-gray-200 p-4 transition hover:border-primary-400 dark:border-gray-800 sm:flex-row"
      >
        <img
          v-if="post.coverImageUrl"
          :src="`${apiBaseUrl}${post.coverImageUrl}`"
          :alt="post.title"
          class="h-40 w-full rounded-md object-cover sm:w-56"
        />
        <div class="flex flex-col justify-center">
          <h2 class="text-xl font-semibold text-gray-900 dark:text-white">{{ post.title }}</h2>
          <p v-if="post.excerpt" class="mt-2 text-gray-500 dark:text-gray-400">{{ post.excerpt }}</p>
        </div>
      </NuxtLink>
    </div>
    <p v-else class="text-gray-500 dark:text-gray-400">{{ $t('blog.empty') }}</p>
  </div>
</template>

<script setup lang="ts">
import { useApiClient } from '~/shared/api/client';
import type { BlogPost } from '~/shared/api/types';

const { locale } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const config = useRuntimeConfig();
const apiBaseUrl = config.public.apiBaseUrl;

const { data: posts } = await useAsyncData<BlogPost[]>(
  () => `blog-list-${locale.value}`,
  () => api.get<BlogPost[]>(`/blog?locale=${locale.value}`),
  { watch: [locale] },
);
</script>
