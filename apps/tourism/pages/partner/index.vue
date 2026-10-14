<script setup lang="ts">
import DailyBarChart from '~/components/shared/ui/DailyBarChart.vue';
import RatingStars from '~/components/shared/ui/RatingStars.vue';
import StatTile from '~/components/shared/ui/StatTile.vue';
import type { DailyStats, ListingStats, Order } from '~/shared/api/types';
import { formatAmount } from '~/shared/lib/format-price';
import { listingPath } from '~/shared/lib/listing-path';
import {
  bishkekToday,
  percentChange,
  shareOf,
  usePartnerFinances,
  usePartnerReviews,
  usePartnerStats,
  useReceivedOrders,
} from '~/shared/lib/partner-cabinet';
import { useRequirePartner } from '~/shared/lib/require-auth';

definePageMeta({ layout: 'partner' });

const { t, locale } = useI18n();
const localePath = useLocalePath();
const auth = useAuth();

useRequirePartner();
useSeoMeta({ title: () => t('cabinet.title'), robots: 'noindex' });

const days = ref(30);
const statsQuery = usePartnerStats(days);
const financesQuery = usePartnerFinances();
const ordersQuery = useReceivedOrders();
const reviewsQuery = usePartnerReviews(ref(true), 3);

const stats = computed(() => statsQuery.data.value ?? null);
const finances = computed(() => financesQuery.data.value ?? null);
const newReviews = computed(() => reviewsQuery.data.value?.items ?? []);
const unansweredTotal = computed(() => reviewsQuery.data.value?.total ?? 0);

const money = (amount: number): string =>
  `${formatAmount(locale.value, amount)} ${t('listing.currency')}`;

const viewsPoints = computed(
  () => stats.value?.daily.map((d: DailyStats) => ({ day: d.day, value: d.views })) ?? [],
);
const contactsPoints = computed(
  () => stats.value?.daily.map((d: DailyStats) => ({ day: d.day, value: d.contacts })) ?? [],
);

const topListings = computed(() =>
  [...(stats.value?.listings ?? [])]
    .filter((l: ListingStats) => l.views > 0 || l.contacts > 0)
    .sort((a: ListingStats, b: ListingStats) => b.views - a.views)
    .slice(0, 5),
);

// Upcoming departures: paid bookings grouped by tour and date.
const departures = computed(() => {
  const today = bishkekToday();
  const groups = new Map<
    string,
    { title: string; date: string; guests: number; bookings: number }
  >();
  for (const order of ordersQuery.data.value ?? []) {
    if (order.status !== 'paid' || !order.itemDate || order.itemDate < today) continue;
    const key = `${order.itemId}|${order.itemDate}`;
    const group = groups.get(key) ?? {
      title: order.itemTitle,
      date: order.itemDate,
      guests: 0,
      bookings: 0,
    };
    group.guests += order.quantity;
    group.bookings += 1;
    groups.set(key, group);
  }
  return [...groups.values()].sort((a, b) => a.date.localeCompare(b.date)).slice(0, 5);
});

const recentOrders = computed(() =>
  (ordersQuery.data.value ?? []).filter((o: Order) => o.status === 'paid').slice(0, 3),
);

const hasListings = computed(() => (stats.value?.listings.length ?? 0) > 0);
const hasTelegram = computed(() => Boolean(auth.user.value?.telegramId));
const onboarding = computed(() => [
  { done: hasListings.value, label: t('cabinet.onboarding.listing'), to: '/partner/listings' },
  { done: hasTelegram.value, label: t('cabinet.onboarding.telegram'), to: '/profile' },
]);
const onboardingLeft = computed(
  () => onboarding.value.filter((step: { done: boolean }) => !step.done).length,
);

function formatDate(day: string): string {
  return new Date(`${day}T12:00:00`).toLocaleDateString(locale.value, {
    day: 'numeric',
    month: 'long',
  });
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <div>
      <h1 class="text-2xl font-extrabold text-text sm:text-3xl">{{ $t('cabinet.title') }}</h1>
      <p class="mt-1 text-sm text-text-muted">{{ $t('cabinet.dashboard.subtitle', { days }) }}</p>
    </div>

    <div v-if="stats && onboardingLeft > 0" class="rounded-card border border-brand-500/30 bg-brand-500/5 px-4 py-4">
      <p class="font-semibold text-text">{{ $t('cabinet.onboarding.title') }}</p>
      <ul class="mt-3 flex flex-col gap-2">
        <li v-for="step in onboarding" :key="step.label" class="flex items-center gap-2 text-sm">
          <UIcon
            :name="step.done ? 'i-heroicons-check-circle-solid' : 'i-heroicons-minus-circle'"
            class="h-5 w-5 shrink-0"
            :class="step.done ? 'text-emerald-600 dark:text-emerald-400' : 'text-text-muted'"
          />
          <span v-if="step.done" class="text-text-muted line-through">{{ step.label }}</span>
          <NuxtLink v-else :to="localePath(step.to)" class="font-medium text-text hover:underline">{{ step.label }}</NuxtLink>
        </li>
      </ul>
    </div>

    <div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
      <StatTile
        :label="$t('cabinet.stats.views')"
        icon="i-heroicons-eye"
        :value="stats ? formatAmount(locale, stats.totals.views) : '—'"
        :delta="stats ? percentChange(stats.totals.views, stats.previous.views) : null"
        :hint="$t('cabinet.dashboard.vsPrevious')"
      />
      <StatTile
        :label="$t('cabinet.stats.contacts')"
        icon="i-heroicons-chat-bubble-left-ellipsis"
        :value="stats ? formatAmount(locale, stats.totals.contacts) : '—'"
        :delta="stats ? percentChange(stats.totals.contacts, stats.previous.contacts) : null"
        :hint="stats ? $t('cabinet.stats.conversionHint', { value: shareOf(stats.totals.contacts, stats.totals.views) }) : null"
      />
      <StatTile
        :label="$t('cabinet.dashboard.monthBookings')"
        icon="i-mdi-ticket-confirmation-outline"
        :value="finances ? String(finances.month.orders) : '—'"
        :hint="finances ? $t('cabinet.dashboard.guests', { count: finances.month.guests }) : null"
      />
      <StatTile
        :label="$t('cabinet.finances.owed')"
        icon="i-heroicons-banknotes"
        :value="finances ? money(finances.owed) : '—'"
        :hint="finances ? $t('cabinet.finances.owedOrders', { count: finances.owedOrders }) : null"
      />
    </div>

    <div class="grid grid-cols-1 gap-6 rounded-card border border-border bg-surface p-4 sm:grid-cols-2">
      <DailyBarChart :title="$t('cabinet.stats.viewsPerDay')" :points="viewsPoints" />
      <DailyBarChart :title="$t('cabinet.stats.contactsPerDay')" :points="contactsPoints" />
      <p class="text-xs text-text-muted sm:col-span-2">{{ $t('cabinet.stats.howCounted') }}</p>
    </div>

    <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
      <section class="flex flex-col gap-3">
        <div class="flex items-center justify-between">
          <h2 class="font-semibold text-text">{{ $t('cabinet.dashboard.departures') }}</h2>
          <NuxtLink :to="localePath('/partner/bookings')" class="text-sm text-primary-600 hover:underline dark:text-primary-400">
            {{ $t('cabinet.dashboard.allBookings') }}
          </NuxtLink>
        </div>
        <div v-if="departures.length > 0" class="flex flex-col gap-2">
          <div
            v-for="departure in departures"
            :key="`${departure.title}-${departure.date}`"
            class="flex items-center justify-between gap-3 rounded-card border border-border bg-surface px-4 py-3"
          >
            <div class="min-w-0">
              <p class="truncate font-medium text-text">{{ departure.title }}</p>
              <p class="text-xs text-text-muted">{{ formatDate(departure.date) }}</p>
            </div>
            <div class="shrink-0 text-right">
              <p class="font-bold tabular-nums text-text">{{ $t('cabinet.dashboard.guests', { count: departure.guests }) }}</p>
              <p class="text-xs text-text-muted">{{ $t('cabinet.dashboard.bookingsCount', { count: departure.bookings }) }}</p>
            </div>
          </div>
        </div>
        <p
          v-else
          class="rounded-card border border-dashed border-border bg-surface px-4 py-6 text-center text-sm text-text-muted"
        >
          {{ recentOrders.length > 0 ? $t('cabinet.dashboard.noDepartures') : $t('booking.noReceived') }}
        </p>
      </section>

      <section class="flex flex-col gap-3">
        <div class="flex items-center justify-between">
          <h2 class="font-semibold text-text">
            {{ $t('cabinet.dashboard.newReviews') }}
            <span v-if="unansweredTotal > 0" class="text-text-muted">({{ unansweredTotal }})</span>
          </h2>
          <NuxtLink :to="localePath('/partner/reviews')" class="text-sm text-primary-600 hover:underline dark:text-primary-400">
            {{ $t('cabinet.dashboard.answer') }}
          </NuxtLink>
        </div>
        <div v-if="newReviews.length > 0" class="flex flex-col gap-2">
          <NuxtLink
            v-for="review in newReviews"
            :key="review.id"
            :to="localePath('/partner/reviews')"
            class="flex flex-col gap-1 rounded-card border border-border bg-surface px-4 py-3 hover:border-brand-500/40"
          >
            <div class="flex items-center justify-between gap-2">
              <span class="truncate text-sm font-medium text-text">{{ review.listing.title }}</span>
              <RatingStars :value="review.rating" />
            </div>
            <p class="line-clamp-2 text-sm text-text-muted">{{ review.text }}</p>
          </NuxtLink>
        </div>
        <p
          v-else
          class="rounded-card border border-dashed border-border bg-surface px-4 py-6 text-center text-sm text-text-muted"
        >
          {{ $t('cabinet.reviews.allAnswered') }}
        </p>
      </section>
    </div>

    <section v-if="topListings.length > 0" class="flex flex-col gap-3">
      <div class="flex items-center justify-between">
        <h2 class="font-semibold text-text">{{ $t('cabinet.dashboard.topListings') }}</h2>
        <NuxtLink :to="localePath('/partner/stats')" class="text-sm text-primary-600 hover:underline dark:text-primary-400">
          {{ $t('cabinet.dashboard.allStats') }}
        </NuxtLink>
      </div>
      <div class="flex flex-col divide-y divide-border rounded-card border border-border bg-surface">
        <div v-for="item in topListings" :key="item.listing.id" class="flex items-center justify-between gap-3 px-4 py-3">
          <NuxtLink :to="localePath(listingPath(item.listing))" class="min-w-0 truncate font-medium text-text hover:underline">
            {{ item.listing.title }}
          </NuxtLink>
          <div class="flex shrink-0 gap-4 text-sm tabular-nums text-text-muted">
            <span class="flex items-center gap-1"><UIcon name="i-heroicons-eye" class="h-4 w-4" />{{ item.views }}</span>
            <span class="flex items-center gap-1"><UIcon name="i-heroicons-chat-bubble-left-ellipsis" class="h-4 w-4" />{{ item.contacts }}</span>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>
