<script setup lang="ts">
const props = withDefaults(defineProps<{ value: number; size?: 'sm' | 'md' | 'lg' }>(), {
  size: 'sm',
});

const SIZE_CLASSES = { sm: 'h-4 w-4', md: 'h-5 w-5', lg: 'h-7 w-7' } as const;

// Rounded to the nearest whole star — half-stars aren't worth the extra
// icon for a summary this small.
const filled = computed(() => Math.round(props.value));
</script>

<template>
  <span class="inline-flex items-center" :aria-label="`${value.toFixed(1)} / 5`">
    <UIcon
      v-for="star in 5"
      :key="star"
      name="i-heroicons-star-solid"
      :class="[SIZE_CLASSES[size], star <= filled ? 'text-amber-400' : 'text-gray-300 dark:text-gray-600']"
    />
  </span>
</template>
