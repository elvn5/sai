<script setup lang="ts">
import ImageUploadField from '~/components/ImageUploadField.vue';
import { useApiClient } from '~/shared/api/client';
import {
  LOCALE_OPTIONS,
  Locale,
  type PageSeo,
  SEO_PAGE_OPTIONS,
  SeoPage,
} from '~/shared/api/types';

definePageMeta({ layout: 'default' });

const api = useApiClient();
const loading = ref(false);
const saving = ref(false);
const records = ref<PageSeo[]>([]);

const selectedPage = ref<SeoPage>(SeoPage.HOME);
const selectedLocale = ref<Locale>(Locale.RU);

const form = reactive({
  title: '' as string | null,
  description: '' as string | null,
  ogImageUrl: null as string | null,
});

function applySelection(): void {
  const existing = records.value.find(
    (item: PageSeo) => item.page === selectedPage.value && item.locale === selectedLocale.value,
  );
  form.title = existing?.title ?? '';
  form.description = existing?.description ?? '';
  form.ogImageUrl = existing?.ogImageUrl ?? null;
}

async function load(): Promise<void> {
  loading.value = true;
  try {
    records.value = await api.get<PageSeo[]>('/page-seo/admin/all');
    applySelection();
  } finally {
    loading.value = false;
  }
}

async function save(): Promise<void> {
  saving.value = true;
  try {
    await api.put(`/page-seo/${selectedPage.value}/${selectedLocale.value}`, {
      title: form.title || null,
      description: form.description || null,
      ogImageUrl: form.ogImageUrl,
    });
    await load();
  } finally {
    saving.value = false;
  }
}

watch([selectedPage, selectedLocale], applySelection);
onMounted(load);
</script>

<template>
  <div class="mx-auto max-w-2xl">
    <h1 class="mb-4 text-xl font-semibold">SEO</h1>

    <div class="mb-4 grid grid-cols-2 gap-4">
      <UFormGroup label="Раздел сайта">
        <USelect v-model="selectedPage" :options="SEO_PAGE_OPTIONS" />
      </UFormGroup>
      <UFormGroup label="Язык">
        <USelect v-model="selectedLocale" :options="LOCALE_OPTIONS" />
      </UFormGroup>
    </div>

    <div v-if="!loading" class="flex flex-col gap-4 rounded-lg border bg-white p-4">
      <UFormGroup label="SEO заголовок (title)" hint="Показывается во вкладке браузера и в поиске">
        <UInput v-model="form.title" />
      </UFormGroup>
      <UFormGroup label="SEO описание (meta description)" hint="Показывается в результатах поиска">
        <UTextarea v-model="form.description" :rows="3" />
      </UFormGroup>
      <UFormGroup label="Изображение для соцсетей (Open Graph)">
        <ImageUploadField v-model="form.ogImageUrl" />
      </UFormGroup>

      <div class="flex justify-end">
        <UButton :loading="saving" @click="save">Сохранить</UButton>
      </div>
    </div>
  </div>
</template>
