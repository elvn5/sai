<script setup lang="ts">
import ImageUploadField from '~/components/ImageUploadField.vue';
import RichTextEditor from '~/components/RichTextEditor.vue';
import { useApiClient } from '~/shared/api/client';
import {
  GUIDE_DIFFICULTY_OPTIONS,
  GUIDE_SEASON_OPTIONS,
  GuideDifficulty,
  GuideSeason,
  LOCALE_OPTIONS,
  Locale,
} from '~/shared/api/types';

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
  difficulty: GuideDifficulty.EASY,
  season: GuideSeason.ALL_YEAR,
  equipmentChecklistText: '',
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
    const guide = await api.get<typeof form & { id: string; equipmentChecklist: string[] }>(
      `/guides/admin/${id}`,
    );
    form.locale = guide.locale;
    form.slug = guide.slug;
    form.title = guide.title;
    form.excerpt = guide.excerpt;
    form.content = guide.content;
    form.difficulty = guide.difficulty;
    form.season = guide.season;
    form.equipmentChecklistText = guide.equipmentChecklist.join('\n');
    form.coverImageUrl = guide.coverImageUrl;
    form.seoTitle = guide.seoTitle;
    form.seoDescription = guide.seoDescription;
    form.isPublished = guide.isPublished;
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
      difficulty: form.difficulty,
      season: form.season,
      equipmentChecklist: form.equipmentChecklistText
        .split('\n')
        .map((line: string) => line.trim())
        .filter(Boolean),
      coverImageUrl: form.coverImageUrl,
      seoTitle: form.seoTitle || null,
      seoDescription: form.seoDescription || null,
      isPublished: form.isPublished,
    };

    if (isNew) {
      const created = await api.post<{ id: string }>('/guides', payload);
      await navigateTo(`/guides/${created.id}`);
    } else {
      await api.put(`/guides/${id}`, payload);
      await navigateTo('/guides');
    }
  } finally {
    saving.value = false;
  }
}

async function remove(): Promise<void> {
  if (isNew || !confirm('Удалить этот гайд?')) return;
  await api.delete(`/guides/${id}`);
  await navigateTo('/guides');
}

onMounted(load);
</script>

<template>
  <div class="mx-auto max-w-3xl">
    <div class="mb-4 flex items-center justify-between">
      <h1 class="text-xl font-semibold">{{ isNew ? 'Новый гайд' : 'Редактирование гайда' }}</h1>
      <div class="flex gap-2">
        <UButton variant="ghost" to="/guides">Назад</UButton>
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

      <div class="grid grid-cols-2 gap-4">
        <UFormGroup label="Сложность">
          <USelect v-model="form.difficulty" :options="GUIDE_DIFFICULTY_OPTIONS" />
        </UFormGroup>
        <UFormGroup label="Сезон">
          <USelect v-model="form.season" :options="GUIDE_SEASON_OPTIONS" />
        </UFormGroup>
      </div>

      <UFormGroup label="Список снаряжения" hint="По одному пункту на строку">
        <UTextarea v-model="form.equipmentChecklistText" :rows="5" placeholder="Палатка&#10;Спальный мешок&#10;Треккинговые палки" />
      </UFormGroup>

      <UFormGroup label="Обложка">
        <ImageUploadField v-model="form.coverImageUrl" />
      </UFormGroup>

      <UFormGroup label="Содержание">
        <RichTextEditor v-model="form.content" />
      </UFormGroup>

      <UDivider label="SEO" />

      <UFormGroup label="SEO заголовок" hint="Если пусто — используется заголовок гайда">
        <UInput v-model="form.seoTitle" />
      </UFormGroup>
      <UFormGroup label="SEO описание" hint="Если пусто — используется краткое описание">
        <UTextarea v-model="form.seoDescription" :rows="2" />
      </UFormGroup>

      <UCheckbox v-model="form.isPublished" label="Опубликовано (видно на сайте)" />
    </div>
  </div>
</template>
