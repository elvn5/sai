<script setup lang="ts">
import { useApiClient } from '~/shared/api/client';
import type { Guide } from '~/shared/api/types';

const route = useRoute();
const { locale, t } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const config = useRuntimeConfig();
const apiBaseUrl = config.public.apiBaseUrl;

const slug = route.params.slug as string;

const { data: guide } = await useAsyncData<Guide | null>(
  `guide-${locale.value}-${slug}`,
  async () => {
    try {
      return await api.get<Guide>(`/guides/${locale.value}/${slug}`);
    } catch {
      return null;
    }
  },
);

if (!guide.value) {
  throw createError({ statusCode: 404, statusMessage: 'Guide not found' });
}

useSeoMeta({
  title: guide.value.seoTitle || guide.value.title,
  description: guide.value.seoDescription || guide.value.excerpt || undefined,
  ogTitle: guide.value.seoTitle || guide.value.title,
  ogDescription: guide.value.seoDescription || guide.value.excerpt || undefined,
  ogImage: guide.value.coverImageUrl ? `${apiBaseUrl}${guide.value.coverImageUrl}` : undefined,
});
</script>

<template>
  <article v-if="guide" class="mx-auto max-w-3xl px-4 py-12">
    <NuxtLink
      :to="localePath('/guides')"
      class="text-sm text-primary-600 hover:underline dark:text-primary-400"
    >
      &larr; {{ $t('guides.heading') }}
    </NuxtLink>

    <div class="mt-4 flex flex-wrap gap-2">
      <UBadge color="primary" variant="subtle">{{ t(`guides.difficulty.${guide.difficulty}`) }}</UBadge>
      <UBadge color="secondary" variant="subtle">{{ t(`guides.season.${guide.season}`) }}</UBadge>
    </div>

    <h1 class="mt-4 text-3xl font-extrabold text-text sm:text-4xl">{{ guide.title }}</h1>

    <img
      v-if="guide.coverImageUrl"
      :src="`${apiBaseUrl}${guide.coverImageUrl}`"
      :alt="guide.title"
      class="mt-6 w-full rounded-lg object-cover"
    />

    <div
      v-if="guide.equipmentChecklist.length > 0"
      class="mt-8 rounded-card border border-border bg-surface p-4"
    >
      <h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-text-muted">
        {{ $t('guides.equipment') }}
      </h2>
      <ul class="flex flex-col gap-2">
        <li v-for="entry in guide.equipmentChecklist" :key="entry" class="flex items-center gap-2 text-text">
          <span class="h-1.5 w-1.5 rounded-full bg-primary-600 dark:bg-primary-400" />
          {{ entry }}
        </li>
      </ul>
    </div>

    <div class="prose prose-neutral dark:prose-invert mt-8 max-w-none" v-html="guide.content" />
  </article>
</template>
