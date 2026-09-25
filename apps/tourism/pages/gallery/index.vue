<script setup lang="ts">
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { useApiClient } from '~/shared/api/client';
import { SeoPage } from '~/shared/api/types';
import type { GalleryItem } from '~/shared/api/types';
import { resolveMediaUrl } from '~/shared/lib/resolve-media-url';

const PAGE_SIZE = 20;
const SEARCH_DEBOUNCE_MS = 300;
const ALLOWED_IMAGE_TYPES = new Set(['image/jpeg', 'image/png', 'image/webp']);

const { t, locale } = useI18n();
const api = useApiClient();
const auth = useAuth();
const queryClient = useQueryClient();
const config = useRuntimeConfig();
const apiBaseUrl = config.public.apiBaseUrl;

const activeLocation = ref<string | null>(null);
const searchInput = ref('');
const search = ref('');
let searchDebounceTimer: ReturnType<typeof setTimeout> | undefined;

watch(searchInput, (value) => {
  clearTimeout(searchDebounceTimer);
  searchDebounceTimer = setTimeout(() => {
    search.value = value.trim();
  }, SEARCH_DEBOUNCE_MS);
});

const locationsQuery = useQuery({
  queryKey: ['gallery-locations'],
  queryFn: () => api.get<string[]>('/gallery/locations'),
});
await locationsQuery.suspense();
const locations = locationsQuery.data;

const galleryQuery = useInfiniteQuery({
  queryKey: computed(() => ['gallery', activeLocation.value, search.value] as const),
  queryFn: ({ pageParam }) => {
    const params = new URLSearchParams({ limit: String(PAGE_SIZE), offset: String(pageParam) });
    if (activeLocation.value) params.set('location', activeLocation.value);
    if (search.value) params.set('search', search.value);
    return api.get<GalleryItem[]>(`/gallery?${params.toString()}`);
  },
  initialPageParam: 0,
  getNextPageParam: (lastPage, allPages) =>
    lastPage.length === PAGE_SIZE ? allPages.length * PAGE_SIZE : undefined,
});
await galleryQuery.suspense();

const items = computed(() => galleryQuery.data.value?.pages.flat() ?? []);

const sentinel = ref<HTMLElement | null>(null);
let observer: IntersectionObserver | null = null;

function handleIntersect(entries: IntersectionObserverEntry[]): void {
  if (
    entries[0]?.isIntersecting &&
    galleryQuery.hasNextPage.value &&
    !galleryQuery.isFetchingNextPage.value
  ) {
    galleryQuery.fetchNextPage();
  }
}

onMounted(() => {
  observer = new IntersectionObserver(handleIntersect, { rootMargin: '200px' });
  watch(
    sentinel,
    (el, prevEl) => {
      if (prevEl) observer?.unobserve(prevEl);
      if (el) observer?.observe(el);
    },
    { immediate: true },
  );
});

onBeforeUnmount(() => {
  observer?.disconnect();
});

const isAddOpen = ref(false);
const selectedFile = ref<File | null>(null);
const caption = ref('');
const formLocation = ref('');
const submitError = ref<string | null>(null);

const pageSeoQuery = usePageSeo(SeoPage.GALLERY, locale);
await pageSeoQuery.suspense();
const pageSeo = pageSeoQuery.data;

useSeoMeta({
  title: () => pageSeo.value?.title || t('gallery.heading'),
  description: () => pageSeo.value?.description || t('gallery.metaDescription'),
  ogTitle: () => pageSeo.value?.title || t('gallery.heading'),
  ogDescription: () => pageSeo.value?.description || t('gallery.metaDescription'),
  ogImage: () =>
    pageSeo.value?.ogImageUrl ? `${apiBaseUrl}${pageSeo.value.ogImageUrl}` : undefined,
});

const lightboxIndex = ref<number | null>(null);
const currentItem = computed(() =>
  lightboxIndex.value !== null ? (items.value[lightboxIndex.value] ?? null) : null,
);

function openLightbox(index: number): void {
  lightboxIndex.value = index;
}

function closeLightbox(): void {
  lightboxIndex.value = null;
}

async function showNext(): Promise<void> {
  if (lightboxIndex.value === null) return;
  const nextIndex = lightboxIndex.value + 1;
  if (nextIndex >= items.value.length) {
    if (galleryQuery.hasNextPage.value && !galleryQuery.isFetchingNextPage.value) {
      await galleryQuery.fetchNextPage();
      if (nextIndex < items.value.length) lightboxIndex.value = nextIndex;
    }
    return;
  }
  lightboxIndex.value = nextIndex;
}

function showPrevious(): void {
  if (lightboxIndex.value === null || lightboxIndex.value === 0) return;
  lightboxIndex.value -= 1;
}

function onKeydown(event: KeyboardEvent): void {
  if (lightboxIndex.value === null) return;
  if (event.key === 'ArrowRight') void showNext();
  else if (event.key === 'ArrowLeft') showPrevious();
  else if (event.key === 'Escape') closeLightbox();
}

onMounted(() => window.addEventListener('keydown', onKeydown));
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown));

function openAdd(): void {
  selectedFile.value = null;
  caption.value = '';
  formLocation.value = '';
  submitError.value = null;
  isAddOpen.value = true;
}

function onFileChange(event: Event): void {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0] ?? null;
  if (file && !ALLOWED_IMAGE_TYPES.has(file.type)) {
    submitError.value = t('gallery.unsupportedFileType');
    selectedFile.value = null;
    target.value = '';
    return;
  }
  submitError.value = null;
  selectedFile.value = file;
}

const createMutation = useMutation({
  mutationFn: async () => {
    const file = selectedFile.value;
    if (!file) throw new Error('missing-fields');

    // Direct-to-R2 upload: the file goes straight from the browser to the
    // bucket via a presigned URL, never through this server.
    const { uploadUrl, publicUrl } = await api.post<{
      uploadUrl: string;
      key: string;
      publicUrl: string;
    }>('/storage/upload-url', {
      filename: file.name,
      contentType: file.type,
      folder: 'gallery',
    });

    const uploadResponse = await fetch(uploadUrl, {
      method: 'PUT',
      headers: { 'Content-Type': file.type },
      body: file,
    });
    if (!uploadResponse.ok) throw new Error('upload-failed');

    return api.post<GalleryItem>('/gallery', {
      imageUrl: publicUrl,
      location: formLocation.value.trim() || null,
      caption: caption.value || null,
    });
  },
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ['gallery'] });
    queryClient.invalidateQueries({ queryKey: ['gallery-locations'] });
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
    queryClient.invalidateQueries({ queryKey: ['gallery-locations'] });
    closeLightbox();
  },
});

function handleDelete(): void {
  if (!currentItem.value) return;
  if (!confirm(t('gallery.deleteConfirm'))) return;
  deleteMutation.mutate(currentItem.value.id);
}
</script>

<template>
  <section class="mx-auto max-w-6xl px-4 py-12">
    <div class="mb-6 flex items-center justify-between">
      <h1 class="text-3xl font-extrabold text-text">{{ $t('gallery.heading') }}</h1>
      <UButton v-if="auth.user.value" color="primary" icon="i-heroicons-plus" @click="openAdd">
        {{ $t('gallery.addButton') }}
      </UButton>
    </div>

    <UInput
      v-model="searchInput"
      icon="i-heroicons-magnifying-glass"
      :placeholder="$t('gallery.searchPlaceholder')"
      class="mb-6"
    />

    <div v-if="locations && locations.length > 0" class="mb-8 flex flex-wrap gap-2">
      <button
        type="button"
        class="rounded-full border px-4 py-1.5 text-sm font-medium transition-colors"
        :class="
          activeLocation === null
            ? 'border-brand-500 bg-brand-500/10 text-brand-600 dark:text-brand-400'
            : 'border-border text-text-muted hover:border-brand-500/50 hover:text-text'
        "
        @click="activeLocation = null"
      >
        {{ $t('gallery.allLocations') }}
      </button>
      <button
        v-for="loc in locations"
        :key="loc"
        type="button"
        class="rounded-full border px-4 py-1.5 text-sm font-medium transition-colors"
        :class="
          activeLocation === loc
            ? 'border-brand-500 bg-brand-500/10 text-brand-600 dark:text-brand-400'
            : 'border-border text-text-muted hover:border-brand-500/50 hover:text-text'
        "
        @click="activeLocation = loc"
      >
        {{ loc }}
      </button>
    </div>

    <div v-if="items.length > 0" class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
      <button
        v-for="(item, index) in items"
        :key="item.id"
        type="button"
        class="group relative aspect-square overflow-hidden rounded-card border border-border"
        @click="openLightbox(index)"
      >
        <img
          :src="resolveMediaUrl(apiBaseUrl, item.imageUrl)"
          :alt="item.altText ?? ''"
          loading="lazy"
          class="h-full w-full object-cover transition group-hover:scale-105"
        />
        <span
          v-if="item.location"
          class="absolute inset-x-0 bottom-0 truncate bg-gradient-to-t from-black/70 to-transparent px-2 py-1.5 text-left text-xs text-white"
        >
          <UIcon name="i-heroicons-map-pin" class="mr-1 h-3 w-3" />{{ item.location }}
        </span>
      </button>
    </div>
    <p v-else class="text-text-muted">{{ $t('gallery.empty') }}</p>

    <div ref="sentinel" class="h-1" />
    <p v-if="galleryQuery.isFetchingNextPage.value" class="mt-4 text-center text-sm text-text-muted">
      {{ $t('gallery.loadingMore') }}
    </p>

    <Teleport to="body">
      <div v-if="currentItem" class="fixed inset-0 z-50 flex flex-col bg-black/95" @click.self="closeLightbox">
        <button
          type="button"
          class="absolute right-4 top-4 z-10 flex h-10 w-10 items-center justify-center rounded-full bg-white/10 text-white hover:bg-white/20"
          :aria-label="$t('gallery.close')"
          @click="closeLightbox"
        >
          <UIcon name="i-heroicons-x-mark" class="h-6 w-6" />
        </button>

        <button
          v-if="lightboxIndex !== 0"
          type="button"
          class="absolute left-2 top-1/2 z-10 flex h-12 w-12 -translate-y-1/2 items-center justify-center rounded-full bg-white/10 text-white hover:bg-white/20 sm:left-4"
          :aria-label="$t('gallery.previous')"
          @click="showPrevious"
        >
          <UIcon name="i-heroicons-chevron-left" class="h-7 w-7" />
        </button>

        <button
          v-if="lightboxIndex !== null && (lightboxIndex < items.length - 1 || galleryQuery.hasNextPage.value)"
          type="button"
          class="absolute right-2 top-1/2 z-10 flex h-12 w-12 -translate-y-1/2 items-center justify-center rounded-full bg-white/10 text-white hover:bg-white/20 sm:right-4"
          :aria-label="$t('gallery.next')"
          @click="showNext"
        >
          <UIcon name="i-heroicons-chevron-right" class="h-7 w-7" />
        </button>

        <div class="flex flex-1 items-center justify-center p-4 sm:p-12" @click.self="closeLightbox">
          <img
            :src="resolveMediaUrl(apiBaseUrl, currentItem.imageUrl)"
            :alt="currentItem.altText ?? ''"
            class="max-h-full max-w-full object-contain"
          />
        </div>

        <div class="flex flex-col items-center gap-2 pb-6 text-center text-white">
          <p v-if="currentItem.location" class="flex items-center gap-1 text-sm text-white/80">
            <UIcon name="i-heroicons-map-pin" class="h-4 w-4" />{{ currentItem.location }}
          </p>
          <p v-if="currentItem.caption" class="max-w-lg px-4 text-sm text-white/90">
            {{ currentItem.caption }}
          </p>
          <UButton
            v-if="auth.user.value && currentItem.userId === auth.user.value.id"
            color="gray"
            variant="soft"
            size="sm"
            :loading="deleteMutation.isPending.value"
            @click="handleDelete"
          >
            {{ $t('gallery.deleteButton') }}
          </UButton>
        </div>
      </div>
    </Teleport>

    <UModal v-model="isAddOpen">
      <div class="flex flex-col gap-4 p-4">
        <h2 class="text-lg font-semibold text-text">{{ $t('gallery.addTitle') }}</h2>

        <UFormGroup :label="$t('gallery.uploadLabel')">
          <input
            type="file"
            accept="image/jpeg,image/png,image/webp"
            class="block w-full text-sm text-text file:mr-3 file:rounded-md file:border-0 file:bg-brand-600 file:px-3 file:py-2 file:text-sm file:font-medium file:text-white"
            @change="onFileChange"
          />
        </UFormGroup>

        <UFormGroup :label="$t('gallery.locationLabel')" :hint="$t('gallery.locationHint')">
          <UInput v-model="formLocation" :maxlength="255" />
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
