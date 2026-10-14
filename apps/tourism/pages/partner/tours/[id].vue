<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import TourForm from '~/components/features/tour-form/TourForm.vue';
import type { TourFormPayload } from '~/components/features/tour-form/form-payload';
import { useApiClient } from '~/shared/api/client';
import type { Tour } from '~/shared/api/types';
import { useRequirePartner } from '~/shared/lib/require-auth';

definePageMeta({ layout: 'partner' });

const route = useRoute();
const { t } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const auth = useAuth();
const queryClient = useQueryClient();

useRequirePartner();
useSeoMeta({ title: () => t('cabinet.listings.editTour'), robots: 'noindex' });

// A partner edits only their own tours, so their list is the source.
const myToursQuery = useQuery({
  queryKey: ['tours', 'mine'],
  queryFn: () => api.get<Tour[]>('/tours/mine'),
  enabled: auth.isPartner,
});
const tour = computed(
  () => myToursQuery.data.value?.find((item: Tour) => item.id === route.params.id) ?? null,
);
const submitError = ref<string | null>(null);

const updateMutation = useMutation({
  mutationFn: (payload: TourFormPayload) => api.put<Tour>(`/tours/${tour.value?.id}`, payload),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ['tours'] });
    navigateTo(localePath('/partner/listings'));
  },
  onError: () => {
    submitError.value = t('listing.form.saveError');
  },
});

function onSubmit(payload: TourFormPayload): void {
  submitError.value = null;
  updateMutation.mutate(payload);
}
</script>

<template>
  <div class="flex max-w-2xl flex-col gap-2">
    <NuxtLink :to="localePath('/partner/listings')" class="text-sm text-primary-600 hover:underline dark:text-primary-400">
      &larr; {{ $t('cabinet.nav.listings') }}
    </NuxtLink>
    <h1 class="text-2xl font-extrabold text-text sm:text-3xl">{{ $t('cabinet.listings.editTour') }}</h1>
    <TourForm
      v-if="tour"
      :key="tour.id"
      class="mt-6"
      :initial="tour"
      :submitting="updateMutation.isPending.value"
      :submit-error="submitError"
      :submit-label="$t('listing.form.save')"
      @submit="onSubmit"
    />
    <p v-else-if="myToursQuery.isFetched.value" class="mt-6 text-text-muted">{{ $t('listing.notOwner') }}</p>
  </div>
</template>
