<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import TrailForm from '~/components/features/trail-form/TrailForm.vue';
import { useApiClient } from '~/shared/api/client';
import type { Trail } from '~/shared/api/types';

const { t } = useI18n();
const route = useRoute();
const api = useApiClient();
const auth = useAuth();
const slug = String(route.params.slug);

useRequirePartner();
const trailQuery = useQuery({
  queryKey: ['trail', slug],
  queryFn: () => api.get<Trail>(`/trails/${slug}`),
});
const editable = computed(
  () => trailQuery.data.value && trailQuery.data.value.userId === auth.user.value?.id,
);
useSeoMeta({ title: () => t('trails.form.editTitle'), robots: 'noindex' });
</script>

<template>
  <section class="mx-auto max-w-3xl px-4 py-12">
    <h1 class="text-3xl font-extrabold text-text">{{ $t('trails.form.editTitle') }}</h1>
    <TrailForm v-if="editable && trailQuery.data.value" :key="trailQuery.data.value.id" :trail="trailQuery.data.value" />
    <p v-else-if="trailQuery.isFetched.value && auth.checked.value" class="mt-8 text-text-muted">{{ $t('trails.notYours') }}</p>
  </section>
</template>
