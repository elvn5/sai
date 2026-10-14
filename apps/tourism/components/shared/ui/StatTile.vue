<script setup lang="ts">
// One headline number of the partner cabinet, optionally with its change
// against the previous period (in percent; null hides it).
defineProps<{
  label: string;
  value: string;
  icon: string;
  hint?: string | null;
  delta?: number | null;
}>();
</script>

<template>
  <div class="flex flex-col gap-1 rounded-card border border-border bg-surface px-4 py-3">
    <div class="flex items-center gap-2 text-sm text-text-muted">
      <UIcon :name="icon" class="h-4 w-4 shrink-0" />
      <span class="truncate">{{ label }}</span>
    </div>
    <p class="text-2xl font-extrabold tabular-nums text-text">{{ value }}</p>
    <p class="flex flex-wrap items-center gap-x-2 text-xs text-text-muted">
      <span
        v-if="delta !== null && delta !== undefined"
        class="inline-flex items-center gap-0.5 font-semibold"
        :class="delta >= 0 ? 'text-emerald-700 dark:text-emerald-400' : 'text-red-700 dark:text-red-400'"
      >
        <UIcon :name="delta >= 0 ? 'i-heroicons-arrow-trending-up' : 'i-heroicons-arrow-trending-down'" class="h-3.5 w-3.5" />
        {{ delta >= 0 ? '+' : '' }}{{ delta }}%
      </span>
      <span v-if="hint">{{ hint }}</span>
    </p>
  </div>
</template>
