<script setup lang="ts">
import { useMutation, useQueryClient } from '@tanstack/vue-query';
import { useApiClient } from '~/shared/api/client';
import type { Review } from '~/shared/api/types';

// The listing owner's public answer to a review: write, edit or remove it.
const props = defineProps<{ review: Review }>();

const { t } = useI18n();
const api = useApiClient();
const queryClient = useQueryClient();

const MAX_LENGTH = 2000;
const editing = ref(!props.review.reply);
const text = ref(props.review.reply?.text ?? '');
const error = ref<string | null>(null);

function invalidate(): void {
  queryClient.invalidateQueries({ queryKey: ['partner', 'reviews'] });
  queryClient.invalidateQueries({ queryKey: ['reviews'] });
}

const saveMutation = useMutation({
  mutationFn: () => api.put<Review>(`/reviews/${props.review.id}/reply`, { text: text.value }),
  onSuccess: () => {
    editing.value = false;
    invalidate();
  },
  onError: () => {
    error.value = t('cabinet.reviews.saveError');
  },
});

const removeMutation = useMutation({
  mutationFn: () => api.delete<Review>(`/reviews/${props.review.id}/reply`),
  onSuccess: () => {
    text.value = '';
    editing.value = true;
    invalidate();
  },
});

function save(): void {
  error.value = null;
  if (!text.value.trim()) {
    error.value = t('cabinet.reviews.emptyReply');
    return;
  }
  saveMutation.mutate();
}

function remove(): void {
  if (window.confirm(t('cabinet.reviews.deleteConfirm'))) removeMutation.mutate();
}
</script>

<template>
  <div v-if="review.reply && !editing" class="rounded-lg border-l-4 border-brand-500 bg-brand-500/5 px-4 py-3">
    <p class="text-xs font-semibold text-brand-700 dark:text-brand-300">{{ $t('reviews.ownerReply') }}</p>
    <p class="mt-1 whitespace-pre-line text-sm text-text">{{ review.reply.text }}</p>
    <div class="mt-2 flex gap-2">
      <UButton size="xs" color="gray" variant="soft" icon="i-heroicons-pencil-square" @click="editing = true">
        {{ $t('listing.edit') }}
      </UButton>
      <UButton
        size="xs"
        color="gray"
        variant="ghost"
        icon="i-heroicons-trash"
        :loading="removeMutation.isPending.value"
        @click="remove"
      >
        {{ $t('reviews.delete') }}
      </UButton>
    </div>
  </div>
  <form v-else class="flex flex-col gap-2" @submit.prevent="save">
    <UTextarea
      v-model="text"
      :rows="3"
      autoresize
      :maxlength="MAX_LENGTH"
      :placeholder="$t('cabinet.reviews.replyPlaceholder')"
    />
    <p v-if="error" class="text-sm text-brand-600">{{ error }}</p>
    <div class="flex gap-2">
      <UButton type="submit" size="sm" color="primary" :loading="saveMutation.isPending.value">
        {{ $t('cabinet.reviews.publishReply') }}
      </UButton>
      <UButton v-if="review.reply" size="sm" color="gray" variant="ghost" @click="editing = false">
        {{ $t('cabinet.reviews.cancel') }}
      </UButton>
    </div>
  </form>
</template>
