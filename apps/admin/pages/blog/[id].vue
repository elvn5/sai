<script setup lang="ts">
import ImageUploadField from '~/components/ImageUploadField.vue';
import RichTextEditor from '~/components/RichTextEditor.vue';
import { useApiClient } from '~/shared/api/client';
import { LOCALE_OPTIONS, Locale } from '~/shared/api/types';

definePageMeta({ layout: 'default' });

const route = useRoute();
const api = useApiClient();

const id = route.params.id as string;
const isNew = id === 'new';
const loading = ref(false);
const saving = ref(false);
const slugTouched = ref(false);

const form = reactive({
  locale: Locale.RU,
  slug: '',
  title: '',
  excerpt: '' as string | null,
  content: '',
  coverImageUrl: null as string | null,
  seoTitle: '' as string | null,
  seoDescription: '' as string | null,
  isPublished: false,
});

function slugify(value: string): string {
  return value
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

watch(
  () => form.title,
  (title: string) => {
    if (!isNew || slugTouched.value) return;
    form.slug = slugify(title);
  },
);

async function load(): Promise<void> {
  if (isNew) return;
  loading.value = true;
  try {
    const post = await api.get<typeof form & { id: string }>(`/blog/admin/${id}`);
    form.locale = post.locale;
    form.slug = post.slug;
    form.title = post.title;
    form.excerpt = post.excerpt;
    form.content = post.content;
    form.coverImageUrl = post.coverImageUrl;
    form.seoTitle = post.seoTitle;
    form.seoDescription = post.seoDescription;
    form.isPublished = post.isPublished;
  } finally {
    loading.value = false;
  }
}

async function save(): Promise<void> {
  saving.value = true;
  try {
    const payload = {
      locale: form.locale,
      slug: form.slug,
      title: form.title,
      excerpt: form.excerpt || null,
      content: form.content,
      coverImageUrl: form.coverImageUrl,
      seoTitle: form.seoTitle || null,
      seoDescription: form.seoDescription || null,
      isPublished: form.isPublished,
    };

    if (isNew) {
      const created = await api.post<{ id: string }>('/blog', payload);
      await navigateTo(`/blog/${created.id}`);
    } else {
      await api.put(`/blog/${id}`, payload);
      await navigateTo('/blog');
    }
  } finally {
    saving.value = false;
  }
}

async function remove(): Promise<void> {
  if (isNew || !confirm('Удалить эту статью?')) return;
  await api.delete(`/blog/${id}`);
  await navigateTo('/blog');
}

onMounted(load);
</script>

<template>
  <div class="mx-auto max-w-3xl">
    <div class="mb-4 flex items-center justify-between">
      <h1 class="text-xl font-semibold">{{ isNew ? 'Новая статья' : 'Редактирование статьи' }}</h1>
      <div class="flex gap-2">
        <UButton variant="ghost" to="/blog">Назад</UButton>
        <UButton v-if="!isNew" color="red" variant="ghost" @click="remove">Удалить</UButton>
        <UButton :loading="saving" @click="save">Сохранить</UButton>
      </div>
    </div>

    <div v-if="!loading" class="flex flex-col gap-4">
      <div class="grid grid-cols-2 gap-4">
        <UFormGroup label="Язык">
          <USelect v-model="form.locale" :options="LOCALE_OPTIONS" />
        </UFormGroup>
        <UFormGroup label="Slug (для URL)">
          <UInput v-model="form.slug" @input="slugTouched = true" />
        </UFormGroup>
      </div>

      <UFormGroup label="Заголовок">
        <UInput v-model="form.title" />
      </UFormGroup>

      <UFormGroup label="Краткое описание (анонс)">
        <UTextarea v-model="form.excerpt" :rows="2" />
      </UFormGroup>

      <UFormGroup label="Обложка">
        <ImageUploadField v-model="form.coverImageUrl" />
      </UFormGroup>

      <UFormGroup label="Содержание">
        <RichTextEditor v-model="form.content" />
      </UFormGroup>

      <UDivider label="SEO" />

      <UFormGroup label="SEO заголовок" hint="Если пусто — используется заголовок статьи">
        <UInput v-model="form.seoTitle" />
      </UFormGroup>
      <UFormGroup label="SEO описание" hint="Если пусто — используется краткое описание">
        <UTextarea v-model="form.seoDescription" :rows="2" />
      </UFormGroup>

      <UCheckbox v-model="form.isPublished" label="Опубликовано (видно на сайте)" />
    </div>
  </div>
</template>
