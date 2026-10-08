<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import HotelForm from '~/components/features/hotel-form/HotelForm.vue';
import type { HotelFormPayload } from '~/components/features/hotel-form/form-payload';
import { useApiClient } from '~/shared/api/client';
import type { Hotel } from '~/shared/api/types';
import { useRequireAuth } from '~/shared/lib/require-auth';

const route = useRoute();
const { t } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const auth = useAuth();
const queryClient = useQueryClient();

const slug = route.params.slug as string;

useRequireAuth();
useSeoMeta({ title: () => t('hotels.editTitle'), robots: 'noindex' });

const hotelQuery = useQuery({
  queryKey: ['hotel', slug],
  queryFn: () => api.get<Hotel>(`/hotels/${slug}`),
});
await hotelQuery.suspense();
const hotel = hotelQuery.data;

if (!hotel.value) {
  throw createError({ statusCode: 404, statusMessage: 'Hotel not found' });
}

const isOwner = computed(() =>
  Boolean(auth.user.value && hotel.value?.userId === auth.user.value.id),
);
const submitError = ref<string | null>(null);

const updateMutation = useMutation({
  mutationFn: (payload: HotelFormPayload) => api.put<Hotel>(`/hotels/${hotel.value?.id}`, payload),
  onSuccess: (updated) => {
    queryClient.setQueryData(['hotel', slug], updated);
    queryClient.invalidateQueries({ queryKey: ['hotels'] });
    queryClient.invalidateQueries({ queryKey: ['my-hotels'] });
    navigateTo(localePath(`/hotels/${updated.slug}`));
  },
  onError: () => {
    submitError.value = t('listing.form.saveError');
  },
});

function onSubmit(payload: HotelFormPayload): void {
  submitError.value = null;
  updateMutation.mutate(payload);
}
</script>

<template>
  <section class="mx-auto max-w-3xl px-4 py-12">
    <NuxtLink :to="localePath(`/hotels/${slug}`)" class="text-sm text-primary-600 hover:underline dark:text-primary-400">
      &larr; {{ hotel?.name }}
    </NuxtLink>
    <h1 class="mt-2 text-3xl font-extrabold text-text">{{ $t('hotels.editTitle') }}</h1>
    <HotelForm
      v-if="hotel && isOwner"
      class="mt-8"
      :initial="hotel"
      :submitting="updateMutation.isPending.value"
      :submit-error="submitError"
      :submit-label="$t('listing.form.save')"
      @submit="onSubmit"
    />
    <p v-else-if="auth.checked.value" class="mt-8 text-text-muted">{{ $t('listing.notOwner') }}</p>
  </section>
</template>
