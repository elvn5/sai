<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import TrailCard from '~/components/entities/trail/TrailCard.vue';
import { useApiClient } from '~/shared/api/client';
import type { ConditionReport, Trail } from '~/shared/api/types';
import {
  AREAS,
  GRADES,
  KINDS,
  SEASONS,
  TRAIL_STATES,
  formatGrade,
  optionIcon,
} from '~/shared/lib/trail-options';

const { t } = useI18n();
const localePath = useLocalePath();
const route = useRoute();
const api = useApiClient();
const auth = useAuth();

const area = ref(typeof route.query.area === 'string' ? route.query.area : '');
const kind = ref('');
const season = ref('');
const grade = ref('');
const searchInput = ref('');
const search = ref('');
let searchTimer: ReturnType<typeof setTimeout> | undefined;
watch(searchInput, (value: string) => {
  clearTimeout(searchTimer);
  searchTimer = setTimeout(() => {
    search.value = value;
  }, 300);
});

const query = computed(() => {
  const params = new URLSearchParams();
  if (area.value) params.set('area', area.value);
  if (kind.value) params.set('kind', kind.value);
  if (season.value) params.set('season', season.value);
  if (grade.value) params.set('grade', grade.value);
  if (search.value.trim()) params.set('search', search.value.trim());
  return params.toString();
});

const trailsQuery = useQuery({
  queryKey: ['trails', query],
  queryFn: () => api.get<Trail[]>(`/trails${query.value ? `?${query.value}` : ''}`),
});
const areasQuery = useQuery({
  queryKey: ['trail-areas'],
  queryFn: () => api.get<Record<string, number>>('/trails/areas'),
});
const recentQuery = useQuery({
  queryKey: ['conditions', 'recent'],
  queryFn: () => api.get<ConditionReport[]>('/conditions?limit=8'),
});
await Promise.all([trailsQuery.suspense(), areasQuery.suspense(), recentQuery.suspense()]);

const trails = computed(() => trailsQuery.data.value ?? []);
const areaChips = computed(() =>
  AREAS.filter((a) => (areasQuery.data.value?.[a] ?? 0) > 0).map((a) => ({
    value: a,
    count: areasQuery.data.value?.[a] ?? 0,
  })),
);

const kindOptions = computed(() => [
  { value: '', label: t('trails.filters.anyKind') },
  ...KINDS.map((k) => ({ value: k.value, label: t(`trails.kind.${k.value}`) })),
]);
const seasonOptions = computed(() => [
  { value: '', label: t('trails.filters.anySeason') },
  ...SEASONS.map((s) => ({ value: s.value, label: t(`trails.season.${s.value}`) })),
]);
const gradeOptions = computed(() => [
  { value: '', label: t('trails.filters.anyGrade') },
  { value: 'trail', label: t('trails.grade.trail') },
  { value: '1a,1b', label: `${formatGrade('1a')}–${formatGrade('1b')}` },
  { value: '2a,2b', label: `${formatGrade('2a')}–${formatGrade('2b')}` },
  { value: GRADES.slice(5).join(','), label: t('trails.filters.harder') },
]);

useSeoMeta({
  title: () => t('trails.title'),
  description: () => t('trails.subtitle'),
  ogTitle: () => t('trails.title'),
  ogDescription: () => t('trails.subtitle'),
});
</script>

<template>
  <section class="mx-auto max-w-6xl px-4 py-12">
    <div class="mb-6 flex flex-wrap items-end justify-between gap-4">
      <div>
        <h1 class="text-3xl font-extrabold text-text">{{ $t('trails.title') }}</h1>
        <p class="mt-1 max-w-2xl text-text-muted">{{ $t('trails.subtitle') }}</p>
      </div>
      <div class="flex gap-2">
        <UButton color="gray" icon="i-mdi-account-group" :to="localePath('/outings')">{{ $t('nav.outings') }}</UButton>
        <UButton v-if="auth.isPartner.value" color="primary" icon="i-heroicons-plus" :to="localePath('/trails/new')">
          {{ $t('trails.add') }}
        </UButton>
      </div>
    </div>

    <div v-if="(recentQuery.data.value ?? []).length > 0" class="mb-8">
      <h2 class="mb-3 flex items-center gap-2 text-sm font-semibold uppercase tracking-wide text-text-muted">
        <span class="relative flex h-2 w-2">
          <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-60" />
          <span class="relative inline-flex h-2 w-2 rounded-full bg-emerald-500" />
        </span>
        {{ $t('trails.conditions.recent') }}
      </h2>
      <div class="-mx-4 flex snap-x gap-3 overflow-x-auto px-4 pb-2">
        <NuxtLink
          v-for="report in recentQuery.data.value"
          :key="report.id"
          :to="localePath(`/trails/${report.trail.slug}`)"
          class="w-64 shrink-0 snap-start rounded-card border border-border bg-surface p-3 transition hover:border-brand-400"
        >
          <p class="truncate text-sm font-semibold text-text">{{ report.trail.name }}</p>
          <p class="mt-1 flex items-center gap-1 text-xs text-text-muted">
            <UIcon :name="optionIcon(TRAIL_STATES, report.trailState)" class="h-4 w-4" />
            {{ $t(`trails.state.${report.trailState}`) }} · {{ $t(`trails.traction.${report.traction}`) }}
          </p>
          <p v-if="report.text" class="mt-1 line-clamp-2 text-xs text-text">{{ report.text }}</p>
          <p class="mt-1 text-xs text-text-muted">{{ new Date(`${report.observedOn}T12:00:00`).toLocaleDateString() }}</p>
        </NuxtLink>
      </div>
    </div>

    <div class="mb-4 flex flex-wrap gap-2">
      <UButton size="xs" :color="area === '' ? 'primary' : 'gray'" :variant="area === '' ? 'solid' : 'soft'" @click="area = ''">
        {{ $t('trails.filters.allAreas') }}
      </UButton>
      <UButton
        v-for="chip in areaChips"
        :key="chip.value"
        size="xs"
        :color="area === chip.value ? 'primary' : 'gray'"
        :variant="area === chip.value ? 'solid' : 'soft'"
        @click="area = chip.value"
      >
        {{ $t(`trails.area.${chip.value}`) }} · {{ chip.count }}
      </UButton>
    </div>

    <div class="mb-8 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <UInput v-model="searchInput" icon="i-heroicons-magnifying-glass" :placeholder="$t('trails.filters.search')" />
      <USelect v-model="kind" :options="kindOptions" option-attribute="label" />
      <USelect v-model="grade" :options="gradeOptions" option-attribute="label" />
      <USelect v-model="season" :options="seasonOptions" option-attribute="label" />
    </div>

    <div v-if="trails.length > 0" class="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
      <TrailCard v-for="trail in trails" :key="trail.id" :trail="trail" />
    </div>
    <div v-else class="flex flex-col items-center gap-3 rounded-card border border-dashed border-border bg-surface px-6 py-14 text-center">
      <UIcon name="i-mdi-map-search-outline" class="h-10 w-10 text-text-muted" />
      <p class="text-text-muted">{{ $t('trails.empty') }}</p>
    </div>
  </section>
</template>
