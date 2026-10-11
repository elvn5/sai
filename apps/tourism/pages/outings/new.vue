<script setup lang="ts">
import OutingForm from '~/components/features/outing-form/OutingForm.vue';
import TelegramNotice from '~/components/features/telegram-notice/TelegramNotice.vue';

const { t } = useI18n();
const route = useRoute();
const trailSlug = typeof route.query.trail === 'string' ? route.query.trail : undefined;

useRequireAuth();
useSeoMeta({ title: () => t('outings.form.newTitle'), robots: 'noindex' });
</script>

<template>
  <section class="mx-auto max-w-2xl px-4 py-12">
    <h1 class="text-3xl font-extrabold text-text">{{ $t('outings.form.newTitle') }}</h1>
    <p class="mt-2 text-text-muted">{{ $t('outings.form.newHint') }}</p>
    <div class="mt-4"><TelegramNotice /></div>
    <OutingForm :trail-slug="trailSlug" />
  </section>
</template>
