<script setup lang="ts">
import { useApiClient } from '~/shared/api/client';
import type { GalleryItem } from '~/shared/api/types';

definePageMeta({ layout: 'default' });

const api = useApiClient();
const config = useRuntimeConfig();
const items = ref<GalleryItem[]>([]);
const loading = ref(false);
const uploading = ref(false);
const fileInput = ref<HTMLInputElement | null>(null);
const savingIds = ref<Set<string>>(new Set());

function imageUrl(item: GalleryItem): string {
  return `${config.public.apiBaseUrl}${item.imageUrl}`;
}

async function load(): Promise<void> {
  loading.value = true;
  try {
    items.value = await api.get<GalleryItem[]>('/gallery/admin/all');
  } finally {
    loading.value = false;
  }
}

async function onFileSelected(event: Event): Promise<void> {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  if (!file) return;

  uploading.value = true;
  try {
    const { url } = await api.upload<{ url: string }>('/uploads/image', file);
    await api.post('/gallery', { imageUrl: url, sortOrder: items.value.length });
    await load();
  } finally {
    uploading.value = false;
    target.value = '';
  }
}

async function saveItem(item: GalleryItem): Promise<void> {
  savingIds.value.add(item.id);
  try {
    await api.put(`/gallery/${item.id}`, {
      caption: item.caption || null,
      altText: item.altText || null,
      sortOrder: item.sortOrder,
      isPublished: item.isPublished,
    });
  } finally {
    savingIds.value.delete(item.id);
  }
}

async function remove(item: GalleryItem): Promise<void> {
  if (!confirm('Удалить это изображение?')) return;
  await api.delete(`/gallery/${item.id}`);
  await load();
}

onMounted(load);
</script>

<template>
  <div>
    <div class="mb-4 flex items-center justify-between">
      <h1 class="text-xl font-semibold">Галерея</h1>
      <div>
        <UButton :loading="uploading" @click="fileInput?.click()">Добавить изображение</UButton>
        <input ref="fileInput" type="file" accept="image/*" class="hidden" @change="onFileSelected" />
      </div>
    </div>

    <div v-if="!loading && items.length === 0" class="rounded-lg border bg-white p-6 text-center text-gray-400">
      Галерея пуста
    </div>

    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <div v-for="item in items" :key="item.id" class="flex flex-col gap-2 rounded-lg border bg-white p-3">
        <img :src="imageUrl(item)" class="h-40 w-full rounded object-cover" alt="" />
        <UInput v-model="item.caption" placeholder="Подпись" size="sm" @change="saveItem(item)" />
        <UInput v-model="item.altText" placeholder="Alt-текст (для SEO)" size="sm" @change="saveItem(item)" />
        <div class="flex items-center justify-between">
          <UInput
            v-model.number="item.sortOrder"
            type="number"
            size="sm"
            class="w-20"
            @change="saveItem(item)"
          />
          <UCheckbox v-model="item.isPublished" label="Показывать" @change="saveItem(item)" />
        </div>
        <div class="flex items-center justify-between">
          <span v-if="savingIds.has(item.id)" class="text-xs text-gray-400">Сохранение...</span>
          <UButton size="xs" color="red" variant="ghost" class="ml-auto" @click="remove(item)">
            Удалить
          </UButton>
        </div>
      </div>
    </div>
  </div>
</template>
