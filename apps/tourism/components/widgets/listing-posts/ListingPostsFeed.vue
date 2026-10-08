<script setup lang="ts">
import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/vue-query';
import { toTypedSchema } from '@vee-validate/zod';
import { useForm } from 'vee-validate';
import { z } from 'zod';
import PhotoUploader from '~/components/features/photo-uploader/PhotoUploader.vue';
import PhotoGrid from '~/components/widgets/photo-grid/PhotoGrid.vue';
import { useApiClient } from '~/shared/api/client';
import type { ListingPost, Paginated, ReviewTargetType } from '~/shared/api/types';

const PAGE_SIZE = 5;
const MAX_PHOTOS = 10;

// News and photo updates posted by a listing's owner — the "content" side
// of a hotel or guide page, like a mini social feed.
const props = defineProps<{
  targetType: ReviewTargetType;
  targetId: string;
  ownerId: string | null;
}>();

const { t, locale } = useI18n();
const api = useApiClient();
const auth = useAuth();
const queryClient = useQueryClient();

const queryKey = computed(() => ['listing-posts', props.targetType, props.targetId] as const);

const postsQuery = useInfiniteQuery({
  queryKey,
  queryFn: ({ pageParam }) =>
    api.get<Paginated<ListingPost>>(
      `/listing-posts?targetType=${props.targetType}&targetId=${props.targetId}&limit=${PAGE_SIZE}&offset=${pageParam}`,
    ),
  initialPageParam: 0,
  getNextPageParam: (lastPage, allPages) => {
    const loaded = allPages.reduce((sum, page) => sum + page.items.length, 0);
    return loaded < lastPage.total ? loaded : undefined;
  },
});

const posts = computed(() => postsQuery.data.value?.pages.flatMap((page) => page.items) ?? []);
const isOwner = computed(() => auth.user.value !== null && auth.user.value.id === props.ownerId);

const photos = ref<string[]>([]);
const submitError = ref<string | null>(null);

const { handleSubmit, defineField, errors, resetForm } = useForm({
  validationSchema: toTypedSchema(z.object({ text: z.string().trim().max(3000).optional() })),
});
const [text, textAttrs] = defineField('text');

const createMutation = useMutation({
  mutationFn: (values: { text?: string }) =>
    api.post<ListingPost>('/listing-posts', {
      targetType: props.targetType,
      targetId: props.targetId,
      text: values.text ?? '',
      photos: photos.value,
    }),
  onSuccess: () => {
    resetForm();
    photos.value = [];
    queryClient.invalidateQueries({ queryKey: queryKey.value });
  },
  onError: () => {
    submitError.value = t('posts.submitError');
  },
});

const deleteMutation = useMutation({
  mutationFn: (id: string) => api.delete(`/listing-posts/${id}`),
  onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKey.value }),
});

const onSubmit = handleSubmit((values) => {
  submitError.value = null;
  if (!values.text?.trim() && photos.value.length === 0) {
    submitError.value = t('posts.emptyError');
    return;
  }
  createMutation.mutate(values);
});

function handleDelete(id: string): void {
  if (confirm(t('posts.deleteConfirm'))) deleteMutation.mutate(id);
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
  <section v-if="isOwner || posts.length > 0">
    <h2 class="text-2xl font-bold text-text">{{ $t('posts.heading') }}</h2>

    <UCard v-if="isOwner" class="mt-4">
      <form class="flex flex-col gap-3" @submit="onSubmit">
        <UTextarea
          v-model="text"
          v-bind="textAttrs"
          :rows="3"
          :maxlength="3000"
          :placeholder="$t('posts.composerPlaceholder')"
        />
        <p v-if="errors.text" class="text-sm text-red-600">{{ errors.text }}</p>
        <PhotoUploader v-model="photos" :max="MAX_PHOTOS" folder="posts" />
        <p v-if="submitError" class="text-sm text-red-600">{{ submitError }}</p>
        <UButton type="submit" color="primary" class="self-start" :loading="createMutation.isPending.value">
          {{ $t('posts.submit') }}
        </UButton>
      </form>
    </UCard>

    <div class="mt-4 flex flex-col gap-4">
      <UCard v-for="post in posts" :key="post.id">
        <p class="text-xs text-text-muted">{{ formatDate(post.createdAt) }}</p>
        <p v-if="post.text" class="mt-2 whitespace-pre-line text-text">{{ post.text }}</p>
        <PhotoGrid v-if="post.photos.length > 0" :photos="post.photos" class="mt-3" />
        <UButton
          v-if="isOwner"
          color="gray"
          variant="ghost"
          size="xs"
          class="mt-2"
          :loading="deleteMutation.isPending.value"
          @click="handleDelete(post.id)"
        >
          {{ $t('posts.delete') }}
        </UButton>
      </UCard>
    </div>
    <p v-if="isOwner && posts.length === 0 && !postsQuery.isLoading.value" class="mt-4 text-sm text-text-muted">
      {{ $t('posts.emptyOwner') }}
    </p>

    <UButton
      v-if="postsQuery.hasNextPage.value"
      color="gray"
      variant="soft"
      class="mt-4"
      :loading="postsQuery.isFetchingNextPage.value"
      @click="postsQuery.fetchNextPage()"
    >
      {{ $t('listing.loadMore') }}
    </UButton>
  </section>
</template>
