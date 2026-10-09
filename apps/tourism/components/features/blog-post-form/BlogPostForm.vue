<script setup lang="ts">
import { useMutation, useQueryClient } from '@tanstack/vue-query';
import { toTypedSchema } from '@vee-validate/zod';
import { useForm } from 'vee-validate';
import { z } from 'zod';
import TiptapEditor from '~/components/features/guide-editor/TiptapEditor.vue';
import { ApiError, useApiClient } from '~/shared/api/client';
import type { BlogPost } from '~/shared/api/types';

// Creates a partner's blog post, or edits one when `post` is set.
const props = defineProps<{ post?: BlogPost }>();

const { t, locale } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const queryClient = useQueryClient();
const config = useRuntimeConfig();

const coverFile = ref<File | null>(null);
const coverPreviewUrl = ref<string | null>(
  props.post?.coverImageUrl ? `${config.public.apiBaseUrl}${props.post.coverImageUrl}` : null,
);
const submitError = ref<string | null>(null);

const formSchema = toTypedSchema(
  z.object({
    title: z.string().min(1, t('login.required')).max(255),
    excerpt: z.string().max(500).optional(),
    content: z.string().min(1, t('login.required')),
    isPublished: z.boolean(),
  }),
);

const { handleSubmit, defineField, errors } = useForm({
  validationSchema: formSchema,
  initialValues: {
    title: props.post?.title ?? '',
    excerpt: props.post?.excerpt ?? '',
    content: props.post?.content ?? '',
    isPublished: props.post?.isPublished ?? true,
  },
});

const [title, titleAttrs] = defineField('title');
const [excerpt, excerptAttrs] = defineField('excerpt');
const [content] = defineField('content');
const [isPublished] = defineField('isPublished');

function onCoverChange(event: Event): void {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0] ?? null;
  coverFile.value = file;
  if (coverPreviewUrl.value?.startsWith('blob:')) URL.revokeObjectURL(coverPreviewUrl.value);
  coverPreviewUrl.value = file ? URL.createObjectURL(file) : null;
}

onBeforeUnmount(() => {
  if (coverPreviewUrl.value?.startsWith('blob:')) URL.revokeObjectURL(coverPreviewUrl.value);
});

interface FormValues {
  title: string;
  excerpt?: string;
  content: string;
  isPublished: boolean;
}

// A new post's slug comes from its title; on a clash, a short suffix
// keeps it unique.
async function create(body: Record<string, unknown>, baseSlug: string): Promise<BlogPost> {
  try {
    return await api.post<BlogPost>('/blog', { ...body, locale: locale.value, slug: baseSlug });
  } catch (error) {
    if (!(error instanceof ApiError) || error.status !== 409) throw error;
    const suffix = Math.random().toString(36).slice(2, 6);
    return api.post<BlogPost>('/blog', {
      ...body,
      locale: locale.value,
      slug: `${baseSlug}-${suffix}`,
    });
  }
}

const saveMutation = useMutation({
  mutationFn: async (values: FormValues) => {
    let coverImageUrl = props.post?.coverImageUrl ?? null;
    if (coverFile.value) {
      const formData = new FormData();
      formData.append('file', coverFile.value);
      coverImageUrl = (await api.upload<{ url: string }>('/uploads/image', formData)).url;
    }
    const body = {
      title: values.title,
      excerpt: values.excerpt || null,
      content: values.content,
      coverImageUrl,
      isPublished: values.isPublished,
    };
    if (props.post) return api.put<BlogPost>(`/blog/${props.post.id}`, body);
    return create(body, slugify(values.title));
  },
  onSuccess: (post) => {
    queryClient.invalidateQueries({ queryKey: ['blog'] });
    queryClient.invalidateQueries({ queryKey: ['my-blog-posts'] });
    navigateTo(
      localePath(
        post.isPublished ? `/blog/${post.slug}` : { path: '/profile', query: { tab: 'listings' } },
      ),
    );
  },
  onError: () => {
    submitError.value = t('blog.form.saveError');
  },
});

const onSubmit = handleSubmit((values) => {
  submitError.value = null;
  saveMutation.mutate(values);
});
</script>

<template>
  <form class="mt-8 flex flex-col gap-6" @submit="onSubmit">
    <UFormGroup :label="$t('blog.form.title')" :error="errors.title">
      <UInput v-model="title" v-bind="titleAttrs" size="lg" />
    </UFormGroup>

    <UFormGroup :label="$t('blog.form.excerpt')" :error="errors.excerpt">
      <UInput v-model="excerpt" v-bind="excerptAttrs" />
    </UFormGroup>

    <UFormGroup :label="$t('blog.form.cover')">
      <input
        type="file"
        accept="image/*"
        class="block w-full text-sm text-text file:mr-3 file:rounded-md file:border-0 file:bg-brand-600 file:px-3 file:py-2 file:text-sm file:font-medium file:text-white"
        @change="onCoverChange"
      />
      <img v-if="coverPreviewUrl" :src="coverPreviewUrl" alt="" class="mt-3 h-48 w-full rounded-lg object-cover" />
    </UFormGroup>

    <UFormGroup :label="$t('blog.form.content')" :error="errors.content">
      <TiptapEditor v-model="content" :placeholder="$t('blog.form.contentPlaceholder')" />
    </UFormGroup>

    <div class="flex items-center gap-3">
      <UToggle v-model="isPublished" />
      <span class="text-sm text-text">{{ $t('blog.form.publish') }}</span>
    </div>

    <p v-if="submitError" class="text-sm text-brand-600">{{ submitError }}</p>

    <UButton type="submit" color="primary" block :loading="saveMutation.isPending.value">
      {{ post ? $t('blog.form.save') : $t('blog.form.create') }}
    </UButton>
  </form>
</template>
