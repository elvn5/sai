<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import ListingContactCard from '~/components/widgets/listing-contact/ListingContactCard.vue';
import ListingPostsFeed from '~/components/widgets/listing-posts/ListingPostsFeed.vue';
import ReviewsSection from '~/components/widgets/reviews/ReviewsSection.vue';
import { useApiClient } from '~/shared/api/client';
import { ReviewTargetType, type Tour } from '~/shared/api/types';
import { formatAmount } from '~/shared/lib/format-price';
import { resolveMediaUrl } from '~/shared/lib/resolve-media-url';

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
  ogImage: tour.value.coverImageUrl
    ? resolveMediaUrl(apiBaseUrl, tour.value.coverImageUrl)
    : undefined,
});

const priceLabel = computed(() =>
  tour.value?.price
    ? t('tours.priceValue', { price: formatAmount(locale.value, tour.value.price) })
    : null,
);

function refreshTour(): void {
  void tourQuery.refetch();
}

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
  <article v-if="tour" class="mx-auto max-w-6xl px-4 py-12">
    <NuxtLink :to="localePath('/tours')" class="text-sm text-primary-600 hover:underline dark:text-primary-400">
      &larr; {{ $t('tours.heading') }}
    </NuxtLink>

    <div class="mt-4 flex flex-wrap gap-2">
      <NuxtLink v-if="tour.destination" :to="localePath(`/tours?destination=${tour.destination}`)">
        <UBadge color="secondary" variant="subtle">
          <UIcon name="i-heroicons-map-pin" class="mr-1 h-3.5 w-3.5" />
          {{ $t(`destinations.${tour.destination}`) }}
        </UBadge>
      </NuxtLink>
      <UBadge v-if="tour.location" color="gray" variant="subtle">{{ tour.location }}</UBadge>
      <UBadge v-if="tour.durationDays" color="gray" variant="subtle">
        {{ t('tours.durationValue', { days: tour.durationDays }) }}
      </UBadge>
      <UBadge color="gray" variant="subtle">
        <UIcon name="i-heroicons-calendar-days" class="mr-1 h-3.5 w-3.5" />
        {{ formatDateRange(locale, tour.startDate, tour.endDate) }}
      </UBadge>
    </div>

    <h1 class="mt-4 text-3xl font-extrabold text-text sm:text-4xl">{{ tour.title }}</h1>

    <div class="mt-6 flex flex-col gap-10 lg:flex-row">
      <div class="flex min-w-0 flex-1 flex-col gap-10">
        <div>
          <img
            v-if="tour.coverImageUrl"
            :src="resolveMediaUrl(apiBaseUrl, tour.coverImageUrl)"
            :alt="tour.title"
            class="w-full rounded-lg object-cover"
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
        </div>

        <ListingPostsFeed :target-type="ReviewTargetType.TOUR" :target-id="tour.id" :owner-id="tour.userId" />

        <ReviewsSection
          :target-type="ReviewTargetType.TOUR"
          :target-id="tour.id"
          :owner-id="tour.userId"
          @changed="refreshTour"
        />
      </div>

      <aside class="order-first lg:order-none lg:w-80 lg:shrink-0">
        <div class="lg:sticky lg:top-20">
          <ListingContactCard
            :listing-name="tour.title"
            :price-label="priceLabel"
            :rating-average="tour.ratingAverage"
            :review-count="tour.reviewCount"
            :cancellation-policy="tour.cancellationPolicy"
            :phone="tour.phone"
            :whatsapp="tour.whatsapp"
            :telegram="tour.telegram"
          />
        </div>
      </aside>
    </div>
  </article>
</template>
