<script setup lang="ts">
import BlogPostForm from '~/components/features/blog-post-form/BlogPostForm.vue';

const { t } = useI18n();

useRequirePartner();

useSeoMeta({
  title: () => t('blog.form.newTitle'),
  robots: 'noindex',
});
</script>

<template>
  <section class="mx-auto max-w-3xl px-4 py-12">
    <h1 class="text-3xl font-extrabold text-text">{{ $t('blog.form.newTitle') }}</h1>
    <BlogPostForm />
  </section>
</template>
