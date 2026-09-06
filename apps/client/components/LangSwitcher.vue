<script setup lang="ts">
/** Переключатель языка (ru / ky / en). Меняет локаль и маршрут через i18n. */
type Loc = { code: 'ru' | 'ky' | 'en'; name: string };

const { locale, locales, setLocale } = useI18n();

const items = computed(() =>
  (locales.value as Loc[]).map((l) => ({
    label: l.name,
    icon: l.code === locale.value ? 'i-lucide-check' : undefined,
    onSelect: () => setLocale(l.code),
  })),
);

const current = computed(() => (locales.value as Loc[]).find((l) => l.code === locale.value));
</script>

<template>
  <UDropdownMenu :items="items">
    <UButton
      color="neutral"
      variant="ghost"
      icon="i-lucide-languages"
      :label="(current?.code ?? 'ru').toUpperCase()"
      :aria-label="current?.name"
    />
  </UDropdownMenu>
</template>
