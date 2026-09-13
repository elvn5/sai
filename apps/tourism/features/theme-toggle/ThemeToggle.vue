<script setup lang="ts">
const colorMode = useColorMode();
const isDark = computed(() => colorMode.value === 'dark');

function toggle(): void {
  colorMode.preference = isDark.value ? 'light' : 'dark';
}
</script>

<template>
  <ClientOnly>
    <button
      type="button"
      class="flex h-9 w-9 items-center justify-center rounded-full border border-border bg-surface text-text hover:bg-bg"
      :aria-label="$t('theme.toggle')"
      @click="toggle"
    >
      <span aria-hidden="true">{{ isDark ? '🌙' : '☀️' }}</span>
    </button>
    <template #fallback>
      <span class="block h-9 w-9 rounded-full border border-border bg-surface" aria-hidden="true" />
    </template>
  </ClientOnly>
</template>
