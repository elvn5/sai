<script setup lang="ts">
import { useApiClient } from '~/shared/api/client';
import type { BlogPost } from '~/shared/api/types';

const route = useRoute();
const { locale } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const config = useRuntimeConfig();
const apiBaseUrl = config.public.apiBaseUrl;

const slug = route.params.slug as string;

const { data: post } = await useAsyncData<BlogPost | null>(
  `blog-post-${locale.value}-${slug}`,
  async () => {
    try {
      return await api.get<BlogPost>(`/blog/${locale.value}/${slug}`);
    } catch {
      return null;
    }
  },
);

if (!post.value) {
  throw createError({ statusCode: 404, statusMessage: 'Post not found' });
}

useSeoMeta({
  title: post.value.seoTitle || post.value.title,
  description: post.value.seoDescription || post.value.excerpt || undefined,
  ogTitle: post.value.seoTitle || post.value.title,
  ogDescription: post.value.seoDescription || post.value.excerpt || undefined,
  ogImage: post.value.coverImageUrl ? `${apiBaseUrl}${post.value.coverImageUrl}` : undefined,
});
</script>

<template>
  <article v-if="post" class="mx-auto max-w-3xl px-4 py-12">
    <NuxtLink :to="localePath('/blog')" class="text-sm text-primary hover:underline">
      &larr; {{ $t('blog.heading') }}
    </NuxtLink>

    <h1 class="mt-4 text-3xl font-extrabold text-text sm:text-4xl">{{ post.title }}</h1>

    <img
      v-if="post.coverImageUrl"
      :src="`${apiBaseUrl}${post.coverImageUrl}`"
      :alt="post.title"
      class="mt-6 w-full rounded-lg object-cover"
    />

    <div class="prose prose-neutral dark:prose-invert mt-8 max-w-none" v-html="post.content" />
  </article>
</template>
