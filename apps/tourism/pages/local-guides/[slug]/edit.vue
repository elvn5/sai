<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import LocalGuideForm from '~/components/features/local-guide-form/LocalGuideForm.vue';
import type { LocalGuideFormPayload } from '~/components/features/local-guide-form/form-payload';
import { useApiClient } from '~/shared/api/client';
import type { LocalGuide } from '~/shared/api/types';
import { useRequireAuth } from '~/shared/lib/require-auth';

const route = useRoute();
const { t } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const auth = useAuth();
const queryClient = useQueryClient();

const slug = route.params.slug as string;

useRequireAuth();
useSeoMeta({ title: () => t('localGuides.editTitle'), robots: 'noindex' });

const guideQuery = useQuery({
  queryKey: ['local-guide', slug],
  queryFn: () => api.get<LocalGuide>(`/local-guides/${slug}`),
});
await guideQuery.suspense();
const guide = guideQuery.data;

if (!guide.value) {
  throw createError({ statusCode: 404, statusMessage: 'Guide not found' });
}

const isOwner = computed(() =>
  Boolean(auth.user.value && guide.value?.userId === auth.user.value.id),
);
const submitError = ref<string | null>(null);

const updateMutation = useMutation({
  mutationFn: (payload: LocalGuideFormPayload) =>
    api.put<LocalGuide>(`/local-guides/${guide.value?.id}`, payload),
  onSuccess: (updated) => {
    queryClient.setQueryData(['local-guide', slug], updated);
    queryClient.setQueryData(['my-local-guide'], { guide: updated });
    queryClient.invalidateQueries({ queryKey: ['local-guides'] });
    navigateTo(localePath(`/local-guides/${updated.slug}`));
  },
  onError: () => {
    submitError.value = t('listing.form.saveError');
  },
});

function onSubmit(payload: LocalGuideFormPayload): void {
  submitError.value = null;
  updateMutation.mutate(payload);
}
</script>

<template>
  <section class="mx-auto max-w-3xl px-4 py-12">
    <NuxtLink :to="localePath(`/local-guides/${slug}`)" class="text-sm text-primary-600 hover:underline dark:text-primary-400">
      &larr; {{ guide?.name }}
    </NuxtLink>
    <h1 class="mt-2 text-3xl font-extrabold text-text">{{ $t('localGuides.editTitle') }}</h1>
    <LocalGuideForm
      v-if="guide && isOwner"
      class="mt-8"
      :initial="guide"
      :submitting="updateMutation.isPending.value"
      :submit-error="submitError"
      :submit-label="$t('listing.form.save')"
      @submit="onSubmit"
    />
    <p v-else-if="auth.checked.value" class="mt-8 text-text-muted">{{ $t('listing.notOwner') }}</p>
  </section>
</template>
