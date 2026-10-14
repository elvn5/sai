<script setup lang="ts">
import ReviewReplyForm from '~/components/features/review-reply/ReviewReplyForm.vue';
import RatingStars from '~/components/shared/ui/RatingStars.vue';
import PhotoGrid from '~/components/widgets/photo-grid/PhotoGrid.vue';
import type { PartnerReview } from '~/shared/api/types';
import { listingPath } from '~/shared/lib/listing-path';
import { usePartnerReviews } from '~/shared/lib/partner-cabinet';
import { useRequirePartner } from '~/shared/lib/require-auth';

definePageMeta({ layout: 'partner' });

const { t, locale } = useI18n();
const localePath = useLocalePath();

useRequirePartner();
useSeoMeta({ title: () => t('cabinet.nav.reviews'), robots: 'noindex' });

const unanswered = ref(false);
const reviewsQuery = usePartnerReviews(unanswered);
const reviews = computed(() => reviewsQuery.data.value?.items ?? []);

const filterOptions = computed(() => [
  { value: false, label: t('cabinet.reviews.all') },
  { value: true, label: t('cabinet.reviews.unanswered') },
]);

function authorName(review: PartnerReview): string {
  return review.author.firstName ?? review.author.username ?? t('reviews.anonymous');
}

function formatDate(value: string): string {
  return new Date(value).toLocaleDateString(locale.value, {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  });
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <div class="flex flex-wrap items-end justify-between gap-3">
      <div>
        <h1 class="text-2xl font-extrabold text-text sm:text-3xl">{{ $t('cabinet.nav.reviews') }}</h1>
        <p class="mt-1 text-sm text-text-muted">{{ $t('cabinet.reviews.subtitle') }}</p>
      </div>
      <div class="flex rounded-lg border border-border bg-surface p-0.5">
        <button
          v-for="option in filterOptions"
          :key="String(option.value)"
          type="button"
          class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
          :class="unanswered === option.value ? 'bg-brand-600 text-white' : 'text-text-muted hover:text-text'"
          @click="unanswered = option.value"
        >
          {{ option.label }}
        </button>
      </div>
    </div>

    <div v-if="reviews.length > 0" class="flex flex-col gap-3">
      <article
        v-for="review in reviews"
        :key="review.id"
        class="flex flex-col gap-3 rounded-card border border-border bg-surface px-4 py-4"
      >
        <NuxtLink
          :to="localePath(listingPath(review.listing))"
          class="self-start text-xs font-semibold uppercase tracking-wide text-text-muted hover:underline"
        >
          {{ review.listing.title }}
        </NuxtLink>
        <div class="flex items-center justify-between gap-3">
          <div>
            <p class="font-medium text-text">{{ authorName(review) }}</p>
            <p class="text-xs text-text-muted">{{ formatDate(review.createdAt) }}</p>
          </div>
          <RatingStars :value="review.rating" />
        </div>
        <p class="whitespace-pre-line text-text">{{ review.text }}</p>
        <PhotoGrid v-if="review.photos.length > 0" :photos="review.photos" class="max-w-md" />
        <ReviewReplyForm :key="`${review.id}-${review.reply?.repliedAt ?? ''}`" :review="review" />
      </article>
    </div>
    <div
      v-else-if="!reviewsQuery.isLoading.value"
      class="flex flex-col items-center gap-2 rounded-card border border-dashed border-border bg-surface px-6 py-12 text-center"
    >
      <UIcon name="i-heroicons-chat-bubble-left-right" class="h-10 w-10 text-text-muted" />
      <p class="font-medium text-text">
        {{ unanswered ? $t('cabinet.reviews.allAnswered') : $t('cabinet.reviews.empty') }}
      </p>
      <p class="max-w-sm text-sm text-text-muted">{{ $t('cabinet.reviews.why') }}</p>
    </div>
  </div>
</template>
