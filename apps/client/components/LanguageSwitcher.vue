<template>
  <UDropdown :items="items" :popper="{ placement: 'bottom-end' }">
    <UButton
      color="gray"
      variant="ghost"
      icon="i-heroicons-language-20-solid"
      :aria-label="$t('language.switch')"
    />
  </UDropdown>
</template>

<script setup lang="ts">
const { locale, locales, setLocale } = useI18n();

const items = computed(() => [
  locales.value.map((l) => ({
    label: typeof l === 'string' ? l : l.name,
    active: (typeof l === 'string' ? l : l.code) === locale.value,
    click: () => setLocale(typeof l === 'string' ? l : l.code),
  })),
]);
</script>
