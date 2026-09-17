<script setup lang="ts">
import { useMutation, useQueryClient } from '@tanstack/vue-query';
import { toTypedSchema } from '@vee-validate/zod';
import { useForm } from 'vee-validate';
import { z } from 'zod';
import TiptapEditor from '~/components/features/guide-editor/TiptapEditor.vue';
import IconPicker from '~/components/features/icon-picker/IconPicker.vue';
import { useApiClient } from '~/shared/api/client';
import { GuideDifficulty, GuideSeason } from '~/shared/api/types';
import type { Guide, GuideAttachment } from '~/shared/api/types';
import { EQUIPMENT_CATALOG } from '~/shared/lib/guide-equipment';

const { t, locale } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const auth = useAuth();
const queryClient = useQueryClient();
const { isTma, getInitData } = useTelegramWebApp();

const coverFile = ref<File | null>(null);
const coverPreviewUrl = ref<string | null>(null);
const attachedFiles = ref<File[]>([]);
const submitError = ref<string | null>(null);
const equipment = ref<string[]>([]);

const difficultyOptions = [
  {
    value: GuideDifficulty.EASY,
    icon: 'i-mdi-signal-cellular-1',
    label: t('guides.difficulty.easy'),
  },
  {
    value: GuideDifficulty.MODERATE,
    icon: 'i-mdi-signal-cellular-2',
    label: t('guides.difficulty.moderate'),
  },
  {
    value: GuideDifficulty.HARD,
    icon: 'i-mdi-signal-cellular-3',
    label: t('guides.difficulty.hard'),
  },
];

const seasonOptions = [
  {
    value: GuideSeason.SUMMER,
    icon: 'i-mdi-white-balance-sunny',
    label: t('guides.season.summer'),
  },
  { value: GuideSeason.WINTER, icon: 'i-mdi-snowflake', label: t('guides.season.winter') },
  { value: GuideSeason.ALL_YEAR, icon: 'i-mdi-calendar-sync', label: t('guides.season.all_year') },
];

const equipmentOptions = EQUIPMENT_CATALOG.map((entry) => ({
  value: entry.key,
  icon: entry.icon,
  label: t(`guides.equipmentItems.${entry.key}`),
}));

const formSchema = toTypedSchema(
  z.object({
    title: z.string().min(1, t('login.required')).max(255),
    excerpt: z.string().max(500).optional(),
    content: z.string().min(1, t('login.required')).max(20000),
    difficulty: z.nativeEnum(GuideDifficulty),
    season: z.nativeEnum(GuideSeason),
  }),
);

const { handleSubmit, defineField, errors } = useForm({
  validationSchema: formSchema,
  initialValues: {
    difficulty: GuideDifficulty.EASY,
    season: GuideSeason.ALL_YEAR,
    content: '',
  },
});

const [title, titleAttrs] = defineField('title');
const [excerpt, excerptAttrs] = defineField('excerpt');
const [content] = defineField('content');
const [difficulty] = defineField('difficulty');
const [season] = defineField('season');

useSeoMeta({
  title: () => t('guides.newTitle'),
  robots: 'noindex',
});

onMounted(async () => {
  if (!auth.checked.value) {
    await auth.ensureSession({ isTma: isTma.value, initData: getInitData() });
  }
  if (!auth.user.value) {
    await navigateTo(localePath('/login'));
  }
});

function onCoverChange(event: Event): void {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0] ?? null;
  coverFile.value = file;

  if (coverPreviewUrl.value) URL.revokeObjectURL(coverPreviewUrl.value);
  coverPreviewUrl.value = file ? URL.createObjectURL(file) : null;
}

onBeforeUnmount(() => {
  if (coverPreviewUrl.value) URL.revokeObjectURL(coverPreviewUrl.value);
});

function onFilesSelected(event: Event): void {
  const target = event.target as HTMLInputElement;
  const files = Array.from(target.files ?? []);
  attachedFiles.value = [...attachedFiles.value, ...files];
  target.value = '';
}

function removeFile(index: number): void {
  attachedFiles.value = attachedFiles.value.filter((_file: File, i: number) => i !== index);
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

const createMutation = useMutation({
  mutationFn: async (values: {
    title: string;
    excerpt?: string;
    content: string;
    difficulty: GuideDifficulty;
    season: GuideSeason;
  }) => {
    let coverImageUrl: string | null = null;
    if (coverFile.value) {
      const formData = new FormData();
      formData.append('file', coverFile.value);
      const upload = await api.upload<{ url: string }>('/uploads/image', formData);
      coverImageUrl = upload.url;
    }

    const attachments: GuideAttachment[] = [];
    for (const file of attachedFiles.value) {
      const formData = new FormData();
      formData.append('file', file);
      const upload = await api.upload<GuideAttachment>('/uploads/file', formData);
      attachments.push(upload);
    }

    return api.post<Guide>('/guides', {
      locale: locale.value,
      slug: slugify(values.title),
      title: values.title,
      excerpt: values.excerpt || null,
      content: values.content,
      difficulty: values.difficulty,
      season: values.season,
      equipmentChecklist: equipment.value,
      coverImageUrl,
      attachments,
    });
  },
  onSuccess: (guide) => {
    queryClient.invalidateQueries({ queryKey: ['guides'] });
    navigateTo(localePath(`/guides/${guide.slug}`));
  },
  onError: () => {
    submitError.value = t('guides.createError');
  },
});

const onSubmit = handleSubmit((values) => {
  submitError.value = null;
  createMutation.mutate(values);
});
</script>

<template>
  <section class="mx-auto max-w-5xl px-4 py-12">
    <h1 class="text-3xl font-extrabold text-text">{{ $t('guides.newTitle') }}</h1>

    <form class="mt-8" @submit="onSubmit">
      <div class="grid grid-cols-1 gap-8 lg:grid-cols-[280px_1fr]">
        <aside
          class="order-2 flex h-fit flex-col gap-6 rounded-card border border-border bg-surface p-4 lg:order-1 lg:sticky lg:top-24"
        >
          <UFormGroup :label="$t('guides.difficultyLabel')">
            <IconPicker v-model="difficulty" :options="difficultyOptions" />
          </UFormGroup>

          <UFormGroup :label="$t('guides.seasonLabel')">
            <IconPicker v-model="season" :options="seasonOptions" />
          </UFormGroup>

          <UFormGroup :label="$t('guides.equipmentLabel')" :hint="$t('guides.equipmentHint')">
            <IconPicker v-model="equipment" :options="equipmentOptions" multiple />
          </UFormGroup>
        </aside>

        <div class="order-1 flex flex-col gap-6 lg:order-2">
          <UFormGroup :label="$t('guides.titleLabel')" :error="errors.title">
            <UInput v-model="title" v-bind="titleAttrs" size="lg" />
          </UFormGroup>

          <UFormGroup :label="$t('guides.excerptLabel')" :error="errors.excerpt">
            <UInput v-model="excerpt" v-bind="excerptAttrs" />
          </UFormGroup>

          <UFormGroup :label="$t('guides.coverLabel')">
            <input
              type="file"
              accept="image/*"
              class="block w-full text-sm text-text file:mr-3 file:rounded-md file:border-0 file:bg-brand-600 file:px-3 file:py-2 file:text-sm file:font-medium file:text-white"
              @change="onCoverChange"
            />
            <img
              v-if="coverPreviewUrl"
              :src="coverPreviewUrl"
              alt=""
              class="mt-3 h-48 w-full rounded-lg object-cover"
            />
          </UFormGroup>

          <UFormGroup :label="$t('guides.contentLabel')" :error="errors.content">
            <TiptapEditor v-model="content" :placeholder="$t('guides.contentPlaceholder')" />
          </UFormGroup>

          <UFormGroup :label="$t('guides.filesLabel')" :hint="$t('guides.filesHint')">
            <input
              type="file"
              multiple
              accept=".pdf,.gpx,.kml,.zip,.xml"
              class="block w-full text-sm text-text file:mr-3 file:rounded-md file:border-0 file:bg-brand-600 file:px-3 file:py-2 file:text-sm file:font-medium file:text-white"
              @change="onFilesSelected"
            />
            <ul v-if="attachedFiles.length > 0" class="mt-3 flex flex-col gap-2">
              <li
                v-for="(file, index) in attachedFiles"
                :key="`${file.name}-${index}`"
                class="flex items-center justify-between rounded-card border border-border bg-surface px-3 py-2 text-sm"
              >
                <span class="flex items-center gap-2 text-text">
                  <UIcon name="i-mdi-file-outline" class="h-4 w-4 shrink-0 text-text-muted" />
                  {{ file.name }}
                  <span class="text-text-muted">({{ formatFileSize(file.size) }})</span>
                </span>
                <button
                  type="button"
                  class="text-text-muted transition-colors hover:text-brand-600"
                  :aria-label="$t('guides.removeFile')"
                  @click="removeFile(index)"
                >
                  <UIcon name="i-heroicons-x-mark" class="h-4 w-4" />
                </button>
              </li>
            </ul>
          </UFormGroup>

          <p v-if="submitError" class="text-sm text-brand-600">{{ submitError }}</p>

          <UButton type="submit" color="primary" block :loading="createMutation.isPending.value">
            {{ $t('guides.submit') }}
          </UButton>
        </div>
      </div>
    </form>
  </section>
</template>
