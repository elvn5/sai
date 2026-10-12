<script setup lang="ts">
import OrderCard from '~/components/entities/order/OrderCard.vue';
import { useApiClient } from '~/shared/api/client';
import type { Order } from '~/shared/api/types';

// Where the bank sends the customer back: checks the payment right away,
// then keeps checking for a while (the bank can take a moment to settle).
const POLL_MS = 3000;
const MAX_POLLS = 40;

const { t } = useI18n();
const route = useRoute();
const localePath = useLocalePath();
const api = useApiClient();
const id = typeof route.query.order === 'string' ? route.query.order : '';

const order = ref<Order | null>(null);
const failed = ref(false);
let polls = 0;
let timer: ReturnType<typeof setTimeout> | undefined;

async function check(): Promise<void> {
  try {
    order.value = await api.post<Order>(`/orders/${id}/check`);
  } catch {
    failed.value = true;
    return;
  }
  if (order.value.status === 'pending' && ++polls < MAX_POLLS) {
    timer = setTimeout(check, POLL_MS);
  }
}

useRequireAuth();
onMounted(() => {
  if (id) void check();
  else failed.value = true;
});
onBeforeUnmount(() => clearTimeout(timer));

const view = computed(() => {
  switch (order.value?.status) {
    case 'paid':
      return {
        icon: 'i-mdi-check-circle',
        color: 'text-emerald-500',
        title: t('booking.result.paid'),
        text: t('booking.result.paidText'),
      };
    case 'failed':
      return {
        icon: 'i-mdi-close-circle',
        color: 'text-red-500',
        title: t('booking.result.failed'),
        text: t('booking.result.failedText'),
      };
    case 'expired':
      return {
        icon: 'i-mdi-timer-off-outline',
        color: 'text-text-muted',
        title: t('booking.result.expired'),
        text: t('booking.result.failedText'),
      };
    default:
      return {
        icon: 'i-mdi-loading',
        color: 'text-brand-500 animate-spin',
        title: t('booking.result.pending'),
        text: t('booking.result.pendingText'),
      };
  }
});

useSeoMeta({ title: () => t('booking.result.title'), robots: 'noindex' });
</script>

<template>
  <section class="mx-auto flex max-w-lg flex-col items-center gap-5 px-4 py-16 text-center">
    <template v-if="failed">
      <UIcon name="i-mdi-alert-circle-outline" class="h-16 w-16 text-text-muted" />
      <h1 class="text-2xl font-extrabold text-text">{{ $t('booking.result.notFound') }}</h1>
    </template>
    <template v-else>
      <UIcon :name="view.icon" class="h-16 w-16" :class="view.color" />
      <h1 class="text-2xl font-extrabold text-text">{{ view.title }}</h1>
      <p class="text-text-muted">{{ view.text }}</p>
      <OrderCard v-if="order" :order="order" class="w-full text-left" />
    </template>
    <UButton color="primary" :to="localePath({ path: '/profile', query: { tab: 'bookings' } })">{{ $t('booking.myBookings') }}</UButton>
  </section>
</template>
