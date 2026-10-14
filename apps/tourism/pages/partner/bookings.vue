<script setup lang="ts">
import OrderCard from '~/components/entities/order/OrderCard.vue';
import type { Order } from '~/shared/api/types';
import { bishkekToday, downloadCsv, useReceivedOrders } from '~/shared/lib/partner-cabinet';
import { useRequirePartner } from '~/shared/lib/require-auth';

definePageMeta({ layout: 'partner' });

const { t } = useI18n();

useRequirePartner();
useSeoMeta({ title: () => t('cabinet.nav.bookings'), robots: 'noindex' });

type When = 'upcoming' | 'past' | 'all';

const ordersQuery = useReceivedOrders();
const when = ref<When>('upcoming');
const showRefunded = ref(false);
const search = ref('');

const whenOptions = computed(() => [
  { value: 'upcoming', label: t('cabinet.bookings.upcoming') },
  { value: 'past', label: t('cabinet.bookings.past') },
  { value: 'all', label: t('cabinet.bookings.all') },
]);

const orders = computed(() => {
  const today = bishkekToday();
  const needle = search.value.trim().toLowerCase();
  const list = (ordersQuery.data.value ?? []).filter((order: Order) => {
    if (!showRefunded.value && order.status === 'refunded') return false;
    const date = order.itemDate ?? '';
    if (when.value === 'upcoming' && date < today) return false;
    if (when.value === 'past' && date >= today) return false;
    if (!needle) return true;
    return [order.contactName, order.contactPhone, order.itemTitle].some((field) =>
      field.toLowerCase().includes(needle),
    );
  });
  // Upcoming trips soonest first; the rest newest first.
  return when.value === 'upcoming'
    ? list.sort((a: Order, b: Order) => (a.itemDate ?? '').localeCompare(b.itemDate ?? ''))
    : list;
});

const guests = computed(() =>
  orders.value
    .filter((o: Order) => o.status === 'paid')
    .reduce((sum: number, o: Order) => sum + o.quantity, 0),
);

function exportCsv(): void {
  downloadCsv(`ermekten-bookings-${bishkekToday()}.csv`, [
    [
      t('cabinet.bookings.tour'),
      t('cabinet.bookings.date'),
      t('cabinet.bookings.guests'),
      t('cabinet.bookings.name'),
      t('cabinet.bookings.phone'),
      t('cabinet.bookings.comment'),
      t('cabinet.bookings.paid'),
      t('cabinet.bookings.earned'),
      t('cabinet.bookings.status'),
    ],
    ...orders.value.map((o: Order) => [
      o.itemTitle,
      o.itemDate,
      o.quantity,
      o.contactName,
      o.contactPhone,
      o.comment,
      o.amount,
      o.payeeAmount,
      t(`booking.status.${o.status}`),
    ]),
  ]);
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <div class="flex flex-wrap items-end justify-between gap-3">
      <div>
        <h1 class="text-2xl font-extrabold text-text sm:text-3xl">{{ $t('cabinet.nav.bookings') }}</h1>
        <p class="mt-1 text-sm text-text-muted">{{ $t('cabinet.bookings.subtitle') }}</p>
      </div>
      <UButton color="gray" icon="i-heroicons-arrow-down-tray" :disabled="orders.length === 0" @click="exportCsv">
        CSV
      </UButton>
    </div>

    <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
      <USelect v-model="when" :options="whenOptions" option-attribute="label" class="sm:w-48" />
      <UInput
        v-model="search"
        icon="i-heroicons-magnifying-glass"
        :placeholder="$t('cabinet.bookings.search')"
        class="flex-1"
      />
      <UCheckbox v-model="showRefunded" :label="$t('cabinet.bookings.showRefunded')" />
    </div>

    <p v-if="orders.length > 0" class="text-sm text-text-muted">
      {{ $t('cabinet.bookings.summary', { bookings: orders.length, guests }) }}
    </p>

    <div v-if="orders.length > 0" class="flex flex-col gap-2">
      <OrderCard v-for="order in orders" :key="order.id" :order="order" for-partner />
    </div>
    <div
      v-else-if="!ordersQuery.isLoading.value"
      class="flex flex-col items-center gap-2 rounded-card border border-dashed border-border bg-surface px-6 py-12 text-center"
    >
      <UIcon name="i-mdi-ticket-confirmation-outline" class="h-10 w-10 text-text-muted" />
      <p class="font-medium text-text">{{ $t('booking.noReceived') }}</p>
      <p class="max-w-sm text-sm text-text-muted">{{ $t('cabinet.bookings.emptyHint') }}</p>
    </div>
  </div>
</template>
