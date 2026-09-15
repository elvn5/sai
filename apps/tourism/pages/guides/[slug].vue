<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { useApiClient } from '~/shared/api/client';
import type { Guide } from '~/shared/api/types';

const route = useRoute();
const { locale, t } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const auth = useAuth();
const queryClient = useQueryClient();
const config = useRuntimeConfig();
const apiBaseUrl = config.public.apiBaseUrl;

const slug = route.params.slug as string;

const guideQuery = useQuery({
  queryKey: ['guides', locale, slug],
  queryFn: async () => {
    try {
      return await api.get<Guide>(`/guides/${locale.value}/${slug}`);
    } catch {
      return null;
    }
  },
});
// Must await on the client too, not just SSR — otherwise a client-side
// navigation here (e.g. right after creating a guide) hits the 404 check
// below before the query has resolved.
await guideQuery.suspense();

const guide = guideQuery.data;

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

const deleteMutation = useMutation({
  mutationFn: (id: string) => api.delete(`/guides/${id}`),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ['guides'] });
    navigateTo(localePath('/guides'));
  },
});

function handleDelete(): void {
  if (!guide.value) return;
  if (!confirm(t('guides.deleteConfirm'))) return;
  deleteMutation.mutate(guide.value.id);
}
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

    <UButton
      v-if="auth.user.value && guide.userId === auth.user.value.id"
      color="gray"
      variant="soft"
      class="mt-8"
      :loading="deleteMutation.isPending.value"
      @click="handleDelete"
    >
      {{ $t('guides.deleteButton') }}
    </UButton>
  </article>
</template>
