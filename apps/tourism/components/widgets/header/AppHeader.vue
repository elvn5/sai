<script setup lang="ts">
import LanguageSwitcher from '~/components/features/language-switch/LanguageSwitcher.vue';
import ThemeToggle from '~/components/features/theme-toggle/ThemeToggle.vue';

const { isTma } = useTelegramWebApp();
</script>

<template>
  <header
    v-if="!isTma"
    class="sticky top-0 z-40 border-b border-border bg-surface/90 backdrop-blur"
  >
    <div class="mx-auto flex h-16 max-w-6xl items-center justify-between px-4">
      <NuxtLink to="/" class="text-lg font-bold text-text">
        {{ $t('brand.name') }}
      </NuxtLink>
      <nav class="hidden items-center gap-6 text-sm font-medium text-text-muted sm:flex">
        <NuxtLink to="/" class="hover:text-text">{{ $t('nav.feed') }}</NuxtLink>
        <NuxtLink to="/gallery" class="hover:text-text">{{ $t('nav.gallery') }}</NuxtLink>
        <NuxtLink to="/guides" class="hover:text-text">{{ $t('nav.guides') }}</NuxtLink>
        <NuxtLink to="/blog" class="hover:text-text">{{ $t('nav.blog') }}</NuxtLink>
      </nav>
      <div class="flex items-center gap-2">
        <LanguageSwitcher />
        <ThemeToggle />
      </div>
    </div>
  </header>
</template>
