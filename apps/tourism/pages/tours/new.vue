<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import TourForm from '~/components/features/tour-form/TourForm.vue';
import type { TourFormPayload } from '~/components/features/tour-form/form-payload';
import { ApiError, useApiClient } from '~/shared/api/client';
import type { Tour } from '~/shared/api/types';
import { useRequirePartner } from '~/shared/lib/require-auth';
import { slugify } from '~/shared/lib/slugify';

// ?from=<tour id> duplicates one of the partner's tours for new dates.
const route = useRoute();
const { t, locale } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const auth = useAuth();
const queryClient = useQueryClient();

const sourceId = computed(() => (typeof route.query.from === 'string' ? route.query.from : null));
const submitError = ref<string | null>(null);

useSeoMeta({
  title: () => (sourceId.value ? t('cabinet.listings.duplicateTitle') : t('tours.newTitle')),
  robots: 'noindex',
});

useRequirePartner();

const myToursQuery = useQuery({
  queryKey: ['tours', 'mine'],
  queryFn: () => api.get<Tour[]>('/tours/mine'),
  enabled: computed(() => Boolean(sourceId.value) && auth.isPartner.value),
});
const source = computed<Partial<Tour> | null>(() => {
  const tour = myToursQuery.data.value?.find((item: Tour) => item.id === sourceId.value);
  return tour ? { ...tour, startDate: '', endDate: '' } : null;
});
const ready = computed(() => !sourceId.value || myToursQuery.isFetched.value);

const createMutation = useMutation({
  mutationFn: (payload: TourFormPayload) =>
    api.post<Tour>('/tours', {
      ...payload,
      locale: locale.value,
      // A copy shares its title, so its dates keep the address unique.
      slug: slugify(sourceId.value ? `${payload.title} ${payload.startDate}` : payload.title),
    }),
  onSuccess: (tour) => {
    queryClient.invalidateQueries({ queryKey: ['tours'] });
    navigateTo(localePath(sourceId.value ? '/partner/listings' : `/tours/${tour.slug}`));
  },
  onError: (error) => {
    submitError.value =
      error instanceof ApiError && error.status === 409
        ? t('cabinet.listings.slugTaken')
        : t('tours.createError');
  },
});

function onSubmit(payload: TourFormPayload): void {
  submitError.value = null;
  createMutation.mutate(payload);
}
</script>

<template>
  <section class="mx-auto max-w-2xl px-4 py-12">
    <NuxtLink
      v-if="auth.isPartner.value"
      :to="localePath('/partner/listings')"
      class="text-sm text-primary-600 hover:underline dark:text-primary-400"
    >
      &larr; {{ $t('cabinet.title') }}
    </NuxtLink>
    <h1 class="mt-2 text-3xl font-extrabold text-text">
      {{ sourceId ? $t('cabinet.listings.duplicateTitle') : $t('tours.newTitle') }}
    </h1>
    <p v-if="sourceId" class="mt-2 text-sm text-text-muted">{{ $t('cabinet.listings.duplicateHint') }}</p>

    <TourForm
      v-if="ready"
      :key="source?.id ?? 'new'"
      class="mt-8"
      :initial="source"
      :submitting="createMutation.isPending.value"
      :submit-error="submitError"
      :submit-label="$t('tours.submit')"
      @submit="onSubmit"
    />
  </section>
</template>
