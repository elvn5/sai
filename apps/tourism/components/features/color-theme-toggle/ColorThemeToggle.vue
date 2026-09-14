<script setup lang="ts">
const { theme, toggleTheme } = useColorTheme();
</script>

<template>
  <ClientOnly>
    <UButton color="gray" variant="ghost" :aria-label="$t('colorTheme.toggle')" @click="toggleTheme">
      <span
        class="block h-4 w-4 rounded-full border border-current transition-colors"
        :style="{ backgroundColor: theme === 'azure' ? '#0ea5e9' : '#e63946' }"
        aria-hidden="true"
      />
    </UButton>
    <template #fallback>
      <span class="block h-9 w-9 rounded-full border border-border bg-surface" aria-hidden="true" />
    </template>
  </ClientOnly>
</template>
