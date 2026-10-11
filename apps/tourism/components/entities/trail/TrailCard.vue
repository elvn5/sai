<script setup lang="ts">
import ConditionBadge from '~/components/entities/condition/ConditionBadge.vue';
import type { Trail } from '~/shared/api/types';
import { resolveMediaUrl } from '~/shared/lib/resolve-media-url';
import { KINDS, formatGrade, gradeClass, optionIcon } from '~/shared/lib/trail-options';

const props = defineProps<{ trail: Trail }>();

const localePath = useLocalePath();
const config = useRuntimeConfig();
const cover = computed(() => props.trail.coverImageUrl ?? props.trail.photos[0] ?? null);
</script>

<template>
  <NuxtLink
    :to="localePath(`/trails/${trail.slug}`)"
    class="group flex flex-col overflow-hidden rounded-card border border-border bg-surface transition hover:-translate-y-0.5 hover:shadow-lg"
  >
    <div class="relative aspect-[16/9] overflow-hidden bg-gradient-to-br from-brand-700 to-secondary-700">
      <img
        v-if="cover"
        :src="resolveMediaUrl(config.public.apiBaseUrl, cover)"
        :alt="trail.name"
        loading="lazy"
        class="h-full w-full object-cover transition duration-500 group-hover:scale-105"
      />
      <UIcon v-else :name="optionIcon(KINDS, trail.kind)" class="absolute inset-0 m-auto h-14 w-14 text-white/40" />
      <span class="absolute left-2 top-2 rounded-full px-2.5 py-1 text-xs font-bold backdrop-blur" :class="gradeClass(trail.grade)">
        {{ trail.grade === 'trail' ? $t('trails.grade.trail') : formatGrade(trail.grade) }}
      </span>
      <span
        v-if="trail.summitElevation"
        class="absolute bottom-2 right-2 rounded-full bg-black/60 px-2.5 py-1 text-xs font-semibold text-white"
      >
        ▲ {{ trail.summitElevation }} {{ $t('trails.units.m') }}
      </span>
    </div>

    <div class="flex flex-1 flex-col gap-2 p-4">
      <p class="flex items-center gap-1 text-sm text-text-muted">
        <UIcon :name="optionIcon(KINDS, trail.kind)" class="h-4 w-4" />
        {{ $t(`trails.kind.${trail.kind}`) }} · {{ $t(`trails.area.${trail.area}`) }}
      </p>
      <h3 class="text-lg font-semibold leading-snug text-text">{{ trail.name }}</h3>
      <p v-if="trail.summary" class="line-clamp-2 text-sm text-text-muted">{{ trail.summary }}</p>

      <div class="flex flex-wrap gap-x-4 gap-y-1 text-xs text-text-muted">
        <span v-if="trail.durationHours" class="inline-flex items-center gap-1">
          <UIcon name="i-mdi-clock-outline" class="h-3.5 w-3.5" />
          {{ $t('trails.units.hours', { count: trail.durationHours }) }}
        </span>
        <span v-if="trail.days > 1" class="inline-flex items-center gap-1">
          <UIcon name="i-mdi-calendar-range" class="h-3.5 w-3.5" />
          {{ $t('trails.units.days', { count: trail.days }) }}
        </span>
        <span v-if="trail.distanceKm" class="inline-flex items-center gap-1">
          <UIcon name="i-mdi-map-marker-distance" class="h-3.5 w-3.5" />
          {{ trail.distanceKm }} {{ $t('trails.units.km') }}
        </span>
      </div>

      <div class="mt-auto flex flex-wrap items-center justify-between gap-2 pt-2">
        <ConditionBadge :summary="trail.latestCondition" />
        <span v-if="trail.upcomingOutings > 0" class="inline-flex items-center gap-1 text-xs font-semibold text-brand-600 dark:text-brand-400">
          <UIcon name="i-mdi-account-group" class="h-4 w-4" />
          {{ $t('trails.outingsCount', { count: trail.upcomingOutings }) }}
        </span>
      </div>
    </div>
  </NuxtLink>
</template>
