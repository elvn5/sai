<script setup lang="ts">
import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/vue-query';
import { toTypedSchema } from '@vee-validate/zod';
import { useForm } from 'vee-validate';
import { z } from 'zod';
import PhotoUploader from '~/components/features/photo-uploader/PhotoUploader.vue';
import RatingInput from '~/components/shared/ui/RatingInput.vue';
import RatingStars from '~/components/shared/ui/RatingStars.vue';
import PhotoGrid from '~/components/widgets/photo-grid/PhotoGrid.vue';
import { ApiError, useApiClient } from '~/shared/api/client';
import type { Review, ReviewTargetType, ReviewsPage } from '~/shared/api/types';

const PAGE_SIZE = 10;
const MAX_PHOTOS = 6;

const props = defineProps<{
  targetType: ReviewTargetType;
  targetId: string;
  ownerId: string | null;
}>();

const emit = defineEmits<{ changed: [] }>();

const { t, locale } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const auth = useAuth();
const queryClient = useQueryClient();

const queryKey = computed(() => ['reviews', props.targetType, props.targetId] as const);

const reviewsQuery = useInfiniteQuery({
  queryKey,
  queryFn: ({ pageParam }) =>
    api.get<ReviewsPage>(
      `/reviews?targetType=${props.targetType}&targetId=${props.targetId}&limit=${PAGE_SIZE}&offset=${pageParam}`,
    ),
  initialPageParam: 0,
  getNextPageParam: (lastPage, allPages) => {
    const loaded = allPages.reduce((sum, page) => sum + page.items.length, 0);
    return loaded < lastPage.total ? loaded : undefined;
  },
});

const reviews = computed<Review[]>(
  () => reviewsQuery.data.value?.pages.flatMap((page) => page.items) ?? [],
);
const summary = computed(() => reviewsQuery.data.value?.pages[0]?.summary ?? null);

const currentUserId = computed(() => auth.user.value?.id ?? null);
const isOwner = computed(
  () => currentUserId.value !== null && currentUserId.value === props.ownerId,
);
const hasReviewed = computed(() =>
  reviews.value.some((review: Review) => review.author.id === currentUserId.value),
);
const canWrite = computed(
  () => currentUserId.value !== null && !isOwner.value && !hasReviewed.value,
);

const photos = ref<string[]>([]);
const submitError = ref<string | null>(null);

const { handleSubmit, defineField, errors, resetForm } = useForm({
  validationSchema: toTypedSchema(
    z.object({
      rating: z
        .number({ required_error: t('reviews.ratingRequired') })
        .int()
        .min(1, t('reviews.ratingRequired'))
        .max(5),
      text: z.string().trim().min(10, t('reviews.textTooShort')).max(3000),
    }),
  ),
  initialValues: { rating: 0, text: '' },
});
const [rating] = defineField('rating');
const [text, textAttrs] = defineField('text');

const createMutation = useMutation({
  mutationFn: (values: { rating: number; text: string }) =>
    api.post<Review>('/reviews', {
      targetType: props.targetType,
      targetId: props.targetId,
      rating: values.rating,
      text: values.text,
      photos: photos.value,
    }),
  onSuccess: () => {
    resetForm();
    photos.value = [];
    queryClient.invalidateQueries({ queryKey: queryKey.value });
    emit('changed');
  },
  onError: (error) => {
    submitError.value =
      error instanceof ApiError && error.status === 409
        ? t('reviews.alreadyReviewed')
        : t('reviews.submitError');
  },
});

const deleteMutation = useMutation({
  mutationFn: (id: string) => api.delete(`/reviews/${id}`),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: queryKey.value });
    emit('changed');
  },
});

const onSubmit = handleSubmit((values) => {
  submitError.value = null;
  createMutation.mutate(values);
});

function handleDelete(id: string): void {
  if (confirm(t('reviews.deleteConfirm'))) deleteMutation.mutate(id);
}

function authorName(review: Review): string {
  return review.author.firstName ?? review.author.username ?? t('reviews.anonymous');
}

function formatDate(value: string): string {
  return new Date(value).toLocaleDateString(locale.value, { year: 'numeric', month: 'long' });
}

function barWidth(star: number): string {
  const count = summary.value?.distribution[String(star)] ?? 0;
  const total = summary.value?.count ?? 0;
  return total > 0 ? `${(count / total) * 100}%` : '0%';
}
</script>

<template>
  <section id="reviews" class="scroll-mt-24">
    <h2 class="text-2xl font-bold text-text">{{ $t('reviews.heading') }}</h2>

    <div v-if="summary && summary.count > 0" class="mt-4 flex flex-col gap-6 sm:flex-row sm:items-center">
      <div class="flex flex-col items-center sm:w-40">
        <span class="text-5xl font-extrabold text-text">{{ summary.average.toFixed(1) }}</span>
        <RatingStars :value="summary.average" size="md" />
        <span class="mt-1 text-sm text-text-muted">{{ $t('reviews.count', { count: summary.count }) }}</span>
      </div>
      <div class="flex flex-1 flex-col gap-1.5">
        <div v-for="star in [5, 4, 3, 2, 1]" :key="star" class="flex items-center gap-2 text-sm">
          <span class="w-3 text-text-muted">{{ star }}</span>
          <UIcon name="i-heroicons-star-solid" class="h-4 w-4 text-amber-400" />
          <div class="h-2 flex-1 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
            <div class="h-full rounded-full bg-amber-400" :style="{ width: barWidth(star) }" />
          </div>
          <span class="w-6 text-right text-text-muted">{{ summary.distribution[String(star)] ?? 0 }}</span>
        </div>
      </div>
    </div>

    <UCard v-if="canWrite" class="mt-6">
      <form class="flex flex-col gap-4" @submit="onSubmit">
        <h3 class="font-semibold text-text">{{ $t('reviews.writeTitle') }}</h3>
        <UFormGroup :label="$t('reviews.ratingLabel')" :error="errors.rating">
          <RatingInput v-model="rating" />
        </UFormGroup>
        <UFormGroup :label="$t('reviews.textLabel')" :error="errors.text">
          <UTextarea v-model="text" v-bind="textAttrs" :rows="4" :placeholder="$t('reviews.textPlaceholder')" :maxlength="3000" />
        </UFormGroup>
        <UFormGroup :label="$t('reviews.photosLabel')" :hint="$t('reviews.photosHint')">
          <PhotoUploader v-model="photos" :max="MAX_PHOTOS" folder="reviews" />
        </UFormGroup>
        <p v-if="submitError" class="text-sm text-red-600">{{ submitError }}</p>
        <UButton type="submit" color="primary" class="self-start" :loading="createMutation.isPending.value">
          {{ $t('reviews.submit') }}
        </UButton>
      </form>
    </UCard>
    <p v-else-if="!auth.user.value" class="mt-6 text-sm text-text-muted">
      <NuxtLink :to="localePath('/login')" class="text-primary-600 hover:underline dark:text-primary-400">
        {{ $t('login.cta') }}
      </NuxtLink>
      {{ $t('reviews.loginToReview') }}
    </p>

    <div v-if="reviews.length > 0" class="mt-6 flex flex-col divide-y divide-border">
      <article v-for="review in reviews" :key="review.id" class="flex flex-col gap-2 py-5">
        <div class="flex items-center gap-3">
          <img
            v-if="review.author.avatarUrl"
            :src="review.author.avatarUrl"
            :alt="authorName(review)"
            class="h-10 w-10 rounded-full object-cover"
          />
          <div
            v-else
            class="flex h-10 w-10 items-center justify-center rounded-full bg-gradient-to-br from-brand-500 to-secondary-500 font-bold text-white"
          >
            {{ authorName(review).charAt(0).toUpperCase() }}
          </div>
          <div class="flex-1">
            <p class="font-medium text-text">{{ authorName(review) }}</p>
            <p class="text-xs text-text-muted">{{ formatDate(review.createdAt) }}</p>
          </div>
          <RatingStars :value="review.rating" />
        </div>
        <p class="whitespace-pre-line text-text">{{ review.text }}</p>
        <PhotoGrid v-if="review.photos.length > 0" :photos="review.photos" class="max-w-md" />
        <div v-if="review.reply" class="ml-4 rounded-lg border-l-4 border-brand-500 bg-brand-500/5 px-4 py-3">
          <p class="text-xs font-semibold text-brand-700 dark:text-brand-300">
            {{ $t('reviews.ownerReply') }} · {{ formatDate(review.reply.repliedAt) }}
          </p>
          <p class="mt-1 whitespace-pre-line text-sm text-text">{{ review.reply.text }}</p>
        </div>
        <UButton
          v-if="review.author.id === currentUserId"
          color="gray"
          variant="ghost"
          size="xs"
          class="self-start"
          :loading="deleteMutation.isPending.value"
          @click="handleDelete(review.id)"
        >
          {{ $t('reviews.delete') }}
        </UButton>
      </article>
    </div>
    <p v-else-if="!reviewsQuery.isLoading.value" class="mt-4 text-text-muted">{{ $t('reviews.empty') }}</p>

    <UButton
      v-if="reviewsQuery.hasNextPage.value"
      color="gray"
      variant="soft"
      class="mt-2"
      :loading="reviewsQuery.isFetchingNextPage.value"
      @click="reviewsQuery.fetchNextPage()"
    >
      {{ $t('listing.loadMore') }}
    </UButton>
  </section>
</template>
