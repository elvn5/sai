<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import BlogPostForm from '~/components/features/blog-post-form/BlogPostForm.vue';
import { useApiClient } from '~/shared/api/client';
import type { BlogPost } from '~/shared/api/types';

const { t } = useI18n();
const route = useRoute();
const api = useApiClient();
const auth = useAuth();

useRequirePartner();

const postsQuery = useQuery({
  queryKey: ['my-blog-posts'],
  queryFn: () => api.get<BlogPost[]>('/blog/mine'),
  enabled: auth.isPartner,
});
const post = computed(() => postsQuery.data.value?.find((p: BlogPost) => p.id === route.params.id));

useSeoMeta({
  title: () => t('blog.form.editTitle'),
  robots: 'noindex',
});
</script>

<template>
  <section class="mx-auto max-w-3xl px-4 py-12">
    <h1 class="text-3xl font-extrabold text-text">{{ $t('blog.form.editTitle') }}</h1>
    <BlogPostForm v-if="post" :key="post.id" :post="post" />
    <p v-else-if="postsQuery.isFetched.value" class="mt-8 text-text-muted">{{ $t('blog.form.notFound') }}</p>
  </section>
</template>
