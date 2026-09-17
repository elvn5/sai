<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import { useApiClient } from '~/shared/api/client';
import type { BlogPost } from '~/shared/api/types';

const route = useRoute();
const { locale } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const config = useRuntimeConfig();
const apiBaseUrl = config.public.apiBaseUrl;

const slug = route.params.slug as string;

const postQuery = useQuery({
  queryKey: ['blog', locale, slug],
  queryFn: async () => {
    try {
      return await api.get<BlogPost>(`/blog/${locale.value}/${slug}`);
    } catch {
      return null;
    }
  },
});
// Must await on the client too, not just SSR — otherwise a client-side
// navigation here hits the 404 check below before the query has resolved.
await postQuery.suspense();

const post = postQuery.data;

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

useHead({
  script: [
    {
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'BlogPosting',
        headline: post.value.title,
        description: post.value.seoDescription || post.value.excerpt || undefined,
        image: post.value.coverImageUrl ? `${apiBaseUrl}${post.value.coverImageUrl}` : undefined,
        datePublished: post.value.publishedAt ?? undefined,
        dateModified: post.value.updatedAt,
        inLanguage: locale.value,
      }),
    },
  ],
});
</script>

<template>
  <article v-if="post" class="mx-auto max-w-3xl px-4 py-12">
    <NuxtLink
      :to="localePath('/blog')"
      class="text-sm text-primary-600 hover:underline dark:text-primary-400"
    >
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
