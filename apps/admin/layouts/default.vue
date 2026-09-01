<script setup lang="ts">
import { useAuthStore } from '~/stores/auth';

const auth = useAuthStore();
const route = useRoute();

const links = [
  { label: 'Запрещенные товары', to: '/banned-items' },
  { label: 'Адреса', to: '/addresses' },
  { label: 'Поддержка', to: '/support' },
  { label: 'Посылки', to: '/parcels' },
  { label: 'Пользователи', to: '/users' },
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
  <div v-else class="flex min-h-screen bg-gray-50">
    <aside class="w-64 shrink-0 border-r bg-white p-4">
      <h1 class="mb-6 text-lg font-semibold">Админка посылок</h1>
      <nav class="flex flex-col gap-1">
        <NuxtLink
          v-for="link in links"
          :key="link.to"
          :to="link.to"
          class="rounded px-3 py-2 text-sm hover:bg-gray-100"
          active-class="bg-gray-100 font-medium"
        >
          {{ link.label }}
        </NuxtLink>
      </nav>
      <UButton class="mt-6 w-full" color="gray" variant="soft" @click="handleLogout">Выйти</UButton>
    </aside>
    <main class="flex-1 p-6">
      <slot />
    </main>
  </div>
</template>
