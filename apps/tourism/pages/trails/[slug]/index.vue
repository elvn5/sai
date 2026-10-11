<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import ConditionBadge from '~/components/entities/condition/ConditionBadge.vue';
import ConditionCard from '~/components/entities/condition/ConditionCard.vue';
import OutingCard from '~/components/entities/outing/OutingCard.vue';
import ConditionReportForm from '~/components/features/condition-report-form/ConditionReportForm.vue';
import MountainWeather from '~/components/widgets/mountain-weather/MountainWeather.vue';
import ElevationProfile from '~/components/widgets/trail-map/ElevationProfile.vue';
import TrailMap from '~/components/widgets/trail-map/TrailMap.vue';
import { useApiClient } from '~/shared/api/client';
import type { ConditionReport, Outing, Trail } from '~/shared/api/types';
import { resolveMediaUrl } from '~/shared/lib/resolve-media-url';
import {
  GEAR,
  KINDS,
  SEASONS,
  formatGrade,
  gradeClass,
  optionIcon,
} from '~/shared/lib/trail-options';

const route = useRoute();
const { t } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const auth = useAuth();
const queryClient = useQueryClient();
const config = useRuntimeConfig();
const slug = computed(() => String(route.params.slug));

const trailQuery = useQuery({
  queryKey: ['trail', slug],
  queryFn: async () => {
    try {
      return await api.get<Trail>(`/trails/${slug.value}`);
    } catch {
      return null;
    }
  },
});
const conditionsQuery = useQuery({
  queryKey: ['trail-conditions', slug],
  queryFn: () =>
    api.get<ConditionReport[]>(`/trails/${slug.value}/conditions?limit=20`).catch(() => []),
});
const outingsQuery = useQuery({
  queryKey: ['outings', 'trail', slug],
  queryFn: () => api.get<Outing[]>(`/outings?trail=${slug.value}`).catch(() => []),
});
await Promise.all([trailQuery.suspense(), conditionsQuery.suspense(), outingsQuery.suspense()]);

if (!trailQuery.data.value) {
  throw createError({ statusCode: 404, statusMessage: 'Route not found' });
}

const trail = computed(() => trailQuery.data.value ?? null);
const conditions = computed(() => conditionsQuery.data.value ?? []);
const outings = computed(() => outingsQuery.data.value ?? []);
const canEdit = computed(() =>
  Boolean(trail.value && auth.user.value && trail.value.userId === auth.user.value.id),
);
const cover = computed(() =>
  trail.value ? (trail.value.coverImageUrl ?? trail.value.photos[0] ?? null) : null,
);
const highlight = ref<number | null>(null);
const reportOpen = ref(false);

const stats = computed(() => {
  const tr = trail.value;
  if (!tr) return [];
  const gain = tr.elevationGain ?? tr.trackStats?.elevationGain ?? null;
  const distance = tr.distanceKm ?? tr.trackStats?.distanceKm ?? null;
  return [
    tr.summitElevation && {
      icon: 'i-mdi-summit',
      value: `${tr.summitElevation} ${t('trails.units.m')}`,
      label: t('trails.detail.summit'),
    },
    gain && {
      icon: 'i-mdi-trending-up',
      value: `+${gain} ${t('trails.units.m')}`,
      label: t('trails.detail.gain'),
    },
    distance && {
      icon: 'i-mdi-map-marker-distance',
      value: `${distance} ${t('trails.units.km')}`,
      label: t('trails.detail.distance'),
    },
    tr.durationHours && {
      icon: 'i-mdi-clock-outline',
      value: t('trails.units.hours', { count: tr.durationHours }),
      label: t('trails.detail.duration'),
    },
    {
      icon: 'i-mdi-calendar-range',
      value: t('trails.units.days', { count: tr.days }),
      label: t('trails.detail.days'),
    },
  ].filter((s): s is { icon: string; value: string; label: string } => Boolean(s));
});

// Forecast at the start and at the top.
const weatherPoints = computed(() => {
  const tr = trail.value;
  if (!tr || tr.startLatitude === null || tr.startLongitude === null) return [];
  const start = tr.startElevation ?? tr.trackStats?.minElevation ?? null;
  const top = tr.summitElevation ?? tr.trackStats?.maxElevation ?? null;
  return [
    start && { label: t('trails.weather.start'), meters: start },
    top && top !== start && { label: t('trails.weather.top'), meters: top },
  ].filter((e): e is { label: string; meters: number } => Boolean(e));
});

const deleteReport = useMutation({
  mutationFn: (id: string) => api.delete(`/conditions/${id}`),
  onSuccess: () => queryClient.invalidateQueries({ queryKey: ['trail-conditions', slug] }),
});

function openReport(): void {
  if (!auth.user.value) {
    void navigateTo(localePath('/login'));
    return;
  }
  reportOpen.value = true;
}

useSeoMeta({
  title: () => trail.value?.name,
  description: () => trail.value?.summary ?? t('trails.subtitle'),
  ogTitle: () => trail.value?.name,
  ogImage: () => (cover.value ? resolveMediaUrl(config.public.apiBaseUrl, cover.value) : undefined),
});
</script>

<template>
  <div v-if="trail">
    <section class="relative isolate overflow-hidden px-4 pb-10 pt-16 text-white">
      <img
        v-if="cover"
        :src="resolveMediaUrl(config.public.apiBaseUrl, cover)"
        :alt="trail.name"
        class="absolute inset-0 -z-20 h-full w-full object-cover"
      />
      <div v-if="cover" class="absolute inset-0 -z-10 bg-gradient-to-b from-black/50 via-black/40 to-black/80" />
      <div v-else class="absolute inset-0 -z-10 bg-gradient-to-br from-brand-800 via-brand-900 to-secondary-900" />
      <div class="mx-auto max-w-5xl">
        <NuxtLink :to="localePath({ path: '/trails', query: { area: trail.area } })" class="text-sm text-white/75 hover:text-white">
          ← {{ $t(`trails.area.${trail.area}`) }}
        </NuxtLink>
        <div class="mt-3 flex flex-wrap items-center gap-2">
          <span class="rounded-full px-2.5 py-1 text-xs font-bold backdrop-blur" :class="gradeClass(trail.grade)">
            {{ trail.grade === 'trail' ? $t('trails.grade.trail') : formatGrade(trail.grade) }}
          </span>
          <span class="inline-flex items-center gap-1 rounded-full bg-white/15 px-2.5 py-1 text-xs backdrop-blur">
            <UIcon :name="optionIcon(KINDS, trail.kind)" class="h-4 w-4" />{{ $t(`trails.kind.${trail.kind}`) }}
          </span>
          <span v-if="!trail.isPublished" class="rounded-full bg-amber-500 px-2.5 py-1 text-xs font-semibold">{{ $t('trails.draft') }}</span>
        </div>
        <h1 class="mt-2 text-3xl font-black sm:text-4xl">{{ trail.name }}</h1>
        <p v-if="trail.summary" class="mt-2 max-w-2xl text-white/85">{{ trail.summary }}</p>
        <div class="mt-6 grid grid-cols-2 gap-3 sm:grid-cols-5">
          <div v-for="stat in stats" :key="stat.label" class="rounded-xl bg-white/10 px-3 py-2 backdrop-blur">
            <UIcon :name="stat.icon" class="h-5 w-5 text-white/80" />
            <p class="text-lg font-bold">{{ stat.value }}</p>
            <p class="text-xs text-white/70">{{ stat.label }}</p>
          </div>
        </div>
        <div class="mt-6 flex flex-wrap gap-2">
          <UButton color="primary" size="lg" icon="i-mdi-account-group" :to="localePath({ path: '/outings/new', query: { trail: trail.slug } })">
            {{ $t('trails.detail.organize') }}
          </UButton>
          <UButton size="lg" icon="i-mdi-clipboard-text-outline" class="!bg-white/15 !text-white hover:!bg-white/25" @click="openReport">
            {{ $t('trails.conditions.add') }}
          </UButton>
          <UButton v-if="canEdit" size="lg" icon="i-heroicons-pencil-square" class="!bg-white/15 !text-white hover:!bg-white/25" :to="localePath(`/trails/${trail.slug}/edit`)">
            {{ $t('listing.edit') }}
          </UButton>
        </div>
      </div>
    </section>

    <div class="mx-auto grid max-w-5xl gap-8 px-4 py-10 lg:grid-cols-[1fr_320px]">
      <div class="flex min-w-0 flex-col gap-8">
        <section>
          <TrailMap :track="trail.track ?? []" :start-latitude="trail.startLatitude" :start-longitude="trail.startLongitude" :highlight="highlight" />
          <div class="mt-3">
            <ElevationProfile :track="trail.track ?? []" @hover="highlight = $event" />
          </div>
          <a
            v-if="trail.gpxUrl"
            :href="resolveMediaUrl(config.public.apiBaseUrl, trail.gpxUrl)"
            download
            class="mt-3 inline-flex items-center gap-1 text-sm font-medium text-brand-600 hover:underline dark:text-brand-400"
          >
            <UIcon name="i-mdi-download" class="h-4 w-4" />{{ $t('trails.detail.downloadGpx') }}
          </a>
        </section>

        <section v-if="trail.segments.length > 0">
          <h2 class="mb-4 text-xl font-bold text-text">{{ $t('trails.detail.segments') }}</h2>
          <ol class="relative ml-3 border-l-2 border-dashed border-border">
            <li v-for="(segment, i) in trail.segments" :key="i" class="relative pb-5 pl-6 last:pb-0">
              <span class="absolute -left-[9px] top-1 h-4 w-4 rounded-full border-2 border-surface bg-brand-500" />
              <p class="font-medium text-text">{{ segment.from }} → {{ segment.to }}</p>
              <p class="text-sm text-text-muted">{{ $t('trails.units.hours', { count: segment.hours }) }}</p>
            </li>
          </ol>
        </section>

        <section v-if="trail.description">
          <h2 class="mb-3 text-xl font-bold text-text">{{ $t('trails.detail.description') }}</h2>
          <div class="prose prose-neutral max-w-none dark:prose-invert" v-html="trail.description" />
        </section>

        <section>
          <div class="mb-4 flex items-center justify-between gap-3">
            <h2 class="text-xl font-bold text-text">{{ $t('trails.conditions.title') }}</h2>
            <UButton size="sm" color="primary" variant="soft" icon="i-heroicons-plus" @click="openReport">{{ $t('trails.conditions.add') }}</UButton>
          </div>
          <div v-if="conditions.length > 0" class="flex flex-col gap-3">
            <ConditionCard
              v-for="report in conditions"
              :key="report.id"
              :report="report"
              :can-delete="report.author.id === auth.user.value?.id"
              @delete="deleteReport.mutate(report.id)"
            />
          </div>
          <div v-else class="rounded-card border border-dashed border-border bg-surface px-6 py-8 text-center text-sm text-text-muted">
            {{ $t('trails.conditions.empty') }}
          </div>
        </section>
      </div>

      <aside class="flex flex-col gap-6">
        <div class="rounded-card border border-border bg-surface p-4">
          <p class="mb-2 text-sm font-semibold text-text">{{ $t('trails.conditions.latest') }}</p>
          <ConditionBadge :summary="trail.latestCondition" />
        </div>

        <MountainWeather
          v-if="weatherPoints.length > 0 && trail.startLatitude !== null && trail.startLongitude !== null"
          :latitude="trail.startLatitude"
          :longitude="trail.startLongitude"
          :elevations="weatherPoints"
        />

        <div>
          <div class="mb-3 flex items-center justify-between">
            <h2 class="font-semibold text-text">{{ $t('trails.detail.outings') }}</h2>
            <NuxtLink :to="localePath({ path: '/outings/new', query: { trail: trail.slug } })" class="text-sm text-brand-600 hover:underline dark:text-brand-400">
              + {{ $t('outings.create') }}
            </NuxtLink>
          </div>
          <div v-if="outings.length > 0" class="flex flex-col gap-2">
            <OutingCard v-for="outing in outings" :key="outing.id" :outing="outing" compact />
          </div>
          <p v-else class="text-sm text-text-muted">{{ $t('trails.detail.noOutings') }}</p>
        </div>

        <div v-if="trail.gear.length > 0" class="rounded-card border border-border bg-surface p-4">
          <h2 class="mb-3 font-semibold text-text">{{ $t('trails.detail.gear') }}</h2>
          <ul class="grid grid-cols-2 gap-2 text-sm text-text">
            <li v-for="item in trail.gear" :key="item" class="flex items-center gap-2">
              <UIcon :name="optionIcon(GEAR, item)" class="h-5 w-5 shrink-0 text-brand-600 dark:text-brand-400" />{{ $t(`trails.gear.${item}`) }}
            </li>
          </ul>
        </div>

        <div v-if="trail.seasons.length > 0" class="rounded-card border border-border bg-surface p-4">
          <h2 class="mb-3 font-semibold text-text">{{ $t('trails.detail.seasons') }}</h2>
          <div class="flex flex-wrap gap-2">
            <span v-for="season in trail.seasons" :key="season" class="inline-flex items-center gap-1 rounded-full bg-bg px-2.5 py-1 text-sm text-text">
              <UIcon :name="optionIcon(SEASONS, season)" class="h-4 w-4" />{{ $t(`trails.season.${season}`) }}
            </span>
          </div>
        </div>

        <div v-if="trail.gettingThere" class="rounded-card border border-border bg-surface p-4">
          <h2 class="mb-2 flex items-center gap-2 font-semibold text-text">
            <UIcon name="i-mdi-bus" class="h-5 w-5 text-brand-600 dark:text-brand-400" />{{ $t('trails.detail.gettingThere') }}
          </h2>
          <p class="whitespace-pre-line text-sm text-text">{{ trail.gettingThere }}</p>
        </div>
      </aside>
    </div>

    <ConditionReportForm v-model:open="reportOpen" :trail-id="trail.id" :trail-slug="trail.slug" />
  </div>
</template>
