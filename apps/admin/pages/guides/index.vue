<script setup lang="ts">
import { useApiClient } from '~/shared/api/client';
import { type Guide, LOCALE_OPTIONS, type Locale } from '~/shared/api/types';

definePageMeta({ layout: 'default' });

const api = useApiClient();
const items = ref<Guide[]>([]);
const loading = ref(false);
const localeFilter = ref<Locale | ''>('');

const localeFilterOptions = [{ value: '', label: 'Все языки' }, ...LOCALE_OPTIONS];

function localeLabel(locale: Locale): string {
  return LOCALE_OPTIONS.find((option) => option.value === locale)?.label ?? locale;
}

async function load(): Promise<void> {
  loading.value = true;
  try {
    const query = localeFilter.value ? `?locale=${localeFilter.value}` : '';
    items.value = await api.get<Guide[]>(`/guides/admin/all${query}`);
  } finally {
    loading.value = false;
  }
}

async function remove(item: Guide): Promise<void> {
  if (!confirm(`Удалить гайд "${item.title}"?`)) return;
  await api.delete(`/guides/${item.id}`);
  await load();
}

watch(localeFilter, load);
onMounted(load);
</script>

<template>
  <div>
    <div class="mb-4 flex items-center justify-between">
      <h1 class="text-xl font-semibold">Гайды</h1>
      <div class="flex items-center gap-2">
        <USelect v-model="localeFilter" :options="localeFilterOptions" class="w-40" />
        <UButton to="/guides/new">Новый гайд</UButton>
      </div>
    </div>

    <div class="overflow-hidden rounded-lg border bg-white">
      <table class="w-full text-left text-sm">
        <thead class="bg-gray-50 text-gray-600">
          <tr>
            <th class="px-4 py-2">Заголовок</th>
            <th class="px-4 py-2">Язык</th>
            <th class="px-4 py-2">Slug</th>
            <th class="px-4 py-2">Статус</th>
            <th class="px-4 py-2" />
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in items" :key="item.id" class="border-t">
            <td class="px-4 py-2">
              <NuxtLink :to="`/guides/${item.id}`" class="hover:underline">{{ item.title }}</NuxtLink>
            </td>
            <td class="px-4 py-2 text-gray-500">{{ localeLabel(item.locale) }}</td>
            <td class="px-4 py-2 text-gray-500">{{ item.slug }}</td>
            <td class="px-4 py-2">
              <UBadge :color="item.isPublished ? 'primary' : 'gray'" variant="subtle">
                {{ item.isPublished ? 'Опубликовано' : 'Черновик' }}
              </UBadge>
            </td>
            <td class="px-4 py-2 text-right">
              <UButton size="xs" variant="ghost" :to="`/guides/${item.id}`">Изменить</UButton>
              <UButton size="xs" variant="ghost" color="red" @click="remove(item)">Удалить</UButton>
            </td>
          </tr>
          <tr v-if="!loading && items.length === 0">
            <td colspan="5" class="px-4 py-6 text-center text-gray-400">Гайдов пока нет</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
