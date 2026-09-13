<script setup lang="ts">
withDefaults(defineProps<{ height?: string; width?: string; rounded?: string }>(), {
  height: '1rem',
  width: '100%',
  rounded: '0.5rem',
});
</script>

<template>
  <div
    class="animate-pulse bg-border/70"
    :style="{ height, width, borderRadius: rounded }"
    aria-hidden="true"
  />
</template>
