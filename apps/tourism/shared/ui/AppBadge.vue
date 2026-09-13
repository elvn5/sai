<script setup lang="ts">
withDefaults(
  defineProps<{
    tone?: 'primary' | 'secondary' | 'neutral';
    locked?: boolean;
  }>(),
  { tone: 'neutral', locked: false },
);

const toneClasses: Record<string, string> = {
  primary: 'bg-primary/10 text-primary',
  secondary: 'bg-secondary/15 text-secondary-hover',
  neutral: 'bg-border/60 text-text-muted',
};
</script>

<template>
  <span
    class="inline-flex items-center gap-1 rounded-full px-3 py-1 text-xs font-medium"
    :class="locked ? 'bg-border/40 text-text-muted opacity-60 grayscale' : toneClasses[tone]"
  >
    <slot />
  </span>
</template>
