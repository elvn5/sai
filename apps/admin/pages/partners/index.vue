<script setup lang="ts">
import { useApiClient } from '~/shared/api/client';
import type { AppUser } from '~/shared/api/types';

definePageMeta({ layout: 'default' });

const api = useApiClient();
const requests = ref<AppUser[]>([]);
const loading = ref(false);
const actioningId = ref<string | null>(null);

async function load(): Promise<void> {
  loading.value = true;
  try {
    requests.value = await api.get<AppUser[]>('/partners/requests');
  } finally {
    loading.value = false;
  }
}

async function approve(user: AppUser): Promise<void> {
  actioningId.value = user.id;
  try {
    await api.post(`/partners/${user.id}/approve`);
    await load();
  } finally {
    actioningId.value = null;
  }
}

async function reject(user: AppUser): Promise<void> {
  actioningId.value = user.id;
  try {
    await api.post(`/partners/${user.id}/reject`);
    await load();
  } finally {
    actioningId.value = null;
  }
}

onMounted(load);
</script>

<template>
  <div>
    <h1 class="mb-4 text-xl font-semibold">Заявки на партнёрство</h1>

    <div class="overflow-hidden rounded-lg border bg-white">
      <table class="w-full text-left text-sm">
        <thead class="bg-gray-50 text-gray-600">
          <tr>
            <th class="px-4 py-2">Имя</th>
            <th class="px-4 py-2">Контакты</th>
            <th class="px-4 py-2">Подана</th>
            <th class="px-4 py-2" />
          </tr>
        </thead>
        <tbody>
          <tr v-for="user in requests" :key="user.id" class="border-t">
            <td class="px-4 py-2">{{ [user.firstName, user.lastName].filter(Boolean).join(' ') || '—' }}</td>
            <td class="px-4 py-2 text-gray-500">
              {{ user.email ?? user.phone ?? (user.telegramId ? `tg:${user.telegramId}` : '—') }}
            </td>
            <td class="px-4 py-2 text-gray-500">
              {{ user.partnerRequestedAt ? new Date(user.partnerRequestedAt).toLocaleDateString('ru-RU') : '—' }}
            </td>
            <td class="px-4 py-2 text-right">
              <UButton
                size="xs"
                color="primary"
                :loading="actioningId === user.id"
                @click="approve(user)"
              >
                Одобрить
              </UButton>
              <UButton
                size="xs"
                variant="ghost"
                color="red"
                :loading="actioningId === user.id"
                @click="reject(user)"
              >
                Отклонить
              </UButton>
            </td>
          </tr>
          <tr v-if="!loading && requests.length === 0">
            <td colspan="4" class="px-4 py-6 text-center text-gray-400">Заявок нет</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
