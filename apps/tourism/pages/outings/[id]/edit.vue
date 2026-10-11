<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import OutingForm from '~/components/features/outing-form/OutingForm.vue';
import { useApiClient } from '~/shared/api/client';
import type { Outing } from '~/shared/api/types';

const { t } = useI18n();
const route = useRoute();
const api = useApiClient();
const id = String(route.params.id);

useRequireAuth();
const outingQuery = useQuery({
  queryKey: ['outing', id],
  queryFn: () => api.get<Outing>(`/outings/${id}`),
});
useSeoMeta({ title: () => t('outings.form.editTitle'), robots: 'noindex' });
</script>

<template>
  <section class="mx-auto max-w-2xl px-4 py-12">
    <h1 class="text-3xl font-extrabold text-text">{{ $t('outings.form.editTitle') }}</h1>
    <OutingForm v-if="outingQuery.data.value?.isOrganizer" :key="outingQuery.data.value.id" :outing="outingQuery.data.value" />
    <p v-else-if="outingQuery.isFetched.value" class="mt-8 text-text-muted">{{ $t('outings.notYours') }}</p>
  </section>
</template>
