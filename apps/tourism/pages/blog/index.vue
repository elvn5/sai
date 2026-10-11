<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import { useApiClient } from '~/shared/api/client';
import { type BlogPost, SeoPage } from '~/shared/api/types';

const { locale, t } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const auth = useAuth();
const config = useRuntimeConfig();
const apiBaseUrl = config.public.apiBaseUrl;

const postsQuery = useQuery({
  queryKey: ['blog', locale],
  queryFn: () => api.get<BlogPost[]>(`/blog?locale=${locale.value}`),
});
await postsQuery.suspense();

const posts = postsQuery.data;

const pageSeoQuery = usePageSeo(SeoPage.BLOG, locale);
await pageSeoQuery.suspense();
const pageSeo = pageSeoQuery.data;

useSeoMeta({
  title: () => pageSeo.value?.title || t('blog.heading'),
  description: () => pageSeo.value?.description || t('blog.metaDescription'),
  ogTitle: () => pageSeo.value?.title || t('blog.heading'),
  ogDescription: () => pageSeo.value?.description || t('blog.metaDescription'),
  ogImage: () =>
    pageSeo.value?.ogImageUrl ? `${apiBaseUrl}${pageSeo.value.ogImageUrl}` : undefined,
});
</script>

<template>
  <section class="mx-auto max-w-4xl px-4 py-12">
    <div class="mb-8 flex items-center justify-between gap-4">
      <h1 class="text-3xl font-extrabold text-text">{{ $t('blog.heading') }}</h1>
      <UButton v-if="auth.isPartner.value" color="primary" icon="i-heroicons-pencil-square" :to="localePath('/blog/new')">
        {{ $t('blog.form.newTitle') }}
      </UButton>
    </div>

    <div v-if="posts && posts.length > 0" class="flex flex-col gap-4">
      <UCard v-for="post in posts" :key="post.id">
        <NuxtLink :to="localePath(`/blog/${post.slug}`)" class="flex flex-col gap-4 sm:flex-row">
          <img
            v-if="post.coverImageUrl"
            :src="`${apiBaseUrl}${post.coverImageUrl}`"
            :alt="post.title"
            class="h-40 w-full rounded-lg object-cover sm:w-56"
          />
          <div class="flex flex-col justify-center">
            <h2 class="text-xl font-semibold text-text">{{ post.title }}</h2>
            <p v-if="post.excerpt" class="mt-2 text-text-muted">{{ post.excerpt }}</p>
          </div>
        </NuxtLink>
      </UCard>
    </div>
    <p v-else class="text-text-muted">{{ $t('blog.empty') }}</p>
  </section>
</template>
