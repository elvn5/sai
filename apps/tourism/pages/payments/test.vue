<script setup lang="ts">
import { useMutation, useQuery } from '@tanstack/vue-query';
import { useApiClient } from '~/shared/api/client';
import type { Order } from '~/shared/api/types';
import { formatAmount } from '~/shared/lib/format-price';

// Stands in for the bank's payment page while payments run in test mode
// (PAYMENTS_PROVIDER=fake): no money moves.
const { t, locale } = useI18n();
const route = useRoute();
const api = useApiClient();
const id = typeof route.query.order === 'string' ? route.query.order : '';

useRequireAuth();
const orderQuery = useQuery({
  queryKey: ['order', id],
  queryFn: () => api.get<Order>(`/orders/${id}`),
  enabled: computed(() => Boolean(id) && import.meta.client),
});

const settle = useMutation({
  mutationFn: (success: boolean) =>
    api.post<{ returnUrl: string }>(`/payments/test/${id}`, { success }),
  onSuccess: (res) => {
    window.location.href = res.returnUrl;
  },
});

useSeoMeta({ title: () => t('booking.test.title'), robots: 'noindex' });
</script>

<template>
  <section class="mx-auto max-w-md px-4 py-16">
    <div class="rounded-card border-2 border-dashed border-amber-400 bg-surface p-6 text-center">
      <p class="mb-4 rounded-lg bg-amber-400/15 px-3 py-2 text-sm font-semibold text-amber-700 dark:text-amber-300">
        {{ $t('booking.test.banner') }}
      </p>
      <template v-if="orderQuery.data.value">
        <p class="text-text-muted">{{ orderQuery.data.value.itemTitle }}</p>
        <p class="my-3 text-4xl font-black text-text">
          {{ formatAmount(locale, orderQuery.data.value.amount) }} {{ $t('listing.currency') }}
        </p>
        <div class="mt-6 flex flex-col gap-2">
          <UButton color="primary" size="lg" block :loading="settle.isPending.value" @click="settle.mutate(true)">{{ $t('booking.test.pay') }}</UButton>
          <UButton color="gray" variant="soft" block :disabled="settle.isPending.value" @click="settle.mutate(false)">{{ $t('booking.test.decline') }}</UButton>
        </div>
      </template>
      <p v-else-if="orderQuery.isError.value" class="text-text-muted">{{ $t('booking.result.notFound') }}</p>
    </div>
  </section>
</template>
