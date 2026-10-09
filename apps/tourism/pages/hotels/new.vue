<script setup lang="ts">
import { useMutation, useQueryClient } from '@tanstack/vue-query';
import HotelForm from '~/components/features/hotel-form/HotelForm.vue';
import type { HotelFormPayload } from '~/components/features/hotel-form/form-payload';
import { useApiClient } from '~/shared/api/client';
import type { Hotel } from '~/shared/api/types';
import { useRequireAuth } from '~/shared/lib/require-auth';
import { slugify } from '~/shared/lib/slugify';

const { t } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const queryClient = useQueryClient();

useRequirePartner();
useSeoMeta({ title: () => t('hotels.newTitle'), robots: 'noindex' });

const submitError = ref<string | null>(null);

const createMutation = useMutation({
  mutationFn: (payload: HotelFormPayload) =>
    api.post<Hotel>('/hotels', { ...payload, slug: slugify(payload.name) }),
  onSuccess: (hotel) => {
    queryClient.invalidateQueries({ queryKey: ['hotels'] });
    queryClient.invalidateQueries({ queryKey: ['my-hotels'] });
    navigateTo(localePath(`/hotels/${hotel.slug}`));
  },
  onError: () => {
    submitError.value = t('listing.form.saveError');
  },
});

function onSubmit(payload: HotelFormPayload): void {
  submitError.value = null;
  createMutation.mutate(payload);
}
</script>

<template>
  <section class="mx-auto max-w-3xl px-4 py-12">
    <h1 class="text-3xl font-extrabold text-text">{{ $t('hotels.newTitle') }}</h1>
    <p class="mt-2 text-text-muted">{{ $t('hotels.newSubtitle') }}</p>
    <HotelForm
      class="mt-8"
      :submitting="createMutation.isPending.value"
      :submit-error="submitError"
      :submit-label="$t('listing.form.publish')"
      @submit="onSubmit"
    />
  </section>
</template>
