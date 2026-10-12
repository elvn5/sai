<script setup lang="ts">
import type { Order } from '~/shared/api/types';
import { formatAmount } from '~/shared/lib/format-price';

// A booking: what, when, how many, how much, and its payment status.
// `forPartner` shows the customer and what's owed instead.
const props = defineProps<{ order: Order; forPartner?: boolean }>();
const { locale } = useI18n();
const localePath = useLocalePath();

const STATUS_CLASS: Record<Order['status'], string> = {
  pending: 'bg-amber-500/15 text-amber-700 dark:text-amber-300',
  paid: 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-300',
  failed: 'bg-red-500/15 text-red-700 dark:text-red-300',
  expired: 'bg-gray-500/15 text-text-muted',
  refunded: 'bg-sky-500/15 text-sky-700 dark:text-sky-300',
};

const tourPath = computed(() => {
  if (!props.order.itemSlug) return null;
  const prefix =
    props.order.itemLocale && props.order.itemLocale !== 'ru' ? `/${props.order.itemLocale}` : '';
  return `${prefix}/tours/${props.order.itemSlug}`;
});
const date = computed(() =>
  props.order.itemDate
    ? new Date(`${props.order.itemDate}T12:00:00`).toLocaleDateString(locale.value, {
        day: 'numeric',
        month: 'long',
      })
    : null,
);
</script>

<template>
  <div class="flex flex-col gap-2 rounded-card border border-border bg-surface px-4 py-3">
    <div class="flex items-start justify-between gap-3">
      <div class="min-w-0">
        <NuxtLink v-if="tourPath" :to="tourPath" class="block truncate font-semibold text-text hover:underline">{{ order.itemTitle }}</NuxtLink>
        <span v-else class="block truncate font-semibold text-text">{{ order.itemTitle }}</span>
        <p class="text-xs text-text-muted">
          <template v-if="date">{{ date }} · </template>{{ $t('booking.placesCount', { count: order.quantity }) }}
        </p>
      </div>
      <span class="shrink-0 rounded-full px-2.5 py-1 text-xs font-semibold" :class="STATUS_CLASS[order.status]">
        {{ $t(`booking.status.${order.status}`) }}
      </span>
    </div>
    <div class="flex flex-wrap items-center justify-between gap-2 text-sm">
      <span v-if="forPartner" class="text-text">
        {{ order.contactName }} · <a :href="`tel:${order.contactPhone}`" class="text-brand-600 hover:underline dark:text-brand-400">{{ order.contactPhone }}</a>
      </span>
      <span v-else class="text-text-muted">{{ new Date(order.createdAt).toLocaleDateString(locale) }}</span>
      <span class="font-bold text-text">
        {{ formatAmount(locale, forPartner ? order.payeeAmount : order.amount) }} {{ $t('listing.currency') }}
      </span>
    </div>
    <p v-if="forPartner && order.comment" class="text-sm italic text-text-muted">«{{ order.comment }}»</p>
    <UButton
      v-if="!forPartner && order.status === 'pending' && order.paymentUrl"
      size="xs"
      color="primary"
      class="self-start"
      :to="order.paymentUrl"
      external
    >
      {{ $t('booking.continuePayment') }}
    </UButton>
    <NuxtLink v-if="!forPartner && order.status === 'pending'" :to="localePath({ path: '/payments/return', query: { order: order.id } })" class="text-xs text-text-muted underline">
      {{ $t('booking.checkStatus') }}
    </NuxtLink>
  </div>
</template>
