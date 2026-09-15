<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { useApiClient } from '~/shared/api/client';
import type { GalleryItem } from '~/shared/api/types';

const { t } = useI18n();
const api = useApiClient();
const auth = useAuth();
const queryClient = useQueryClient();
const config = useRuntimeConfig();
const apiBaseUrl = config.public.apiBaseUrl;

const galleryQuery = useQuery({
  queryKey: ['gallery'],
  queryFn: () => api.get<GalleryItem[]>('/gallery'),
});
await galleryQuery.suspense();

const items = galleryQuery.data;

const isOpen = ref(false);
const selected = ref<GalleryItem | null>(null);

const isAddOpen = ref(false);
const selectedFile = ref<File | null>(null);
const caption = ref('');
const submitError = ref<string | null>(null);

useSeoMeta({
  title: () => t('gallery.heading'),
  description: () => t('home.tagline'),
});

function openItem(item: GalleryItem): void {
  selected.value = item;
  isOpen.value = true;
}

function openAdd(): void {
  selectedFile.value = null;
  caption.value = '';
  submitError.value = null;
  isAddOpen.value = true;
}

function onFileChange(event: Event): void {
  const target = event.target as HTMLInputElement;
  selectedFile.value = target.files?.[0] ?? null;
}

const createMutation = useMutation({
  mutationFn: async () => {
    if (!selectedFile.value) throw new Error('no-file');

    const formData = new FormData();
    formData.append('file', selectedFile.value);
    const upload = await api.upload<{ url: string }>('/uploads/image', formData);

    return api.post<GalleryItem>('/gallery', {
      imageUrl: upload.url,
      caption: caption.value || null,
    });
  },
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ['gallery'] });
    isAddOpen.value = false;
  },
  onError: () => {
    submitError.value = t('gallery.uploadError');
  },
});

const deleteMutation = useMutation({
  mutationFn: (id: string) => api.delete(`/gallery/${id}`),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ['gallery'] });
    isOpen.value = false;
  },
});

function handleDelete(): void {
  if (!selected.value) return;
  if (!confirm(t('gallery.deleteConfirm'))) return;
  deleteMutation.mutate(selected.value.id);
}
</script>

<template>
  <section class="mx-auto max-w-6xl px-4 py-12">
    <div class="mb-8 flex items-center justify-between">
      <h1 class="text-3xl font-extrabold text-text">{{ $t('gallery.heading') }}</h1>
      <UButton v-if="auth.user.value" color="primary" icon="i-heroicons-plus" @click="openAdd">
        {{ $t('gallery.addButton') }}
      </UButton>
    </div>

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
        <UButton
          v-if="auth.user.value && selected.userId === auth.user.value.id"
          color="gray"
          variant="soft"
          class="mt-3"
          block
          :loading="deleteMutation.isPending.value"
          @click="handleDelete"
        >
          {{ $t('gallery.deleteButton') }}
        </UButton>
      </div>
    </UModal>

    <UModal v-model="isAddOpen">
      <div class="flex flex-col gap-4 p-4">
        <h2 class="text-lg font-semibold text-text">{{ $t('gallery.addTitle') }}</h2>

        <UFormGroup :label="$t('gallery.uploadLabel')">
          <input
            type="file"
            accept="image/*"
            class="block w-full text-sm text-text file:mr-3 file:rounded-md file:border-0 file:bg-brand-600 file:px-3 file:py-2 file:text-sm file:font-medium file:text-white"
            @change="onFileChange"
          />
        </UFormGroup>

        <UFormGroup :label="$t('gallery.captionLabel')">
          <UInput v-model="caption" :maxlength="500" />
        </UFormGroup>

        <p v-if="submitError" class="text-sm text-brand-600">{{ submitError }}</p>

        <UButton
          color="primary"
          block
          :disabled="!selectedFile"
          :loading="createMutation.isPending.value"
          @click="createMutation.mutate()"
        >
          {{ $t('gallery.submit') }}
        </UButton>
      </div>
    </UModal>
  </section>
</template>
