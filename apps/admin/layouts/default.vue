<script setup lang="ts">
import { useAuthStore } from '~/stores/auth';

const auth = useAuthStore();
const route = useRoute();

// Версия задеплоенного образа (пусто в локальной разработке — тогда не показываем).
const config = useRuntimeConfig();
const version = String(config.public.appVersion ?? '');
const versionLabel = /^\d/.test(version) ? `v${version}` : version;

const links = [
  { label: 'Запрещенные товары', to: '/banned-items' },
  { label: 'Адреса', to: '/addresses' },
  { label: 'Поддержка', to: '/support' },
  { label: 'Посылки', to: '/parcels' },
  { label: 'Пользователи', to: '/users' },
  { label: 'Блог', to: '/blog' },
];

async function handleLogout(): Promise<void> {
  await auth.logout();
  await navigateTo('/login');
}
</script>

<template>
  <div v-if="route.path === '/login'">
    <slot />
  </div>
  <div v-else class="flex min-h-screen bg-steel-50 text-steel-900 dark:bg-steel-900 dark:text-steel-100">
    <aside
      class="flex w-64 shrink-0 flex-col border-r border-steel-200 bg-white p-4 dark:border-steel-800 dark:bg-steel-950"
    >
      <div class="mb-6 flex items-center justify-between gap-2">
        <h1 class="flex items-center gap-2 text-lg font-semibold">
          <span class="inline-block h-5 w-1.5 rounded-full bg-brand-500" aria-hidden="true" />
          <span>As<span class="text-brand-600 dark:text-brand-400">Di</span> Cargo</span>
        </h1>
        <ThemeToggle />
      </div>

      <nav class="flex flex-col gap-1">
        <NuxtLink
          v-for="link in links"
          :key="link.to"
          :to="link.to"
          class="rounded-md px-3 py-2 text-sm text-steel-600 transition-colors hover:bg-steel-100 hover:text-steel-900 dark:text-steel-300 dark:hover:bg-steel-800 dark:hover:text-white"
          active-class="!bg-brand-50 !text-brand-700 font-medium dark:!bg-brand-950 dark:!text-brand-300"
        >
          {{ link.label }}
        </NuxtLink>
      </nav>

      <UButton class="mt-6 w-full" color="neutral" variant="soft" icon="i-lucide-log-out" @click="handleLogout">
        Выйти
      </UButton>

      <p v-if="versionLabel" class="mt-auto pt-6 text-center text-xs text-steel-400 dark:text-steel-600">
        {{ versionLabel }}
      </p>
    </aside>

    <main class="flex-1 p-6">
      <slot />
    </main>
  </div>
</template>
