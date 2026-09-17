<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import { useApiClient } from '~/shared/api/client';
import { type Guide, SeoPage } from '~/shared/api/types';

const { locale, t } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const auth = useAuth();
const config = useRuntimeConfig();
const apiBaseUrl = config.public.apiBaseUrl;

const guidesQuery = useQuery({
  queryKey: ['guides', locale],
  queryFn: () => api.get<Guide[]>(`/guides?locale=${locale.value}`),
});
await guidesQuery.suspense();

const guides = guidesQuery.data;

function difficultyLabel(difficulty: string): string {
  return t(`guides.difficulty.${difficulty}`);
}

const pageSeoQuery = usePageSeo(SeoPage.GUIDES, locale);
await pageSeoQuery.suspense();
const pageSeo = pageSeoQuery.data;

useSeoMeta({
  title: () => pageSeo.value?.title || t('guides.heading'),
  description: () => pageSeo.value?.description || t('guides.metaDescription'),
  ogTitle: () => pageSeo.value?.title || t('guides.heading'),
  ogDescription: () => pageSeo.value?.description || t('guides.metaDescription'),
  ogImage: () =>
    pageSeo.value?.ogImageUrl ? `${apiBaseUrl}${pageSeo.value.ogImageUrl}` : undefined,
});
</script>

<template>
  <section class="mx-auto max-w-4xl px-4 py-12">
    <div class="mb-8 flex items-center justify-between">
      <h1 class="text-3xl font-extrabold text-text">{{ $t('guides.heading') }}</h1>
      <UButton
        v-if="auth.user.value"
        color="primary"
        icon="i-heroicons-plus"
        :to="localePath('/guides/new')"
      >
        {{ $t('guides.addButton') }}
      </UButton>
    </div>

    <div v-if="guides && guides.length > 0" class="flex flex-col gap-4">
      <UCard v-for="guide in guides" :key="guide.id">
        <NuxtLink :to="localePath(`/guides/${guide.slug}`)" class="flex flex-col gap-4 sm:flex-row">
          <img
            v-if="guide.coverImageUrl"
            :src="`${apiBaseUrl}${guide.coverImageUrl}`"
            :alt="guide.title"
            class="h-40 w-full rounded-lg object-cover sm:w-56"
          />
          <div class="flex flex-1 flex-col justify-center">
            <div class="mb-2 flex flex-wrap gap-2">
              <UBadge color="primary" variant="subtle">{{ difficultyLabel(guide.difficulty) }}</UBadge>
              <UBadge color="secondary" variant="subtle">{{ $t(`guides.season.${guide.season}`) }}</UBadge>
            </div>
            <h2 class="text-xl font-semibold text-text">{{ guide.title }}</h2>
            <p v-if="guide.excerpt" class="mt-2 text-text-muted">{{ guide.excerpt }}</p>
          </div>
        </NuxtLink>
      </UCard>
    </div>
    <p v-else class="text-text-muted">{{ $t('guides.empty') }}</p>
  </section>
</template>
