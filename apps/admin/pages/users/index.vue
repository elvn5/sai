<script setup lang="ts">
import { useApiClient } from '~/shared/api/client';
import type { AppUser } from '~/shared/api/types';

definePageMeta({ layout: 'default' });

const api = useApiClient();
const users = ref<AppUser[]>([]);
const loading = ref(false);

async function load(): Promise<void> {
  loading.value = true;
  try {
    users.value = await api.get<AppUser[]>('/users');
  } finally {
    loading.value = false;
  }
}

onMounted(load);
</script>

<template>
  <div>
    <h1 class="mb-4 text-xl font-semibold">Пользователи</h1>

    <div class="overflow-hidden rounded-lg border bg-white">
      <table class="w-full text-left text-sm">
        <thead class="bg-gray-50 text-gray-600">
          <tr>
            <th class="px-4 py-2">Имя</th>
            <th class="px-4 py-2">Username</th>
            <th class="px-4 py-2">Telegram ID</th>
            <th class="px-4 py-2">Телефон</th>
            <th class="px-4 py-2">Регистрация</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="user in users" :key="user.id" class="border-t">
            <td class="px-4 py-2">{{ [user.firstName, user.lastName].filter(Boolean).join(' ') || '—' }}</td>
            <td class="px-4 py-2">{{ user.username ? `@${user.username}` : '—' }}</td>
            <td class="px-4 py-2">{{ user.telegramId }}</td>
            <td class="px-4 py-2">{{ user.phone ?? '—' }}</td>
            <td class="px-4 py-2">{{ new Date(user.createdAt).toLocaleDateString('ru-RU') }}</td>
          </tr>
          <tr v-if="!loading && users.length === 0">
            <td colspan="5" class="px-4 py-6 text-center text-gray-400">Пользователей нет</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
