<script setup lang="ts">
const { locale, locales, setLocale } = useI18n();
const isOpen = ref(false);

const availableLocales = computed(() =>
  (locales.value as Array<{ code: string; name?: string }>).filter((l) => l.code !== locale.value),
);
</script>

<template>
  <div class="relative">
    <button
      type="button"
      class="flex h-9 items-center gap-1 rounded-full border border-border bg-surface px-3 text-sm text-text hover:bg-bg"
      :aria-label="$t('language.switch')"
      @click="isOpen = !isOpen"
    >
      {{ locale.toUpperCase() }}
    </button>
    <ul
      v-if="isOpen"
      class="absolute right-0 mt-2 min-w-32 rounded-xl border border-border bg-surface py-1 shadow-lg"
      @click="isOpen = false"
    >
      <li v-for="item in availableLocales" :key="item.code">
        <button
          type="button"
          class="block w-full px-3 py-1.5 text-left text-sm text-text hover:bg-bg"
          @click="setLocale(item.code)"
        >
          {{ item.name }}
        </button>
      </li>
    </ul>
  </div>
</template>
