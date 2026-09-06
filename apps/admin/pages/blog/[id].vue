<script setup lang="ts">
import { ApiError, useApiClient } from '~/shared/api/client';
import { POST_LANGUAGE_OPTIONS, POST_STATUS_OPTIONS, type Post } from '~/shared/api/types';

definePageMeta({ layout: 'default' });

const route = useRoute();
const api = useApiClient();
const id = computed(() => String(route.params.id));
const isNew = computed(() => id.value === 'new');

const loading = ref(false);
const saving = ref(false);
const error = ref('');

const form = reactive({
  title: '',
  slug: '',
  excerpt: '',
  coverImageUrl: '',
  language: 'ru' as Post['language'],
  status: 'draft' as Post['status'],
  content: '<p></p>',
});

async function load(): Promise<void> {
  if (isNew.value) return;
  loading.value = true;
  try {
    const post = await api.get<Post>(`/posts/admin/${id.value}`);
    form.title = post.title;
    form.slug = post.slug;
    form.excerpt = post.excerpt ?? '';
    form.coverImageUrl = post.coverImageUrl ?? '';
    form.language = post.language;
    form.status = post.status;
    form.content = post.content || '<p></p>';
  } finally {
    loading.value = false;
  }
}

async function save(): Promise<void> {
  error.value = '';
  saving.value = true;
  try {
    const payload = {
      title: form.title,
      slug: form.slug || undefined,
      excerpt: form.excerpt || null,
      coverImageUrl: form.coverImageUrl || null,
      language: form.language,
      status: form.status,
      content: form.content,
    };
    if (isNew.value) {
      await api.post('/posts', payload);
    } else {
      await api.put(`/posts/${id.value}`, payload);
    }
    await navigateTo('/blog');
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : 'Не удалось сохранить';
  } finally {
    saving.value = false;
  }
}

onMounted(load);
</script>

<template>
  <div class="mx-auto max-w-3xl">
    <div class="mb-4 flex items-center justify-between">
      <h1 class="text-xl font-semibold">{{ isNew ? 'Новый пост' : 'Редактирование поста' }}</h1>
      <UButton variant="ghost" color="neutral" icon="i-lucide-arrow-left" to="/blog">К списку</UButton>
    </div>

    <UAlert v-if="error" color="error" variant="soft" :title="error" class="mb-4" />

    <div class="flex flex-col gap-4">
      <UFormField label="Заголовок" required>
        <UInput v-model="form.title" class="w-full" placeholder="Заголовок поста" />
      </UFormField>

      <div class="grid gap-4 sm:grid-cols-2">
        <UFormField label="Слаг (URL)" help="Пусто — сгенерируется из заголовка">
          <UInput v-model="form.slug" class="w-full" placeholder="my-post" />
        </UFormField>
        <UFormField label="Обложка (URL)">
          <UInput v-model="form.coverImageUrl" class="w-full" placeholder="https://…" />
        </UFormField>
      </div>

      <div class="grid gap-4 sm:grid-cols-2">
        <UFormField label="Язык">
          <USelect v-model="form.language" :items="POST_LANGUAGE_OPTIONS" value-key="value" class="w-full" />
        </UFormField>
        <UFormField label="Статус">
          <USelect v-model="form.status" :items="POST_STATUS_OPTIONS" value-key="value" class="w-full" />
        </UFormField>
      </div>

      <UFormField label="Краткое описание (excerpt)" help="Для карточки и SEO-описания">
        <UTextarea v-model="form.excerpt" :rows="2" class="w-full" />
      </UFormField>

      <UFormField label="Содержимое">
        <TiptapEditor v-model="form.content" />
      </UFormField>

      <div class="flex justify-end gap-2">
        <UButton variant="ghost" color="neutral" to="/blog">Отмена</UButton>
        <UButton :loading="saving" icon="i-lucide-save" @click="save">Сохранить</UButton>
      </div>
    </div>
  </div>
</template>
