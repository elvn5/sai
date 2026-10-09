<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import RatingStars from '~/components/shared/ui/RatingStars.vue';
import { useApiClient } from '~/shared/api/client';
import { SeoPage } from '~/shared/api/types';
import type { Tour } from '~/shared/api/types';
import { formatAmount } from '~/shared/lib/format-price';
import { DESTINATION_OPTIONS } from '~/shared/lib/listing-options';
import { queryString } from '~/shared/lib/listing-query';
import { resolveMediaUrl } from '~/shared/lib/resolve-media-url';

const { locale, t } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const auth = useAuth();
const config = useRuntimeConfig();
const apiBaseUrl = config.public.apiBaseUrl;
const route = useRoute();
const router = useRouter();

const activeDestination = ref(queryString(route.query, 'destination') ?? null);
watch(activeDestination, (destination: string | null) => {
  void router.replace({ query: destination ? { destination } : {} });
});

const toursQuery = useQuery({
  queryKey: ['tours', locale, activeDestination],
  queryFn: () => {
    const params = new URLSearchParams({ locale: locale.value });
    if (activeDestination.value) params.set('destination', activeDestination.value);
    return api.get<Tour[]>(`/tours?${params.toString()}`);
  },
});
await toursQuery.suspense();

const tours = toursQuery.data;

const canPost = auth.isPartner;

const pageSeoQuery = usePageSeo(SeoPage.TOURS, locale);
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

    <div class="mb-8 flex flex-wrap gap-2">
      <button
        v-for="option in [{ value: null, labelKey: 'listing.filters.allDestinations' }, ...DESTINATION_OPTIONS]"
        :key="option.value ?? 'all'"
        type="button"
        class="rounded-full border px-4 py-1.5 text-sm font-medium transition-colors"
        :class="
          activeDestination === option.value
            ? 'border-brand-500 bg-brand-500/10 text-brand-600 dark:text-brand-400'
            : 'border-border text-text-muted hover:border-brand-500/50 hover:text-text'
        "
        @click="activeDestination = option.value"
      >
        {{ $t(option.labelKey) }}
      </button>
    </div>

    <div v-if="tours && tours.length > 0" class="flex flex-col gap-4">
      <UCard v-for="tour in tours" :key="tour.id">
        <NuxtLink :to="localePath(`/tours/${tour.slug}`)" class="flex flex-col gap-4 sm:flex-row">
          <img
            v-if="tour.coverImageUrl"
            :src="resolveMediaUrl(apiBaseUrl, tour.coverImageUrl)"
            :alt="tour.title"
            class="h-40 w-full rounded-lg object-cover sm:w-56"
          />
          <div class="flex flex-1 flex-col justify-center">
            <div class="mb-2 flex flex-wrap gap-2">
              <UBadge v-if="tour.destination || tour.location" color="secondary" variant="subtle">
                <UIcon name="i-heroicons-map-pin" class="mr-1 h-3.5 w-3.5" />
                {{ tour.destination ? $t(`destinations.${tour.destination}`) : tour.location }}
              </UBadge>
              <UBadge v-if="tour.cancellationPolicy === 'free'" color="green" variant="subtle">
                {{ $t('listing.cancellation.free') }}
              </UBadge>
              <UBadge v-if="tour.durationDays" color="gray" variant="subtle">
                {{ t('tours.durationValue', { days: tour.durationDays }) }}
              </UBadge>
              <UBadge color="gray" variant="subtle">
                <UIcon name="i-heroicons-calendar-days" class="mr-1 h-3.5 w-3.5" />
                {{ formatDateRange(locale, tour.startDate, tour.endDate) }}
              </UBadge>
            </div>
            <h2 class="text-xl font-semibold text-text">{{ tour.title }}</h2>
            <p v-if="tour.excerpt" class="mt-2 text-text-muted">{{ tour.excerpt }}</p>
            <div class="mt-2 flex flex-wrap items-center justify-between gap-2">
              <span v-if="tour.reviewCount > 0" class="flex items-center gap-1.5 text-sm">
                <RatingStars :value="tour.ratingAverage" />
                <span class="text-text-muted">{{ $t('reviews.count', { count: tour.reviewCount }) }}</span>
              </span>
              <p v-if="tour.price" class="font-semibold text-brand-600 dark:text-brand-400">
                {{ t('tours.priceValue', { price: formatAmount(locale, tour.price) }) }}
              </p>
            </div>
          </div>
        </NuxtLink>
      </UCard>
    </div>
    <p v-else class="text-text-muted">{{ $t('tours.empty') }}</p>
  </section>
</template>
