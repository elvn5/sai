<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import OutingCard from '~/components/entities/outing/OutingCard.vue';
import { useApiClient } from '~/shared/api/client';
import type { Outing } from '~/shared/api/types';

const { t, locale } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const auth = useAuth();

const outingsQuery = useQuery({
  queryKey: ['outings', 'upcoming'],
  queryFn: () => api.get<Outing[]>('/outings?limit=100'),
});
await outingsQuery.suspense();

const freeOnly = ref(false);

// Bishkek calendar day of a timestamp, as YYYY-MM-DD.
function bishkekDay(iso: string | number): string {
  return new Date(iso).toLocaleDateString('sv-SE', { timeZone: 'Asia/Bishkek' });
}

// Groups: today, tomorrow, this weekend, later (by date).
const groups = computed(() => {
  const list = (outingsQuery.data.value ?? []).filter((o) => !freeOnly.value || o.taken < o.seats);
  const today = bishkekDay(Date.now());
  const tomorrow = bishkekDay(Date.now() + 86_400_000);
  const map = new Map<string, Outing[]>();
  for (const outing of list) {
    const day = bishkekDay(outing.startsAt);
    const label =
      day === today
        ? t('outings.groups.today')
        : day === tomorrow
          ? t('outings.groups.tomorrow')
          : new Date(`${day}T12:00:00`).toLocaleDateString(locale.value, {
              weekday: 'long',
              day: 'numeric',
              month: 'long',
            });
    map.set(label, [...(map.get(label) ?? []), outing]);
  }
  return [...map.entries()];
});

useSeoMeta({
  title: () => t('outings.title'),
  description: () => t('outings.subtitle'),
  ogTitle: () => t('outings.title'),
  ogDescription: () => t('outings.subtitle'),
});
</script>

<template>
  <section class="mx-auto max-w-3xl px-4 py-12">
    <div class="mb-8 flex flex-wrap items-end justify-between gap-4">
      <div>
        <h1 class="text-3xl font-extrabold text-text">{{ $t('outings.title') }}</h1>
        <p class="mt-1 max-w-xl text-text-muted">{{ $t('outings.subtitle') }}</p>
      </div>
      <UButton color="primary" icon="i-heroicons-plus" :to="localePath(auth.user.value ? '/outings/new' : '/login')">
        {{ $t('outings.create') }}
      </UButton>
    </div>

    <label class="mb-6 inline-flex cursor-pointer items-center gap-2 text-sm text-text">
      <UToggle v-model="freeOnly" />{{ $t('outings.freeSeatsOnly') }}
    </label>

    <div v-if="groups.length > 0" class="flex flex-col gap-8">
      <div v-for="[label, items] in groups" :key="label">
        <h2 class="mb-3 text-sm font-bold uppercase tracking-wide text-text-muted first-letter:uppercase">{{ label }}</h2>
        <div class="flex flex-col gap-3">
          <OutingCard v-for="outing in items" :key="outing.id" :outing="outing" />
        </div>
      </div>
    </div>
    <div v-else class="flex flex-col items-center gap-3 rounded-card border border-dashed border-border bg-surface px-6 py-14 text-center">
      <UIcon name="i-mdi-account-group-outline" class="h-10 w-10 text-text-muted" />
      <p class="font-medium text-text">{{ $t('outings.empty') }}</p>
      <p class="max-w-sm text-sm text-text-muted">{{ $t('outings.emptyHint') }}</p>
    </div>
  </section>
</template>
