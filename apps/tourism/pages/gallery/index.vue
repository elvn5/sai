<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import { useApiClient } from '~/shared/api/client';
import type { GalleryItem } from '~/shared/api/types';

const { t } = useI18n();
const api = useApiClient();
const config = useRuntimeConfig();
const apiBaseUrl = config.public.apiBaseUrl;

const galleryQuery = useQuery({
  queryKey: ['gallery'],
  queryFn: () => api.get<GalleryItem[]>('/gallery'),
});
if (import.meta.server) await galleryQuery.suspense();

const items = galleryQuery.data;

const isOpen = ref(false);
const selected = ref<GalleryItem | null>(null);

useSeoMeta({
  title: () => t('gallery.heading'),
  description: () => t('home.tagline'),
});

function openItem(item: GalleryItem): void {
  selected.value = item;
  isOpen.value = true;
}
</script>

<template>
  <section class="mx-auto max-w-6xl px-4 py-12">
    <h1 class="mb-8 text-3xl font-extrabold text-text">{{ $t('gallery.heading') }}</h1>

    <div v-if="items && items.length > 0" class="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
      <button
        v-for="item in items"
        :key="item.id"
        type="button"
        class="group overflow-hidden rounded-card border border-border"
        @click="openItem(item)"
      >
        <img
          :src="`${apiBaseUrl}${item.imageUrl}`"
          :alt="item.altText ?? ''"
          class="h-40 w-full object-cover transition group-hover:scale-105"
        />
      </button>
    </div>
    <p v-else class="text-text-muted">{{ $t('gallery.empty') }}</p>

    <UModal v-model="isOpen">
      <div v-if="selected" class="p-2">
        <img
          :src="`${apiBaseUrl}${selected.imageUrl}`"
          :alt="selected.altText ?? ''"
          class="w-full rounded-lg"
        />
        <p v-if="selected.caption" class="mt-2 text-center text-sm text-text-muted">
          {{ selected.caption }}
        </p>
      </div>
    </UModal>
  </section>
</template>
