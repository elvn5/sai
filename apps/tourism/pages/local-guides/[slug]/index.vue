<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import RatingStars from '~/components/shared/ui/RatingStars.vue';
import ListingContactCard from '~/components/widgets/listing-contact/ListingContactCard.vue';
import ListingPostsFeed from '~/components/widgets/listing-posts/ListingPostsFeed.vue';
import PhotoGrid from '~/components/widgets/photo-grid/PhotoGrid.vue';
import ReviewsSection from '~/components/widgets/reviews/ReviewsSection.vue';
import { useApiClient } from '~/shared/api/client';
import { type LocalGuide, ReviewTargetType } from '~/shared/api/types';
import { formatAmount } from '~/shared/lib/format-price';
import { GUIDE_SPECIALIZATION_OPTIONS, findOption } from '~/shared/lib/listing-options';
import { useTrackListingView } from '~/shared/lib/listing-stats';
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

const guideQuery = useQuery({
  queryKey: ['local-guide', slug],
  queryFn: async () => {
    try {
      return await api.get<LocalGuide>(`/local-guides/${slug}`);
    } catch {
      return null;
    }
  },
});
await guideQuery.suspense();
const guide = guideQuery.data;

if (!guide.value) {
  throw createError({ statusCode: 404, statusMessage: 'Guide not found' });
}

useTrackListingView(ReviewTargetType.LOCAL_GUIDE, () => guide.value?.id);

const isOwner = computed(() =>
  Boolean(auth.user.value && guide.value?.userId === auth.user.value.id),
);
const specializations = computed(() =>
  (guide.value?.specializations ?? [])
    .map((value) => findOption(GUIDE_SPECIALIZATION_OPTIONS, value))
    .filter((o) => o !== undefined),
);
const priceLabel = computed(() =>
  guide.value?.pricePerDay != null
    ? t('listing.pricePerDay', { price: formatAmount(locale.value, guide.value.pricePerDay) })
    : null,
);
const avatar = computed(() => guide.value?.avatarUrl ?? null);

useSeoMeta({
  title: () => t('localGuides.detailTitle', { name: guide.value?.name }),
  description: () => guide.value?.headline ?? t('localGuides.metaDescription'),
  ogTitle: () => guide.value?.name,
  ogImage: () => {
    const image = guide.value?.coverImageUrl ?? avatar.value;
    return image ? resolveMediaUrl(apiBaseUrl, image) : undefined;
  },
});

const deleteMutation = useMutation({
  mutationFn: (id: string) => api.delete(`/local-guides/${id}`),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ['local-guides'] });
    queryClient.invalidateQueries({ queryKey: ['my-local-guide'] });
    navigateTo(localePath('/local-guides'));
  },
});

function handleDelete(): void {
  if (guide.value && confirm(t('localGuides.deleteConfirm'))) deleteMutation.mutate(guide.value.id);
}

function refreshGuide(): void {
  void guideQuery.refetch();
}
</script>

<template>
  <article v-if="guide" class="mx-auto max-w-6xl px-4 py-10">
    <NuxtLink :to="localePath('/local-guides')" class="text-sm text-primary-600 hover:underline dark:text-primary-400">
      &larr; {{ $t('localGuides.heading') }}
    </NuxtLink>

    <div class="mt-6 flex flex-col gap-6 sm:flex-row sm:items-center">
      <img
        v-if="avatar"
        :src="resolveMediaUrl(apiBaseUrl, avatar)"
        :alt="guide.name"
        class="h-28 w-28 shrink-0 rounded-full object-cover ring-4 ring-brand-500/20"
      />
      <div
        v-else
        class="flex h-28 w-28 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-brand-500 to-secondary-500 text-4xl font-bold text-white"
      >
        {{ guide.name.charAt(0).toUpperCase() }}
      </div>
      <div class="flex-1">
        <h1 class="text-3xl font-extrabold text-text sm:text-4xl">{{ guide.name }}</h1>
        <p v-if="guide.headline" class="mt-1 text-lg text-text-muted">{{ guide.headline }}</p>
        <a href="#reviews" class="mt-2 flex items-center gap-2 text-sm">
          <RatingStars :value="guide.ratingAverage" />
          <span v-if="guide.reviewCount > 0" class="font-semibold text-text">{{ guide.ratingAverage.toFixed(1) }}</span>
          <span class="text-text-muted">{{ $t('reviews.count', { count: guide.reviewCount }) }}</span>
        </a>
      </div>
      <div v-if="isOwner" class="flex gap-2">
        <UButton color="gray" icon="i-heroicons-pencil-square" :to="localePath(`/local-guides/${guide.slug}/edit`)">
          {{ $t('listing.edit') }}
        </UButton>
        <UButton color="red" variant="soft" icon="i-heroicons-trash" :loading="deleteMutation.isPending.value" @click="handleDelete">
          {{ $t('listing.delete') }}
        </UButton>
      </div>
    </div>

    <div class="mt-8 flex flex-col gap-10 lg:flex-row">
      <div class="flex min-w-0 flex-1 flex-col gap-10">
        <section class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div class="rounded-card border border-border bg-surface p-4">
            <h2 class="text-sm font-medium text-text-muted">{{ $t('localGuides.languagesLabel') }}</h2>
            <div class="mt-2 flex flex-wrap gap-1.5">
              <UBadge v-for="language in guide.languages" :key="language" color="gray" variant="subtle">
                {{ $t(`localGuides.languages.${language}`) }}
              </UBadge>
            </div>
          </div>
          <div class="rounded-card border border-border bg-surface p-4">
            <h2 class="text-sm font-medium text-text-muted">{{ $t('localGuides.destinationsLabel') }}</h2>
            <div class="mt-2 flex flex-wrap gap-1.5">
              <NuxtLink
                v-for="destination in guide.destinations"
                :key="destination"
                :to="localePath(`/local-guides?destination=${destination}`)"
              >
                <UBadge color="secondary" variant="subtle">{{ $t(`destinations.${destination}`) }}</UBadge>
              </NuxtLink>
            </div>
          </div>
          <div class="rounded-card border border-border bg-surface p-4 sm:col-span-2">
            <h2 class="text-sm font-medium text-text-muted">{{ $t('localGuides.specializationsLabel') }}</h2>
            <ul class="mt-2 flex flex-wrap gap-x-5 gap-y-2">
              <li v-for="item in specializations" :key="item.value" class="flex items-center gap-1.5 text-text">
                <UIcon v-if="item.icon" :name="item.icon" class="h-5 w-5 text-brand-600" />
                {{ $t(item.labelKey) }}
              </li>
            </ul>
            <div class="mt-3 flex flex-wrap gap-4 text-sm text-text-muted">
              <span v-if="guide.experienceYears" class="flex items-center gap-1">
                <UIcon name="i-heroicons-academic-cap" class="h-4 w-4" />
                {{ $t('localGuides.experienceValue', { years: guide.experienceYears }) }}
              </span>
              <span v-if="guide.hasVehicle" class="flex items-center gap-1">
                <UIcon name="i-heroicons-truck" class="h-4 w-4" />
                {{ $t('localGuides.vehicleBadge') }}
              </span>
            </div>
          </div>
        </section>

        <section>
          <h2 class="text-2xl font-bold text-text">{{ $t('localGuides.about') }}</h2>
          <div class="prose prose-neutral mt-4 max-w-none dark:prose-invert" v-html="guide.bio" />
        </section>

        <section v-if="guide.photos.length > 0">
          <h2 class="mb-4 text-2xl font-bold text-text">{{ $t('listing.photos.heading') }}</h2>
          <PhotoGrid :photos="guide.photos" :alt="guide.name" />
        </section>

        <ListingPostsFeed :target-type="ReviewTargetType.LOCAL_GUIDE" :target-id="guide.id" :owner-id="guide.userId" />

        <ReviewsSection
          :target-type="ReviewTargetType.LOCAL_GUIDE"
          :target-id="guide.id"
          :owner-id="guide.userId"
          @changed="refreshGuide"
        />
      </div>

      <aside class="order-first lg:order-none lg:w-80 lg:shrink-0">
        <div class="lg:sticky lg:top-20">
          <ListingContactCard
            :target-type="ReviewTargetType.LOCAL_GUIDE"
            :target-id="guide.id"
            :listing-name="guide.name"
            :price-label="priceLabel"
            :rating-average="guide.ratingAverage"
            :review-count="guide.reviewCount"
            :cancellation-policy="null"
            :phone="guide.phone"
            :whatsapp="guide.whatsapp"
            :telegram="guide.telegram"
          />
        </div>
      </aside>
    </div>
  </article>
</template>
