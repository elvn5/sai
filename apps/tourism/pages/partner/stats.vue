<script setup lang="ts">
import DailyBarChart from '~/components/shared/ui/DailyBarChart.vue';
import StatTile from '~/components/shared/ui/StatTile.vue';
import type { DailyStats, ListingStats } from '~/shared/api/types';
import { formatAmount } from '~/shared/lib/format-price';
import { listingPath } from '~/shared/lib/listing-path';
import { downloadCsv, percentChange, shareOf, usePartnerStats } from '~/shared/lib/partner-cabinet';
import { useRequirePartner } from '~/shared/lib/require-auth';

definePageMeta({ layout: 'partner' });

const { t, locale } = useI18n();
const localePath = useLocalePath();

useRequirePartner();
useSeoMeta({ title: () => t('cabinet.nav.stats'), robots: 'noindex' });

const PERIODS = [7, 30, 90];
const days = ref(30);
const statsQuery = usePartnerStats(days);
const stats = computed(() => statsQuery.data.value ?? null);

const viewsPoints = computed(
  () => stats.value?.daily.map((d: DailyStats) => ({ day: d.day, value: d.views })) ?? [],
);
const contactsPoints = computed(
  () => stats.value?.daily.map((d: DailyStats) => ({ day: d.day, value: d.contacts })) ?? [],
);

const listings = computed(() =>
  [...(stats.value?.listings ?? [])].sort(
    (a: ListingStats, b: ListingStats) => b.views - a.views || b.contacts - a.contacts,
  ),
);

const channels = computed(() => {
  const totals = stats.value?.totals;
  if (!totals) return [];
  return [
    {
      key: 'whatsapp',
      label: 'WhatsApp',
      icon: 'i-heroicons-chat-bubble-left-right',
      value: totals.whatsapp,
    },
    {
      key: 'telegram',
      label: 'Telegram',
      icon: 'i-heroicons-paper-airplane',
      value: totals.telegram,
    },
    {
      key: 'phone',
      label: t('cabinet.stats.phone'),
      icon: 'i-heroicons-phone',
      value: totals.phone,
    },
    {
      key: 'website',
      label: t('cabinet.stats.website'),
      icon: 'i-heroicons-globe-alt',
      value: totals.website,
    },
  ];
});
const maxChannel = computed(() =>
  Math.max(1, ...channels.value.map((c: { value: number }) => c.value)),
);

function exportCsv(): void {
  if (!stats.value) return;
  downloadCsv(`ermekten-stats-${stats.value.from}-${stats.value.to}.csv`, [
    [
      t('cabinet.stats.listing'),
      t('cabinet.stats.views'),
      'WhatsApp',
      'Telegram',
      t('cabinet.stats.phone'),
      t('cabinet.stats.website'),
      t('cabinet.stats.contacts'),
    ],
    ...listings.value.map((l: ListingStats) => [
      l.listing.title,
      l.views,
      l.whatsapp,
      l.telegram,
      l.phone,
      l.website,
      l.contacts,
    ]),
  ]);
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <div class="flex flex-wrap items-end justify-between gap-3">
      <div>
        <h1 class="text-2xl font-extrabold text-text sm:text-3xl">{{ $t('cabinet.nav.stats') }}</h1>
        <p class="mt-1 text-sm text-text-muted">{{ $t('cabinet.stats.subtitle') }}</p>
      </div>
      <div class="flex items-center gap-2">
        <div class="flex rounded-lg border border-border bg-surface p-0.5">
          <button
            v-for="period in PERIODS"
            :key="period"
            type="button"
            class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
            :class="days === period ? 'bg-brand-600 text-white' : 'text-text-muted hover:text-text'"
            @click="days = period"
          >
            {{ $t('cabinet.stats.days', { count: period }) }}
          </button>
        </div>
        <UButton color="gray" icon="i-heroicons-arrow-down-tray" :disabled="!stats" @click="exportCsv">CSV</UButton>
      </div>
    </div>

    <div class="grid grid-cols-2 gap-3 lg:grid-cols-3">
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
        :hint="$t('cabinet.dashboard.vsPrevious')"
      />
      <StatTile
        class="col-span-2 lg:col-span-1"
        :label="$t('cabinet.stats.conversion')"
        icon="i-heroicons-funnel"
        :value="stats ? shareOf(stats.totals.contacts, stats.totals.views) : '—'"
        :hint="$t('cabinet.stats.conversionExplained')"
      />
    </div>

    <div class="grid grid-cols-1 gap-6 rounded-card border border-border bg-surface p-4 sm:grid-cols-2">
      <DailyBarChart :title="$t('cabinet.stats.viewsPerDay')" :points="viewsPoints" />
      <DailyBarChart :title="$t('cabinet.stats.contactsPerDay')" :points="contactsPoints" />
      <p class="text-xs text-text-muted sm:col-span-2">{{ $t('cabinet.stats.howCounted') }}</p>
    </div>

    <section class="flex flex-col gap-3">
      <h2 class="font-semibold text-text">{{ $t('cabinet.stats.channels') }}</h2>
      <div class="flex flex-col gap-3 rounded-card border border-border bg-surface p-4">
        <div v-for="channel in channels" :key="channel.key" class="flex items-center gap-3 text-sm">
          <span class="flex w-28 shrink-0 items-center gap-2 text-text">
            <UIcon :name="channel.icon" class="h-4 w-4 text-text-muted" />{{ channel.label }}
          </span>
          <div class="h-2 flex-1 overflow-hidden rounded-full bg-border">
            <div class="h-full rounded-full bg-brand-500" :style="{ width: `${(channel.value / maxChannel) * 100}%` }" />
          </div>
          <span class="w-10 shrink-0 text-right tabular-nums text-text">{{ channel.value }}</span>
        </div>
      </div>
    </section>

    <section class="flex flex-col gap-3">
      <h2 class="font-semibold text-text">{{ $t('cabinet.stats.byListing') }}</h2>
      <div v-if="listings.length > 0" class="overflow-x-auto rounded-card border border-border bg-surface">
        <table class="w-full min-w-[520px] text-sm">
          <thead>
            <tr class="border-b border-border text-left text-xs uppercase tracking-wide text-text-muted">
              <th class="px-4 py-2 font-medium">{{ $t('cabinet.stats.listing') }}</th>
              <th class="px-3 py-2 text-right font-medium">{{ $t('cabinet.stats.views') }}</th>
              <th class="px-3 py-2 text-right font-medium">{{ $t('cabinet.stats.contacts') }}</th>
              <th class="px-4 py-2 text-right font-medium">{{ $t('cabinet.stats.conversion') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <tr v-for="item in listings" :key="item.listing.id">
              <td class="max-w-0 px-4 py-3">
                <NuxtLink :to="localePath(listingPath(item.listing))" class="block truncate font-medium text-text hover:underline">
                  {{ item.listing.title }}
                </NuxtLink>
                <span class="text-xs text-text-muted">{{ $t(`cabinet.kinds.${item.listing.targetType}`) }}</span>
              </td>
              <td class="px-3 py-3 text-right tabular-nums text-text">{{ item.views }}</td>
              <td class="px-3 py-3 text-right tabular-nums text-text">{{ item.contacts }}</td>
              <td class="px-4 py-3 text-right tabular-nums text-text-muted">{{ shareOf(item.contacts, item.views) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div
        v-else-if="stats"
        class="flex flex-col items-center gap-3 rounded-card border border-dashed border-border bg-surface px-6 py-10 text-center"
      >
        <p class="text-sm text-text-muted">{{ $t('cabinet.stats.noListings') }}</p>
        <UButton color="primary" :to="localePath('/partner/listings')">{{ $t('cabinet.nav.listings') }}</UButton>
      </div>
    </section>
  </div>
</template>
