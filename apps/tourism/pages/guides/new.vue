<script setup lang="ts">
import { useMutation, useQueryClient } from '@tanstack/vue-query';
import { toTypedSchema } from '@vee-validate/zod';
import { useForm } from 'vee-validate';
import { z } from 'zod';
import { useApiClient } from '~/shared/api/client';
import { GuideDifficulty, GuideSeason } from '~/shared/api/types';
import type { Guide } from '~/shared/api/types';

const { t, locale } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const auth = useAuth();
const queryClient = useQueryClient();

const coverFile = ref<File | null>(null);
const submitError = ref<string | null>(null);

const difficultyOptions = [
  { value: GuideDifficulty.EASY, label: t('guides.difficulty.easy') },
  { value: GuideDifficulty.MODERATE, label: t('guides.difficulty.moderate') },
  { value: GuideDifficulty.HARD, label: t('guides.difficulty.hard') },
];

const seasonOptions = [
  { value: GuideSeason.SUMMER, label: t('guides.season.summer') },
  { value: GuideSeason.WINTER, label: t('guides.season.winter') },
  { value: GuideSeason.ALL_YEAR, label: t('guides.season.all_year') },
];

const formSchema = toTypedSchema(
  z.object({
    title: z.string().min(1, t('login.required')).max(255),
    excerpt: z.string().max(500).optional(),
    content: z.string().min(1, t('login.required')).max(20000),
    difficulty: z.nativeEnum(GuideDifficulty),
    season: z.nativeEnum(GuideSeason),
    equipment: z.string().optional(),
  }),
);

const { handleSubmit, defineField, errors } = useForm({
  validationSchema: formSchema,
  initialValues: {
    difficulty: GuideDifficulty.EASY,
    season: GuideSeason.ALL_YEAR,
  },
});

const [title, titleAttrs] = defineField('title');
const [excerpt, excerptAttrs] = defineField('excerpt');
const [content, contentAttrs] = defineField('content');
const [difficulty, difficultyAttrs] = defineField('difficulty');
const [season, seasonAttrs] = defineField('season');
const [equipment, equipmentAttrs] = defineField('equipment');

useSeoMeta({
  title: () => t('guides.newTitle'),
  robots: 'noindex',
});

onMounted(async () => {
  if (!auth.checked.value) {
    await auth.checkSession();
  }
  if (!auth.user.value) {
    await navigateTo(localePath('/login'));
  }
});

function onCoverChange(event: Event): void {
  const target = event.target as HTMLInputElement;
  coverFile.value = target.files?.[0] ?? null;
}

const createMutation = useMutation({
  mutationFn: async (values: {
    title: string;
    excerpt?: string;
    content: string;
    difficulty: GuideDifficulty;
    season: GuideSeason;
    equipment?: string;
  }) => {
    let coverImageUrl: string | null = null;
    if (coverFile.value) {
      const formData = new FormData();
      formData.append('file', coverFile.value);
      const upload = await api.upload<{ url: string }>('/uploads/image', formData);
      coverImageUrl = upload.url;
    }

    return api.post<Guide>('/guides', {
      locale: locale.value,
      slug: slugify(values.title),
      title: values.title,
      excerpt: values.excerpt || null,
      content: values.content,
      difficulty: values.difficulty,
      season: values.season,
      equipmentChecklist: (values.equipment ?? '')
        .split(/\r?\n|,/)
        .map((entry) => entry.trim())
        .filter(Boolean),
      coverImageUrl,
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
  <section class="mx-auto max-w-2xl px-4 py-12">
    <h1 class="text-3xl font-extrabold text-text">{{ $t('guides.newTitle') }}</h1>

    <form class="mt-8 flex flex-col gap-4" @submit="onSubmit">
      <UFormGroup :label="$t('guides.titleLabel')" :error="errors.title">
        <UInput v-model="title" v-bind="titleAttrs" />
      </UFormGroup>

      <UFormGroup :label="$t('guides.excerptLabel')" :error="errors.excerpt">
        <UInput v-model="excerpt" v-bind="excerptAttrs" />
      </UFormGroup>

      <div class="grid grid-cols-2 gap-4">
        <UFormGroup :label="$t('guides.difficultyLabel')">
          <USelect v-model="difficulty" v-bind="difficultyAttrs" :options="difficultyOptions" />
        </UFormGroup>
        <UFormGroup :label="$t('guides.seasonLabel')">
          <USelect v-model="season" v-bind="seasonAttrs" :options="seasonOptions" />
        </UFormGroup>
      </div>

      <UFormGroup :label="$t('guides.equipmentLabel')" :hint="$t('guides.equipmentHint')">
        <UTextarea v-model="equipment" v-bind="equipmentAttrs" :rows="3" />
      </UFormGroup>

      <UFormGroup :label="$t('guides.coverLabel')">
        <input
          type="file"
          accept="image/*"
          class="block w-full text-sm text-text file:mr-3 file:rounded-md file:border-0 file:bg-brand-600 file:px-3 file:py-2 file:text-sm file:font-medium file:text-white"
          @change="onCoverChange"
        />
      </UFormGroup>

      <UFormGroup :label="$t('guides.contentLabel')" :error="errors.content">
        <UTextarea v-model="content" v-bind="contentAttrs" :rows="10" />
      </UFormGroup>

      <p v-if="submitError" class="text-sm text-brand-600">{{ submitError }}</p>

      <UButton type="submit" color="primary" block :loading="createMutation.isPending.value">
        {{ $t('guides.submit') }}
      </UButton>
    </form>
  </section>
</template>
