<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import RatingStars from '~/components/shared/ui/RatingStars.vue';
import ListingContactCard from '~/components/widgets/listing-contact/ListingContactCard.vue';
import ListingPostsFeed from '~/components/widgets/listing-posts/ListingPostsFeed.vue';
import LocationMap from '~/components/widgets/location-map/LocationMap.vue';
import PhotoGrid from '~/components/widgets/photo-grid/PhotoGrid.vue';
import ReviewsSection from '~/components/widgets/reviews/ReviewsSection.vue';
import { useApiClient } from '~/shared/api/client';
import { type Hotel, ReviewTargetType } from '~/shared/api/types';
import { formatAmount } from '~/shared/lib/format-price';
import { HOTEL_AMENITY_OPTIONS, findOption } from '~/shared/lib/listing-options';
import { resolveMediaUrl } from '~/shared/lib/resolve-media-url';

const route = useRoute();
const { t, locale } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const auth = useAuth();
const queryClient = useQueryClient();
const config = useRuntimeConfig();
const apiBaseUrl = config.public.apiBaseUrl;

const slug = route.params.slug as string;

const hotelQuery = useQuery({
  queryKey: ['hotel', slug],
  queryFn: async () => {
    try {
      return await api.get<Hotel>(`/hotels/${slug}`);
    } catch {
      return null;
    }
  },
});
await hotelQuery.suspense();
const hotel = hotelQuery.data;

if (!hotel.value) {
  throw createError({ statusCode: 404, statusMessage: 'Hotel not found' });
}

const isOwner = computed(() =>
  Boolean(auth.user.value && hotel.value?.userId === auth.user.value.id),
);
const amenities = computed(() =>
  (hotel.value?.amenities ?? [])
    .map((value) => findOption(HOTEL_AMENITY_OPTIONS, value))
    .filter((o) => o !== undefined),
);
const priceLabel = computed(() =>
  hotel.value?.priceFrom != null
    ? t('listing.priceFromPerNight', { price: formatAmount(locale.value, hotel.value.priceFrom) })
    : null,
);
const cover = computed(() => hotel.value?.coverImageUrl ?? hotel.value?.photos[0] ?? null);

useSeoMeta({
  title: () => `${hotel.value?.name} — ${t(`destinations.${hotel.value?.destination}`)}`,
  description: () =>
    t('hotels.detailMeta', {
      name: hotel.value?.name,
      place: t(`destinations.${hotel.value?.destination}`),
    }),
  ogTitle: () => hotel.value?.name,
  ogImage: () => (cover.value ? resolveMediaUrl(apiBaseUrl, cover.value) : undefined),
});

useHead({
  script: [
    {
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'LodgingBusiness',
        name: hotel.value.name,
        address: hotel.value.address ?? undefined,
        image: cover.value ? resolveMediaUrl(apiBaseUrl, cover.value) : undefined,
        priceRange: hotel.value.priceFrom ? `${hotel.value.priceFrom} KGS+` : undefined,
        geo:
          hotel.value.latitude !== null && hotel.value.longitude !== null
            ? {
                '@type': 'GeoCoordinates',
                latitude: hotel.value.latitude,
                longitude: hotel.value.longitude,
              }
            : undefined,
        aggregateRating:
          hotel.value.reviewCount > 0
            ? {
                '@type': 'AggregateRating',
                ratingValue: hotel.value.ratingAverage,
                reviewCount: hotel.value.reviewCount,
              }
            : undefined,
      }),
    },
  ],
});

const deleteMutation = useMutation({
  mutationFn: (id: string) => api.delete(`/hotels/${id}`),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ['hotels'] });
    navigateTo(localePath('/hotels'));
  },
});

function handleDelete(): void {
  if (hotel.value && confirm(t('hotels.deleteConfirm'))) deleteMutation.mutate(hotel.value.id);
}

// Reviews change the denormalized rating shown in the header.
function refreshHotel(): void {
  void hotelQuery.refetch();
}
</script>

<template>
  <article v-if="hotel" class="mx-auto max-w-6xl px-4 py-10">
    <NuxtLink :to="localePath('/hotels')" class="text-sm text-primary-600 hover:underline dark:text-primary-400">
      &larr; {{ $t('hotels.heading') }}
    </NuxtLink>

    <div class="mt-4 flex flex-wrap items-start justify-between gap-4">
      <div>
        <div class="flex flex-wrap gap-2">
          <UBadge color="gray" variant="subtle">{{ $t(`hotels.types.${hotel.type}`) }}</UBadge>
          <NuxtLink :to="localePath(`/hotels?destination=${hotel.destination}`)">
            <UBadge color="secondary" variant="subtle">
              <UIcon name="i-heroicons-map-pin" class="mr-1 h-3.5 w-3.5" />
              {{ $t(`destinations.${hotel.destination}`) }}
            </UBadge>
          </NuxtLink>
        </div>
        <h1 class="mt-2 text-3xl font-extrabold text-text sm:text-4xl">
          {{ hotel.name }}
          <span v-if="hotel.stars" class="text-xl text-amber-500">{{ '★'.repeat(hotel.stars) }}</span>
        </h1>
        <a href="#reviews" class="mt-2 flex items-center gap-2 text-sm">
          <RatingStars :value="hotel.ratingAverage" />
          <span v-if="hotel.reviewCount > 0" class="font-semibold text-text">{{ hotel.ratingAverage.toFixed(1) }}</span>
          <span class="text-text-muted">{{ $t('reviews.count', { count: hotel.reviewCount }) }}</span>
        </a>
      </div>
      <div v-if="isOwner" class="flex gap-2">
        <UButton color="gray" icon="i-heroicons-pencil-square" :to="localePath(`/hotels/${hotel.slug}/edit`)">
          {{ $t('listing.edit') }}
        </UButton>
        <UButton color="red" variant="soft" icon="i-heroicons-trash" :loading="deleteMutation.isPending.value" @click="handleDelete">
          {{ $t('listing.delete') }}
        </UButton>
      </div>
    </div>

    <PhotoGrid :photos="hotel.photos" :alt="hotel.name" variant="hero" class="mt-6" />

    <div class="mt-8 flex flex-col gap-10 lg:flex-row">
      <div class="flex min-w-0 flex-1 flex-col gap-10">
        <section v-if="amenities.length > 0">
          <h2 class="text-2xl font-bold text-text">{{ $t('hotels.amenitiesLabel') }}</h2>
          <ul class="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3">
            <li v-for="amenity in amenities" :key="amenity.value" class="flex items-center gap-2 text-text">
              <UIcon v-if="amenity.icon" :name="amenity.icon" class="h-5 w-5 text-brand-600" />
              {{ $t(amenity.labelKey) }}
            </li>
          </ul>
        </section>

        <section>
          <h2 class="text-2xl font-bold text-text">{{ $t('hotels.about') }}</h2>
          <div class="prose prose-neutral mt-4 max-w-none dark:prose-invert" v-html="hotel.description" />
          <div v-if="hotel.checkInTime || hotel.checkOutTime" class="mt-4 flex flex-wrap gap-6 text-sm text-text">
            <span v-if="hotel.checkInTime" class="flex items-center gap-2">
              <UIcon name="i-heroicons-arrow-right-end-on-rectangle" class="h-5 w-5 text-text-muted" />
              {{ $t('hotels.checkIn', { time: hotel.checkInTime }) }}
            </span>
            <span v-if="hotel.checkOutTime" class="flex items-center gap-2">
              <UIcon name="i-heroicons-arrow-left-start-on-rectangle" class="h-5 w-5 text-text-muted" />
              {{ $t('hotels.checkOut', { time: hotel.checkOutTime }) }}
            </span>
          </div>
        </section>

        <section v-if="hotel.address || hotel.latitude !== null">
          <h2 class="mb-4 text-2xl font-bold text-text">{{ $t('listing.map.heading') }}</h2>
          <LocationMap :latitude="hotel.latitude" :longitude="hotel.longitude" :address="hotel.address" />
        </section>

        <ListingPostsFeed :target-type="ReviewTargetType.HOTEL" :target-id="hotel.id" :owner-id="hotel.userId" />

        <ReviewsSection
          :target-type="ReviewTargetType.HOTEL"
          :target-id="hotel.id"
          :owner-id="hotel.userId"
          @changed="refreshHotel"
        />
      </div>

      <aside class="order-first lg:order-none lg:w-80 lg:shrink-0">
        <div class="lg:sticky lg:top-20">
          <ListingContactCard
            :listing-name="hotel.name"
            :price-label="priceLabel"
            :rating-average="hotel.ratingAverage"
            :review-count="hotel.reviewCount"
            :cancellation-policy="hotel.cancellationPolicy"
            :phone="hotel.phone"
            :whatsapp="hotel.whatsapp"
            :telegram="hotel.telegram"
            :website="hotel.website"
          />
        </div>
      </aside>
    </div>
  </article>
</template>
