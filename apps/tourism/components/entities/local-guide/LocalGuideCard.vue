<script setup lang="ts">
import RatingStars from '~/components/shared/ui/RatingStars.vue';
import type { LocalGuide } from '~/shared/api/types';
import { formatAmount } from '~/shared/lib/format-price';
import { resolveMediaUrl } from '~/shared/lib/resolve-media-url';

const CARD_DESTINATIONS = 3;

const props = defineProps<{ guide: LocalGuide }>();

const { locale } = useI18n();
const localePath = useLocalePath();
const config = useRuntimeConfig();
const apiBaseUrl = config.public.apiBaseUrl;

const avatar = computed(() => props.guide.avatarUrl ?? props.guide.photos[0] ?? null);
</script>

<template>
  <NuxtLink
    :to="localePath(`/local-guides/${guide.slug}`)"
    class="group flex flex-col gap-3 rounded-card border border-border bg-surface p-4 transition hover:shadow-lg"
  >
    <div class="flex items-center gap-3">
      <img
        v-if="avatar"
        :src="resolveMediaUrl(apiBaseUrl, avatar)"
        :alt="guide.name"
        loading="lazy"
        class="h-16 w-16 shrink-0 rounded-full object-cover"
      />
      <div
        v-else
        class="flex h-16 w-16 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-brand-500 to-secondary-500 text-xl font-bold text-white"
      >
        {{ guide.name.charAt(0).toUpperCase() }}
      </div>
      <div class="min-w-0">
        <h3 class="truncate text-lg font-semibold text-text">{{ guide.name }}</h3>
        <p v-if="guide.headline" class="line-clamp-2 text-sm text-text-muted">{{ guide.headline }}</p>
      </div>
    </div>

    <div class="flex items-center gap-2 text-sm">
      <template v-if="guide.reviewCount > 0">
        <span class="font-semibold text-text">{{ guide.ratingAverage.toFixed(1) }}</span>
        <RatingStars :value="guide.ratingAverage" />
      </template>
      <span class="text-text-muted">{{ $t('reviews.count', { count: guide.reviewCount }) }}</span>
    </div>

    <div class="flex flex-wrap gap-1.5">
      <UBadge v-for="language in guide.languages" :key="language" color="gray" variant="subtle" size="xs">
        {{ language.toUpperCase() }}
      </UBadge>
      <UBadge v-if="guide.hasVehicle" color="secondary" variant="subtle" size="xs">
        <UIcon name="i-heroicons-truck" class="mr-1 h-3 w-3" />
        {{ $t('localGuides.vehicleBadge') }}
      </UBadge>
    </div>

    <p class="flex flex-wrap items-center gap-1 text-sm text-text-muted">
      <UIcon name="i-heroicons-map-pin" class="h-4 w-4" />
      <span v-for="(destination, index) in guide.destinations.slice(0, CARD_DESTINATIONS)" :key="destination">
        {{ $t(`destinations.${destination}`) }}<template v-if="index < Math.min(guide.destinations.length, CARD_DESTINATIONS) - 1">,</template>
      </span>
      <span v-if="guide.destinations.length > CARD_DESTINATIONS">+{{ guide.destinations.length - CARD_DESTINATIONS }}</span>
    </p>

    <div class="mt-auto flex items-end justify-between pt-1">
      <span v-if="guide.experienceYears" class="text-sm text-text-muted">
        {{ $t('localGuides.experienceValue', { years: guide.experienceYears }) }}
      </span>
      <span v-if="guide.pricePerDay !== null" class="ml-auto text-right">
        <span class="text-lg font-bold text-brand-600 dark:text-brand-400">
          {{ formatAmount(locale, guide.pricePerDay) }} {{ $t('listing.currency') }}
        </span>
        <span class="text-sm text-text-muted"> / {{ $t('listing.perDay') }}</span>
      </span>
    </div>
  </NuxtLink>
</template>
