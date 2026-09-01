<script setup lang="ts">
import { useApiClient } from '~/shared/api/client';
import type { Address } from '~/shared/api/types';

definePageMeta({ layout: 'default' });

const api = useApiClient();
const items = ref<Address[]>([]);
const loading = ref(false);

const isModalOpen = ref(false);
const editingId = ref<string | null>(null);
const form = reactive({ title: '', fullAddress: '', comment: '', isActive: true, sortOrder: 0 });

async function load(): Promise<void> {
  loading.value = true;
  try {
    items.value = await api.get<Address[]>('/addresses');
  } finally {
    loading.value = false;
  }
}

function openCreate(): void {
  editingId.value = null;
  form.title = '';
  form.fullAddress = '';
  form.comment = '';
  form.isActive = true;
  form.sortOrder = 0;
  isModalOpen.value = true;
}

function openEdit(item: Address): void {
  editingId.value = item.id;
  form.title = item.title;
  form.fullAddress = item.fullAddress;
  form.comment = item.comment ?? '';
  form.isActive = item.isActive;
  form.sortOrder = item.sortOrder;
  isModalOpen.value = true;
}

async function save(): Promise<void> {
  const payload = {
    title: form.title,
    fullAddress: form.fullAddress,
    comment: form.comment || null,
    isActive: form.isActive,
    sortOrder: form.sortOrder,
  };

  if (editingId.value) {
    await api.put(`/addresses/${editingId.value}`, payload);
  } else {
    await api.post('/addresses', payload);
  }

  isModalOpen.value = false;
  await load();
}

async function remove(item: Address): Promise<void> {
  if (!confirm(`Удалить "${item.title}"?`)) return;
  await api.delete(`/addresses/${item.id}`);
  await load();
}

onMounted(load);
</script>

<template>
  <div>
    <div class="mb-4 flex items-center justify-between">
      <h1 class="text-xl font-semibold">Адреса</h1>
      <UButton @click="openCreate">Добавить</UButton>
    </div>

    <div class="overflow-hidden rounded-lg border bg-white">
      <table class="w-full text-left text-sm">
        <thead class="bg-gray-50 text-gray-600">
          <tr>
            <th class="px-4 py-2">Название</th>
            <th class="px-4 py-2">Адрес</th>
            <th class="px-4 py-2">Активен</th>
            <th class="px-4 py-2" />
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in items" :key="item.id" class="border-t">
            <td class="px-4 py-2">{{ item.title }}</td>
            <td class="px-4 py-2 text-gray-500">{{ item.fullAddress }}</td>
            <td class="px-4 py-2">{{ item.isActive ? 'Да' : 'Нет' }}</td>
            <td class="px-4 py-2 text-right">
              <UButton size="xs" variant="ghost" @click="openEdit(item)">Изменить</UButton>
              <UButton size="xs" variant="ghost" color="red" @click="remove(item)">Удалить</UButton>
            </td>
          </tr>
          <tr v-if="!loading && items.length === 0">
            <td colspan="4" class="px-4 py-6 text-center text-gray-400">Список пуст</td>
          </tr>
        </tbody>
      </table>
    </div>

    <UModal v-model="isModalOpen">
      <div class="p-4">
        <h2 class="mb-4 text-lg font-semibold">{{ editingId ? 'Изменить адрес' : 'Новый адрес' }}</h2>
        <div class="flex flex-col gap-3">
          <UFormGroup label="Название">
            <UInput v-model="form.title" />
          </UFormGroup>
          <UFormGroup label="Полный адрес">
            <UTextarea v-model="form.fullAddress" />
          </UFormGroup>
          <UFormGroup label="Комментарий">
            <UTextarea v-model="form.comment" />
          </UFormGroup>
          <UFormGroup label="Порядок сортировки">
            <UInput v-model.number="form.sortOrder" type="number" />
          </UFormGroup>
          <UCheckbox v-model="form.isActive" label="Активен (виден в боте)" />
        </div>
        <div class="mt-4 flex justify-end gap-2">
          <UButton variant="ghost" @click="isModalOpen = false">Отмена</UButton>
          <UButton @click="save">Сохранить</UButton>
        </div>
      </div>
    </UModal>
  </div>
</template>
