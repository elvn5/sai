<script setup lang="ts">
import { useApiClient } from '~/shared/api/client';
import { type AppUser, PARCEL_STATUS_OPTIONS, type Parcel, ParcelStatus } from '~/shared/api/types';

definePageMeta({ layout: 'default' });

const api = useApiClient();
const items = ref<Parcel[]>([]);
const users = ref<AppUser[]>([]);
const loading = ref(false);

const isModalOpen = ref(false);
const editingId = ref<string | null>(null);
const form = reactive({
  userId: '',
  trackNumber: '',
  description: '',
  status: ParcelStatus.CREATED,
  weightKg: '',
  comment: '',
});

const userOptions = computed(() =>
  users.value.map((user: AppUser) => ({
    label:
      `${user.firstName ?? ''} ${user.username ? `@${user.username}` : user.telegramId}`.trim(),
    value: user.id,
  })),
);

function userLabel(userId: string): string {
  const user = users.value.find((item: AppUser) => item.id === userId);
  if (!user) return userId;
  return user.username ? `@${user.username}` : (user.firstName ?? user.telegramId);
}

function statusLabel(status: ParcelStatus): string {
  return PARCEL_STATUS_OPTIONS.find((option) => option.value === status)?.label ?? status;
}

async function load(): Promise<void> {
  loading.value = true;
  try {
    const [parcels, allUsers] = await Promise.all([
      api.get<Parcel[]>('/parcels'),
      api.get<AppUser[]>('/users'),
    ]);
    items.value = parcels;
    users.value = allUsers;
  } finally {
    loading.value = false;
  }
}

function openCreate(): void {
  editingId.value = null;
  form.userId = users.value[0]?.id ?? '';
  form.trackNumber = '';
  form.description = '';
  form.status = ParcelStatus.CREATED;
  form.weightKg = '';
  form.comment = '';
  isModalOpen.value = true;
}

function openEdit(item: Parcel): void {
  editingId.value = item.id;
  form.userId = item.userId;
  form.trackNumber = item.trackNumber;
  form.description = item.description ?? '';
  form.status = item.status;
  form.weightKg = item.weightKg ?? '';
  form.comment = item.comment ?? '';
  isModalOpen.value = true;
}

async function save(): Promise<void> {
  if (editingId.value) {
    await api.put(`/parcels/${editingId.value}`, {
      trackNumber: form.trackNumber,
      description: form.description || null,
      status: form.status,
      weightKg: form.weightKg || null,
      comment: form.comment || null,
    });
  } else {
    await api.post('/parcels', {
      userId: form.userId,
      trackNumber: form.trackNumber,
      description: form.description || null,
      status: form.status,
      weightKg: form.weightKg || null,
      comment: form.comment || null,
    });
  }

  isModalOpen.value = false;
  await load();
}

async function remove(item: Parcel): Promise<void> {
  if (!confirm(`Удалить посылку "${item.trackNumber}"?`)) return;
  await api.delete(`/parcels/${item.id}`);
  await load();
}

onMounted(load);
</script>

<template>
  <div>
    <div class="mb-4 flex items-center justify-between">
      <h1 class="text-xl font-semibold">Посылки</h1>
      <UButton :disabled="users.length === 0" icon="i-lucide-plus" @click="openCreate">Добавить</UButton>
    </div>

    <div class="overflow-hidden rounded-lg border border-steel-200 bg-white dark:border-steel-800 dark:bg-steel-950">
      <table class="w-full text-left text-sm">
        <thead class="bg-steel-50 text-steel-600 dark:bg-steel-900 dark:text-steel-300">
          <tr>
            <th class="px-4 py-2">Трек-номер</th>
            <th class="px-4 py-2">Пользователь</th>
            <th class="px-4 py-2">Статус</th>
            <th class="px-4 py-2" />
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in items" :key="item.id" class="border-t border-steel-200 dark:border-steel-800">
            <td class="px-4 py-2">{{ item.trackNumber }}</td>
            <td class="px-4 py-2">{{ userLabel(item.userId) }}</td>
            <td class="px-4 py-2">{{ statusLabel(item.status) }}</td>
            <td class="px-4 py-2 text-right">
              <UButton size="xs" variant="ghost" @click="openEdit(item)">Изменить</UButton>
              <UButton size="xs" variant="ghost" color="error" @click="remove(item)">Удалить</UButton>
            </td>
          </tr>
          <tr v-if="!loading && items.length === 0">
            <td colspan="4" class="px-4 py-6 text-center text-steel-400 dark:text-steel-500">Посылок нет</td>
          </tr>
        </tbody>
      </table>
    </div>

    <UModal v-model:open="isModalOpen" :title="editingId ? 'Изменить посылку' : 'Новая посылка'">
      <template #body>
        <div class="flex flex-col gap-3">
          <UFormField label="Пользователь">
            <USelectMenu
              v-model="form.userId"
              :items="userOptions"
              value-key="value"
              :disabled="Boolean(editingId)"
              class="w-full"
            />
          </UFormField>
          <UFormField label="Трек-номер">
            <UInput v-model="form.trackNumber" class="w-full" />
          </UFormField>
          <UFormField label="Описание">
            <UTextarea v-model="form.description" class="w-full" />
          </UFormField>
          <UFormField label="Статус">
            <USelectMenu v-model="form.status" :items="PARCEL_STATUS_OPTIONS" value-key="value" class="w-full" />
          </UFormField>
          <UFormField label="Вес, кг">
            <UInput v-model="form.weightKg" class="w-full" />
          </UFormField>
          <UFormField label="Комментарий">
            <UTextarea v-model="form.comment" class="w-full" />
          </UFormField>
        </div>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton variant="ghost" color="neutral" @click="isModalOpen = false">Отмена</UButton>
          <UButton @click="save">Сохранить</UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>
