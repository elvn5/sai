<script setup lang="ts">
import type { ConditionSummary } from '~/shared/api/types';
import { FRESH_REPORT_DAYS, TRAIL_STATES, daysSince, optionIcon } from '~/shared/lib/trail-options';

// How fresh the latest report is, and what it said.
const props = defineProps<{ summary: ConditionSummary | null }>();

const age = computed(() => (props.summary ? daysSince(props.summary.observedOn) : null));
const fresh = computed(() => age.value !== null && age.value <= FRESH_REPORT_DAYS);
</script>

<template>
  <span
    v-if="summary"
    class="inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium"
    :class="fresh ? 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-300' : 'bg-gray-500/15 text-text-muted'"
  >
    <span class="relative flex h-2 w-2">
      <span v-if="fresh" class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-60" />
      <span class="relative inline-flex h-2 w-2 rounded-full" :class="fresh ? 'bg-emerald-500' : 'bg-gray-400'" />
    </span>
    <UIcon :name="optionIcon(TRAIL_STATES, summary.trailState)" class="h-3.5 w-3.5" />
    {{ $t(`trails.state.${summary.trailState}`) }} ·
    {{ age === 0 ? $t('trails.conditions.today') : $t('trails.conditions.daysAgo', { count: age }) }}
  </span>
  <span v-else class="inline-flex items-center gap-1 rounded-full bg-gray-500/10 px-2.5 py-1 text-xs text-text-muted">
    <UIcon name="i-mdi-help-circle-outline" class="h-3.5 w-3.5" />
    {{ $t('trails.conditions.none') }}
  </span>
</template>
