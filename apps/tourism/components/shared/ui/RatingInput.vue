<script setup lang="ts">
const model = defineModel<number | undefined>({ required: true });
const hovered = ref<number | null>(null);
</script>

<template>
  <div class="inline-flex items-center gap-1" role="radiogroup" @mouseleave="hovered = null">
    <button
      v-for="star in 5"
      :key="star"
      type="button"
      role="radio"
      :aria-checked="model === star"
      :aria-label="`${star} / 5`"
      class="rounded transition-transform hover:scale-110"
      @mouseenter="hovered = star"
      @click="model = star"
    >
      <UIcon
        name="i-heroicons-star-solid"
        class="h-8 w-8"
        :class="star <= (hovered ?? model ?? 0) ? 'text-amber-400' : 'text-gray-300 dark:text-gray-600'"
      />
    </button>
  </div>
</template>
