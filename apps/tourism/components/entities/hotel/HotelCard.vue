<script setup lang="ts">
import RatingStars from '~/components/shared/ui/RatingStars.vue';
import type { Hotel } from '~/shared/api/types';
import { formatAmount } from '~/shared/lib/format-price';
import { HOTEL_AMENITY_OPTIONS, findOption } from '~/shared/lib/listing-options';
import { resolveMediaUrl } from '~/shared/lib/resolve-media-url';

const CARD_AMENITIES = 4;

const props = defineProps<{ hotel: Hotel }>();

const { locale } = useI18n();
const localePath = useLocalePath();
const config = useRuntimeConfig();
const apiBaseUrl = config.public.apiBaseUrl;

const cover = computed(() => props.hotel.coverImageUrl ?? props.hotel.photos[0] ?? null);
const amenities = computed(() =>
  props.hotel.amenities
    .slice(0, CARD_AMENITIES)
    .map((value) => findOption(HOTEL_AMENITY_OPTIONS, value))
    .filter((option) => option !== undefined),
);
</script>

<template>
  <NuxtLink
    :to="localePath(`/hotels/${hotel.slug}`)"
    class="group flex flex-col overflow-hidden rounded-card border border-border bg-surface transition hover:shadow-lg"
  >
    <div class="relative aspect-[4/3] overflow-hidden bg-gray-100 dark:bg-gray-800">
      <img
        v-if="cover"
        :src="resolveMediaUrl(apiBaseUrl, cover)"
        :alt="hotel.name"
        loading="lazy"
        class="h-full w-full object-cover transition group-hover:scale-105"
      />
      <UIcon v-else name="i-heroicons-building-office-2" class="absolute inset-0 m-auto h-12 w-12 text-text-muted" />
      <span class="absolute left-2 top-2 rounded-full bg-black/60 px-2.5 py-1 text-xs font-medium text-white">
        {{ $t(`hotels.types.${hotel.type}`) }}
      </span>
      <span
        v-if="hotel.cancellationPolicy === 'free'"
        class="absolute right-2 top-2 rounded-full bg-green-600 px-2.5 py-1 text-xs font-medium text-white"
      >
        {{ $t('listing.cancellation.free') }}
      </span>
    </div>

    <div class="flex flex-1 flex-col gap-2 p-4">
      <p class="flex items-center gap-1 text-sm text-text-muted">
        <UIcon name="i-heroicons-map-pin" class="h-4 w-4" />
        {{ $t(`destinations.${hotel.destination}`) }}
      </p>
      <h3 class="text-lg font-semibold leading-snug text-text">
        {{ hotel.name }}
        <span v-if="hotel.stars" class="ml-1 text-sm text-amber-500">{{ '★'.repeat(hotel.stars) }}</span>
      </h3>

      <div class="flex items-center gap-2 text-sm">
        <template v-if="hotel.reviewCount > 0">
          <span class="rounded bg-brand-600 px-1.5 py-0.5 font-semibold text-white">{{ hotel.ratingAverage.toFixed(1) }}</span>
          <RatingStars :value="hotel.ratingAverage" />
        </template>
        <span class="text-text-muted">{{ $t('reviews.count', { count: hotel.reviewCount }) }}</span>
      </div>

      <div v-if="amenities.length > 0" class="flex flex-wrap gap-x-3 gap-y-1 text-xs text-text-muted">
        <span v-for="amenity in amenities" :key="amenity.value" class="inline-flex items-center gap-1">
          <UIcon v-if="amenity.icon" :name="amenity.icon" class="h-3.5 w-3.5" />
          {{ $t(amenity.labelKey) }}
        </span>
      </div>

      <p v-if="hotel.priceFrom !== null" class="mt-auto pt-2 text-right text-text">
        <span class="text-sm text-text-muted">{{ $t('listing.from') }}</span>
        <span class="text-xl font-bold text-brand-600 dark:text-brand-400">
          {{ formatAmount(locale, hotel.priceFrom) }} {{ $t('listing.currency') }}
        </span>
        <span class="text-sm text-text-muted"> / {{ $t('listing.perNight') }}</span>
      </p>
    </div>
  </NuxtLink>
</template>
