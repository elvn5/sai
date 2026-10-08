<script setup lang="ts">
import type { ListingOption } from '~/shared/lib/listing-options';

// Multi-select rendered as pill toggles — used for amenities, languages,
// destinations etc. in both filters and forms.
const props = defineProps<{ options: ListingOption[] }>();
const model = defineModel<string[]>({ required: true });

function toggle(value: string): void {
  model.value = model.value.includes(value)
    ? model.value.filter((item) => item !== value)
    : [...model.value, value];
}

const isSelected = (value: string): boolean => model.value.includes(value);
</script>

<template>
  <div class="flex flex-wrap gap-2">
    <button
      v-for="option in props.options"
      :key="option.value"
      type="button"
      :aria-pressed="isSelected(option.value)"
      class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-sm font-medium transition-colors"
      :class="
        isSelected(option.value)
          ? 'border-brand-500 bg-brand-500/10 text-brand-600 dark:text-brand-400'
          : 'border-border text-text-muted hover:border-brand-500/50 hover:text-text'
      "
      @click="toggle(option.value)"
    >
      <UIcon v-if="option.icon" :name="option.icon" class="h-4 w-4" />
      {{ $t(option.labelKey) }}
    </button>
  </div>
</template>
