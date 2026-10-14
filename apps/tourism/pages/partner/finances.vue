<script setup lang="ts">
import OrderCard from '~/components/entities/order/OrderCard.vue';
import StatTile from '~/components/shared/ui/StatTile.vue';
import type { Order } from '~/shared/api/types';
import { formatAmount } from '~/shared/lib/format-price';
import { usePartnerFinances, useReceivedOrders } from '~/shared/lib/partner-cabinet';
import { useRequirePartner } from '~/shared/lib/require-auth';

definePageMeta({ layout: 'partner' });

const { t, locale } = useI18n();

useRequirePartner();
useSeoMeta({ title: () => t('cabinet.nav.finances'), robots: 'noindex' });

const financesQuery = usePartnerFinances();
const ordersQuery = useReceivedOrders();
const finances = computed(() => financesQuery.data.value ?? null);

// Paid bookings the platform still owes the partner for.
const owedOrders = computed(() =>
  (ordersQuery.data.value ?? []).filter((o: Order) => o.status === 'paid' && !o.payoutId),
);

const money = (amount: number): string =>
  `${formatAmount(locale.value, amount)} ${t('listing.currency')}`;

function formatDate(value: string): string {
  return new Date(value).toLocaleDateString(locale.value, {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  });
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <div>
      <h1 class="text-2xl font-extrabold text-text sm:text-3xl">{{ $t('cabinet.nav.finances') }}</h1>
      <p class="mt-1 text-sm text-text-muted">{{ $t('cabinet.finances.subtitle') }}</p>
    </div>

    <div class="grid grid-cols-2 gap-3 lg:grid-cols-3">
      <StatTile
        :label="$t('cabinet.finances.owed')"
        icon="i-heroicons-banknotes"
        :value="finances ? money(finances.owed) : '—'"
        :hint="finances ? $t('cabinet.finances.owedOrders', { count: finances.owedOrders }) : null"
      />
      <StatTile
        :label="$t('cabinet.finances.paidOut')"
        icon="i-heroicons-check-badge"
        :value="finances ? money(finances.paidOut) : '—'"
        :hint="finances ? $t('cabinet.finances.payoutsCount', { count: finances.payouts.length }) : null"
      />
      <StatTile
        class="col-span-2 lg:col-span-1"
        :label="$t('cabinet.finances.monthEarned')"
        icon="i-heroicons-arrow-trending-up"
        :value="finances ? money(finances.month.earned) : '—'"
        :hint="finances ? $t('cabinet.dashboard.guests', { count: finances.month.guests }) : null"
      />
    </div>

    <section v-if="finances" class="flex flex-col gap-3">
      <h2 class="font-semibold text-text">{{ $t('cabinet.finances.monthHeading') }}</h2>
      <dl class="flex flex-col divide-y divide-border rounded-card border border-border bg-surface text-sm">
        <div class="flex justify-between gap-3 px-4 py-3">
          <dt class="text-text-muted">{{ $t('cabinet.finances.revenue') }}</dt>
          <dd class="tabular-nums text-text">{{ money(finances.month.revenue) }}</dd>
        </div>
        <div class="flex justify-between gap-3 px-4 py-3">
          <dt class="text-text-muted">{{ $t('cabinet.finances.fee') }}</dt>
          <dd class="tabular-nums text-text">−{{ money(finances.month.fee) }}</dd>
        </div>
        <div class="flex justify-between gap-3 px-4 py-3 font-semibold">
          <dt class="text-text">{{ $t('cabinet.finances.earned') }}</dt>
          <dd class="tabular-nums text-text">{{ money(finances.month.earned) }}</dd>
        </div>
      </dl>
      <p class="text-xs text-text-muted">{{ $t('cabinet.finances.howPaid') }}</p>
    </section>

    <section class="flex flex-col gap-3">
      <h2 class="font-semibold text-text">{{ $t('cabinet.finances.awaiting') }}</h2>
      <div v-if="owedOrders.length > 0" class="flex flex-col gap-2">
        <OrderCard v-for="order in owedOrders" :key="order.id" :order="order" for-partner />
      </div>
      <p v-else class="rounded-card border border-dashed border-border bg-surface px-4 py-6 text-center text-sm text-text-muted">
        {{ $t('cabinet.finances.nothingOwed') }}
      </p>
    </section>

    <section class="flex flex-col gap-3">
      <h2 class="font-semibold text-text">{{ $t('cabinet.finances.history') }}</h2>
      <div v-if="finances && finances.payouts.length > 0" class="flex flex-col divide-y divide-border rounded-card border border-border bg-surface">
        <div v-for="payout in finances.payouts" :key="payout.id" class="flex items-center justify-between gap-3 px-4 py-3">
          <div class="min-w-0">
            <p class="font-medium text-text">{{ formatDate(payout.createdAt) }}</p>
            <p class="truncate text-xs text-text-muted">
              {{ $t('cabinet.dashboard.bookingsCount', { count: payout.orders }) }}<template v-if="payout.note"> · {{ payout.note }}</template>
            </p>
          </div>
          <span class="shrink-0 font-bold tabular-nums text-text">{{ money(payout.amount) }}</span>
        </div>
      </div>
      <p v-else class="rounded-card border border-dashed border-border bg-surface px-4 py-6 text-center text-sm text-text-muted">
        {{ $t('cabinet.finances.noPayouts') }}
      </p>
    </section>
  </div>
</template>
