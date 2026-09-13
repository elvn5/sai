<script setup lang="ts">
import AppBadge from '~/components/shared/ui/AppBadge.vue';
import AppCard from '~/components/shared/ui/AppCard.vue';
import { useApiClient } from '~/shared/api/client';
import type { Guide } from '~/shared/api/types';

const { locale, t } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const config = useRuntimeConfig();
const apiBaseUrl = config.public.apiBaseUrl;

const { data: guides } = await useAsyncData<Guide[]>(
  () => `guides-list-${locale.value}`,
  () => api.get<Guide[]>(`/guides?locale=${locale.value}`),
  { watch: [locale] },
);

function difficultyLabel(difficulty: string): string {
  return t(`guides.difficulty.${difficulty}`);
}
</script>

<template>
  <section class="mx-auto max-w-4xl px-4 py-12">
    <h1 class="mb-8 text-3xl font-extrabold text-text">{{ $t('guides.heading') }}</h1>

    <div v-if="guides && guides.length > 0" class="flex flex-col gap-4">
      <AppCard v-for="guide in guides" :key="guide.id" as="article">
        <NuxtLink :to="localePath(`/guides/${guide.slug}`)" class="flex flex-col gap-4 sm:flex-row">
          <img
            v-if="guide.coverImageUrl"
            :src="`${apiBaseUrl}${guide.coverImageUrl}`"
            :alt="guide.title"
            class="h-40 w-full rounded-lg object-cover sm:w-56"
          />
          <div class="flex flex-1 flex-col justify-center">
            <div class="mb-2 flex flex-wrap gap-2">
              <AppBadge tone="primary">{{ difficultyLabel(guide.difficulty) }}</AppBadge>
              <AppBadge tone="secondary">{{ $t(`guides.season.${guide.season}`) }}</AppBadge>
            </div>
            <h2 class="text-xl font-semibold text-text">{{ guide.title }}</h2>
            <p v-if="guide.excerpt" class="mt-2 text-text-muted">{{ guide.excerpt }}</p>
          </div>
        </NuxtLink>
      </AppCard>
    </div>
    <p v-else class="text-text-muted">{{ $t('guides.empty') }}</p>
  </section>
</template>
