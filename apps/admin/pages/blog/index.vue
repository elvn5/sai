<script setup lang="ts">
import { useApiClient } from '~/shared/api/client';
import { POST_LANGUAGE_OPTIONS, POST_STATUS_OPTIONS, type Post } from '~/shared/api/types';

definePageMeta({ layout: 'default' });

const api = useApiClient();
const posts = ref<Post[]>([]);
const loading = ref(false);

const langLabel = (v: string) => POST_LANGUAGE_OPTIONS.find((o) => o.value === v)?.label ?? v;
const statusLabel = (v: string) => POST_STATUS_OPTIONS.find((o) => o.value === v)?.label ?? v;

function formatDate(v: string | null): string {
  if (!v) return '—';
  return new Date(v).toLocaleDateString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  });
}

async function load(): Promise<void> {
  loading.value = true;
  try {
    posts.value = await api.get<Post[]>('/posts/admin');
  } finally {
    loading.value = false;
  }
}

async function remove(post: Post): Promise<void> {
  if (!confirm(`Удалить пост "${post.title}"?`)) return;
  await api.delete(`/posts/${post.id}`);
  await load();
}

onMounted(load);
</script>

<template>
  <div>
    <div class="mb-4 flex items-center justify-between">
      <h1 class="text-xl font-semibold">Блог</h1>
      <UButton icon="i-lucide-plus" to="/blog/new">Новый пост</UButton>
    </div>

    <div class="overflow-hidden rounded-lg border border-steel-200 bg-white dark:border-steel-800 dark:bg-steel-950">
      <table class="w-full text-left text-sm">
        <thead class="bg-steel-50 text-steel-600 dark:bg-steel-900 dark:text-steel-300">
          <tr>
            <th class="px-4 py-2">Заголовок</th>
            <th class="px-4 py-2">Язык</th>
            <th class="px-4 py-2">Статус</th>
            <th class="px-4 py-2">Опубликован</th>
            <th class="px-4 py-2" />
          </tr>
        </thead>
        <tbody>
          <tr v-for="post in posts" :key="post.id" class="border-t border-steel-200 dark:border-steel-800">
            <td class="px-4 py-2">
              <div class="font-medium">{{ post.title }}</div>
              <div class="text-xs text-steel-400">/{{ post.slug }}</div>
            </td>
            <td class="px-4 py-2">{{ langLabel(post.language) }}</td>
            <td class="px-4 py-2">
              <UBadge :color="post.status === 'published' ? 'primary' : 'neutral'" variant="subtle">
                {{ statusLabel(post.status) }}
              </UBadge>
            </td>
            <td class="px-4 py-2 text-steel-500 dark:text-steel-400">{{ formatDate(post.publishedAt) }}</td>
            <td class="px-4 py-2 text-right whitespace-nowrap">
              <UButton size="xs" variant="ghost" :to="`/blog/${post.id}`">Изменить</UButton>
              <UButton size="xs" variant="ghost" color="error" @click="remove(post)">Удалить</UButton>
            </td>
          </tr>
          <tr v-if="!loading && posts.length === 0">
            <td colspan="5" class="px-4 py-6 text-center text-steel-400 dark:text-steel-500">Постов пока нет</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
