<script setup lang="ts">
import TrailForm from '~/components/features/trail-form/TrailForm.vue';

const { t } = useI18n();
useRequirePartner();
useSeoMeta({ title: () => t('trails.form.newTitle'), robots: 'noindex' });
</script>

<template>
  <section class="mx-auto max-w-3xl px-4 py-12">
    <h1 class="text-3xl font-extrabold text-text">{{ $t('trails.form.newTitle') }}</h1>
    <TrailForm />
  </section>
</template>
