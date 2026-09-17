<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { useApiClient } from '~/shared/api/client';
import type { Tour } from '~/shared/api/types';

const route = useRoute();
const { locale, t } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const auth = useAuth();
const queryClient = useQueryClient();
const config = useRuntimeConfig();
const apiBaseUrl = config.public.apiBaseUrl;

const slug = route.params.slug as string;

const tourQuery = useQuery({
  queryKey: ['tours', locale, slug],
  queryFn: async () => {
    try {
      return await api.get<Tour>(`/tours/${locale.value}/${slug}`);
    } catch {
      return null;
    }
  },
});
await tourQuery.suspense();

const tour = tourQuery.data;

if (!tour.value) {
  throw createError({ statusCode: 404, statusMessage: 'Tour not found' });
}

useSeoMeta({
  title: tour.value.title,
  description: tour.value.excerpt || undefined,
  ogTitle: tour.value.title,
  ogDescription: tour.value.excerpt || undefined,
  ogImage: tour.value.coverImageUrl ? `${apiBaseUrl}${tour.value.coverImageUrl}` : undefined,
});

const deleteMutation = useMutation({
  mutationFn: (id: string) => api.delete(`/tours/${id}`),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ['tours'] });
    navigateTo(localePath('/tours'));
  },
});

function handleDelete(): void {
  if (!tour.value) return;
  if (!confirm(t('tours.deleteConfirm'))) return;
  deleteMutation.mutate(tour.value.id);
}
</script>

<template>
  <article v-if="tour" class="mx-auto max-w-3xl px-4 py-12">
    <NuxtLink :to="localePath('/tours')" class="text-sm text-primary-600 hover:underline dark:text-primary-400">
      &larr; {{ $t('tours.heading') }}
    </NuxtLink>

    <div class="mt-4 flex flex-wrap gap-2">
      <UBadge v-if="tour.location" color="secondary" variant="subtle">
        <UIcon name="i-heroicons-map-pin" class="mr-1 h-3.5 w-3.5" />
        {{ tour.location }}
      </UBadge>
      <UBadge v-if="tour.durationDays" color="gray" variant="subtle">
        {{ t('tours.durationValue', { days: tour.durationDays }) }}
      </UBadge>
      <UBadge color="gray" variant="subtle">
        <UIcon name="i-heroicons-calendar-days" class="mr-1 h-3.5 w-3.5" />
        {{ formatDateRange(locale, tour.startDate, tour.endDate) }}
      </UBadge>
    </div>

    <h1 class="mt-4 text-3xl font-extrabold text-text sm:text-4xl">{{ tour.title }}</h1>

    <p v-if="tour.price" class="mt-2 text-xl font-semibold text-brand-600 dark:text-brand-400">
      {{ t('tours.priceValue', { price: tour.price }) }}
    </p>

    <img
      v-if="tour.coverImageUrl"
      :src="`${apiBaseUrl}${tour.coverImageUrl}`"
      :alt="tour.title"
      class="mt-6 w-full rounded-lg object-cover"
    />

    <div class="prose prose-neutral dark:prose-invert mt-8 max-w-none" v-html="tour.content" />

    <UButton
      v-if="auth.user.value && tour.userId === auth.user.value.id"
      color="gray"
      variant="soft"
      class="mt-8"
      :loading="deleteMutation.isPending.value"
      @click="handleDelete"
    >
      {{ $t('tours.deleteButton') }}
    </UButton>
  </article>
</template>
