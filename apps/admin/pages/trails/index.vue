<script setup lang="ts">
import { useApiClient } from '~/shared/api/client';
import type { ConditionReportSummary, TrailSummary } from '~/shared/api/types';

definePageMeta({ layout: 'default' });

const api = useApiClient();
const trails = ref<TrailSummary[]>([]);
const reports = ref<ConditionReportSummary[]>([]);
const loading = ref(false);
const busyId = ref<string | null>(null);

async function load(): Promise<void> {
  loading.value = true;
  try {
    [trails.value, reports.value] = await Promise.all([
      api.get<TrailSummary[]>('/trails/admin/all'),
      api.get<ConditionReportSummary[]>('/conditions?limit=50'),
    ]);
  } finally {
    loading.value = false;
  }
}

async function togglePublished(trail: TrailSummary): Promise<void> {
  busyId.value = trail.id;
  try {
    await api.put(`/trails/${trail.id}`, { isPublished: !trail.isPublished });
    await load();
  } finally {
    busyId.value = null;
  }
}

async function removeTrail(trail: TrailSummary): Promise<void> {
  if (!window.confirm(`Удалить маршрут «${trail.name}» вместе со сводками?`)) return;
  await api.delete(`/trails/${trail.id}`);
  await load();
}

async function removeReport(report: ConditionReportSummary): Promise<void> {
  if (!window.confirm('Удалить сводку?')) return;
  await api.delete(`/conditions/${report.id}`);
  await load();
}

onMounted(load);
</script>

<template>
  <div class="flex flex-col gap-8">
    <div>
      <h1 class="text-xl font-semibold">Маршруты</h1>
      <p class="mb-4 text-sm text-gray-500">Маршруты публикуют партнёры на сайте. Здесь можно скрыть или удалить любой.</p>
      <div class="overflow-hidden rounded-lg border bg-white">
        <table class="w-full text-left text-sm">
          <thead class="bg-gray-50 text-gray-600">
            <tr>
              <th class="px-4 py-2">Название</th>
              <th class="px-4 py-2">Район</th>
              <th class="px-4 py-2">Категория</th>
              <th class="px-4 py-2">Статус</th>
              <th class="px-4 py-2" />
            </tr>
          </thead>
          <tbody>
            <tr v-for="trail in trails" :key="trail.id" class="border-t">
              <td class="px-4 py-2">
                {{ trail.name }}
                <span class="ml-1 text-xs text-gray-400">/trails/{{ trail.slug }}</span>
                <span v-if="!trail.userId" class="ml-2 rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-600">редакция</span>
              </td>
              <td class="px-4 py-2 text-gray-500">{{ trail.area }}</td>
              <td class="px-4 py-2 text-gray-500">{{ trail.grade }}</td>
              <td class="px-4 py-2 text-gray-500">{{ trail.isPublished ? 'Опубликован' : 'Скрыт' }}</td>
              <td class="whitespace-nowrap px-4 py-2 text-right">
                <UButton size="xs" variant="ghost" :loading="busyId === trail.id" @click="togglePublished(trail)">
                  {{ trail.isPublished ? 'Скрыть' : 'Опубликовать' }}
                </UButton>
                <UButton size="xs" variant="ghost" color="red" icon="i-heroicons-trash" @click="removeTrail(trail)" />
              </td>
            </tr>
            <tr v-if="!loading && trails.length === 0">
              <td colspan="5" class="px-4 py-6 text-center text-gray-400">Маршрутов пока нет</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div>
      <h2 class="mb-3 text-lg font-semibold">Последние сводки</h2>
      <div class="overflow-hidden rounded-lg border bg-white">
        <table class="w-full text-left text-sm">
          <thead class="bg-gray-50 text-gray-600">
            <tr>
              <th class="px-4 py-2">Маршрут</th>
              <th class="px-4 py-2">Автор</th>
              <th class="px-4 py-2">Дата</th>
              <th class="px-4 py-2">Текст</th>
              <th class="px-4 py-2" />
            </tr>
          </thead>
          <tbody>
            <tr v-for="report in reports" :key="report.id" class="border-t">
              <td class="px-4 py-2">{{ report.trail.name }}</td>
              <td class="px-4 py-2 text-gray-500">{{ report.author.firstName ?? report.author.username ?? '—' }}</td>
              <td class="px-4 py-2 text-gray-500">{{ report.observedOn }}</td>
              <td class="max-w-md truncate px-4 py-2 text-gray-500">{{ report.text ?? `${report.trailState} · ${report.traction}` }}</td>
              <td class="px-4 py-2 text-right">
                <UButton size="xs" variant="ghost" color="red" icon="i-heroicons-trash" @click="removeReport(report)" />
              </td>
            </tr>
            <tr v-if="!loading && reports.length === 0">
              <td colspan="5" class="px-4 py-6 text-center text-gray-400">Сводок пока нет</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
