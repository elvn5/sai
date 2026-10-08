<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import LocalGuideForm from '~/components/features/local-guide-form/LocalGuideForm.vue';
import type { LocalGuideFormPayload } from '~/components/features/local-guide-form/form-payload';
import { useApiClient } from '~/shared/api/client';
import type { LocalGuide } from '~/shared/api/types';
import { useRequireAuth } from '~/shared/lib/require-auth';
import { slugify } from '~/shared/lib/slugify';

const { t } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const auth = useAuth();
const queryClient = useQueryClient();

useRequireAuth();
useSeoMeta({ title: () => t('localGuides.newTitle'), robots: 'noindex' });

// One profile per user — send people who already have one to it.
const myGuideQuery = useQuery({
  queryKey: ['my-local-guide'],
  queryFn: () => api.get<{ guide: LocalGuide | null }>('/local-guides/mine'),
  enabled: computed(() => Boolean(auth.user.value)),
});

const defaultName = computed(() =>
  [auth.user.value?.firstName, auth.user.value?.lastName].filter(Boolean).join(' '),
);
const submitError = ref<string | null>(null);
// Set before the new profile lands in the cache, so the redirect below
// doesn't mistake it for a pre-existing one.
let justCreated = false;

const createMutation = useMutation({
  mutationFn: (payload: LocalGuideFormPayload) =>
    api.post<LocalGuide>('/local-guides', { ...payload, slug: slugify(payload.name) }),
  onSuccess: (guide) => {
    justCreated = true;
    queryClient.invalidateQueries({ queryKey: ['local-guides'] });
    queryClient.setQueryData(['my-local-guide'], { guide });
    navigateTo(localePath(`/local-guides/${guide.slug}`));
  },
  onError: () => {
    submitError.value = t('listing.form.saveError');
  },
});

watch(
  () => myGuideQuery.data.value?.guide,
  (existing: LocalGuide | null | undefined) => {
    if (existing && !justCreated)
      void navigateTo(localePath(`/local-guides/${existing.slug}/edit`));
  },
  { immediate: true },
);

function onSubmit(payload: LocalGuideFormPayload): void {
  submitError.value = null;
  createMutation.mutate(payload);
}
</script>

<template>
  <section class="mx-auto max-w-3xl px-4 py-12">
    <h1 class="text-3xl font-extrabold text-text">{{ $t('localGuides.newTitle') }}</h1>
    <p class="mt-2 text-text-muted">{{ $t('localGuides.newSubtitle') }}</p>
    <LocalGuideForm
      v-if="auth.user.value"
      :key="defaultName"
      class="mt-8"
      :default-name="defaultName"
      :submitting="createMutation.isPending.value"
      :submit-error="submitError"
      :submit-label="$t('listing.form.publish')"
      @submit="onSubmit"
    />
  </section>
</template>
