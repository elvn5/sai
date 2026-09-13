<script setup lang="ts">
withDefaults(defineProps<{ as?: 'div' | 'article' | 'section' }>(), { as: 'div' });
</script>

<template>
  <component
    :is="as"
    class="rounded-card border border-border bg-surface p-4 shadow-sm"
  >
    <slot />
  </component>
</template>
