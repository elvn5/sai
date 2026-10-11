<script setup lang="ts">
import PersonAvatar from '~/components/entities/achievement/PersonAvatar.vue';
import type { ConditionReport } from '~/shared/api/types';
import { resolveMediaUrl } from '~/shared/lib/resolve-media-url';
import {
  FRESH_REPORT_DAYS,
  TRACTIONS,
  TRAIL_STATES,
  daysSince,
  optionIcon,
} from '~/shared/lib/trail-options';

const props = defineProps<{ report: ConditionReport; canDelete?: boolean }>();
defineEmits<{ delete: [] }>();
const { locale } = useI18n();
const config = useRuntimeConfig();

const age = computed(() => daysSince(props.report.observedOn));
const observed = computed(() =>
  new Date(`${props.report.observedOn}T12:00:00`).toLocaleDateString(locale.value, {
    day: 'numeric',
    month: 'long',
  }),
);
</script>

<template>
  <article class="rounded-card border border-border bg-surface p-4" :class="{ 'opacity-70': age > FRESH_REPORT_DAYS }">
    <header class="flex items-center gap-3">
      <PersonAvatar :person="report.author" size="h-9 w-9 text-sm" />
      <div class="min-w-0 flex-1">
        <p class="truncate text-sm font-semibold text-text">{{ personName(report.author) }}</p>
        <p class="text-xs text-text-muted">
          {{ $t('trails.conditions.observed', { date: observed }) }}
          <template v-if="report.reached"> · {{ $t('trails.conditions.reachedShort', { place: report.reached }) }}</template>
        </p>
      </div>
      <UButton v-if="canDelete" size="xs" color="gray" variant="ghost" icon="i-heroicons-trash" :aria-label="$t('trails.conditions.delete')" @click="$emit('delete')" />
    </header>
    <div class="mt-3 flex flex-wrap gap-2 text-xs">
      <span class="inline-flex items-center gap-1 rounded-full bg-bg px-2.5 py-1 text-text">
        <UIcon :name="optionIcon(TRAIL_STATES, report.trailState)" class="h-4 w-4" />{{ $t(`trails.state.${report.trailState}`) }}
      </span>
      <span class="inline-flex items-center gap-1 rounded-full bg-bg px-2.5 py-1 text-text">
        <UIcon :name="optionIcon(TRACTIONS, report.traction)" class="h-4 w-4" />{{ $t(`trails.traction.${report.traction}`) }}
      </span>
      <span v-if="report.snowLine" class="inline-flex items-center gap-1 rounded-full bg-bg px-2.5 py-1 text-text">
        <UIcon name="i-mdi-snowflake" class="h-4 w-4" />{{ $t('trails.conditions.snowLineShort', { meters: report.snowLine }) }}
      </span>
      <span
        v-if="report.avalancheRisk"
        class="inline-flex items-center gap-1 rounded-full px-2.5 py-1"
        :class="report.avalancheRisk === 'low' ? 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-300' : 'bg-red-500/15 text-red-700 dark:text-red-300'"
      >
        <UIcon name="i-mdi-alert-outline" class="h-4 w-4" />{{ $t(`trails.avalanche.${report.avalancheRisk}`) }}
      </span>
    </div>
    <p v-if="report.text" class="mt-3 whitespace-pre-line text-sm text-text">{{ report.text }}</p>
    <div v-if="report.photos.length > 0" class="mt-3 flex gap-2 overflow-x-auto">
      <a v-for="photo in report.photos" :key="photo" :href="resolveMediaUrl(config.public.apiBaseUrl, photo)" target="_blank" rel="noopener">
        <img :src="resolveMediaUrl(config.public.apiBaseUrl, photo)" alt="" loading="lazy" class="h-24 w-32 shrink-0 rounded-lg object-cover" />
      </a>
    </div>
  </article>
</template>
