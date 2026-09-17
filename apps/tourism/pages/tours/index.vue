<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import { useApiClient } from '~/shared/api/client';
import { PartnerStatus, SeoPage } from '~/shared/api/types';
import type { Tour } from '~/shared/api/types';

const { locale, t } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const auth = useAuth();
const config = useRuntimeConfig();
const apiBaseUrl = config.public.apiBaseUrl;

const toursQuery = useQuery({
  queryKey: ['tours', locale],
  queryFn: () => api.get<Tour[]>(`/tours?locale=${locale.value}`),
});
await toursQuery.suspense();

const tours = toursQuery.data;

const canPost = computed(() => auth.user.value?.partnerStatus === PartnerStatus.APPROVED);

const pageSeoQuery = usePageSeo(SeoPage.TOURS);
await pageSeoQuery.suspense();
const pageSeo = pageSeoQuery.data;

useSeoMeta({
  title: () => pageSeo.value?.title || t('tours.heading'),
  description: () => pageSeo.value?.description || t('tours.metaDescription'),
  ogTitle: () => pageSeo.value?.title || t('tours.heading'),
  ogDescription: () => pageSeo.value?.description || t('tours.metaDescription'),
  ogImage: () =>
    pageSeo.value?.ogImageUrl ? `${apiBaseUrl}${pageSeo.value.ogImageUrl}` : undefined,
});
</script>

<template>
  <section class="mx-auto max-w-4xl px-4 py-12">
    <div class="mb-8 flex items-center justify-between">
      <h1 class="text-3xl font-extrabold text-text">{{ $t('tours.heading') }}</h1>
      <UButton v-if="canPost" color="primary" icon="i-heroicons-plus" :to="localePath('/tours/new')">
        {{ $t('tours.addButton') }}
      </UButton>
    </div>

    <div v-if="tours && tours.length > 0" class="flex flex-col gap-4">
      <UCard v-for="tour in tours" :key="tour.id">
        <NuxtLink :to="localePath(`/tours/${tour.slug}`)" class="flex flex-col gap-4 sm:flex-row">
          <img
            v-if="tour.coverImageUrl"
            :src="`${apiBaseUrl}${tour.coverImageUrl}`"
            :alt="tour.title"
            class="h-40 w-full rounded-lg object-cover sm:w-56"
          />
          <div class="flex flex-1 flex-col justify-center">
            <div class="mb-2 flex flex-wrap gap-2">
              <UBadge v-if="tour.location" color="secondary" variant="subtle">
                <UIcon name="i-heroicons-map-pin" class="mr-1 h-3.5 w-3.5" />
                {{ tour.location }}
              </UBadge>
              <UBadge v-if="tour.durationDays" color="gray" variant="subtle">
                {{ t('tours.durationValue', { days: tour.durationDays }) }}
              </UBadge>
            </div>
            <h2 class="text-xl font-semibold text-text">{{ tour.title }}</h2>
            <p v-if="tour.excerpt" class="mt-2 text-text-muted">{{ tour.excerpt }}</p>
            <p v-if="tour.price" class="mt-2 font-semibold text-brand-600 dark:text-brand-400">
              {{ t('tours.priceValue', { price: tour.price }) }}
            </p>
          </div>
        </NuxtLink>
      </UCard>
    </div>
    <p v-else class="text-text-muted">{{ $t('tours.empty') }}</p>
  </section>
</template>
