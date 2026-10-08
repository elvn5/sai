<script setup lang="ts">
import { resolveMediaUrl } from '~/shared/lib/resolve-media-url';

const props = withDefaults(
  defineProps<{ photos: string[]; alt?: string; variant?: 'hero' | 'compact' }>(),
  {
    alt: '',
    variant: 'compact',
  },
);

const config = useRuntimeConfig();
const apiBaseUrl = config.public.apiBaseUrl;

const HERO_VISIBLE = 5;
const openIndex = ref<number | null>(null);

const visible = computed(() =>
  props.variant === 'hero' ? props.photos.slice(0, HERO_VISIBLE) : props.photos,
);
const hiddenCount = computed(() => props.photos.length - visible.value.length);

function src(url: string): string {
  return resolveMediaUrl(apiBaseUrl, url);
}

function step(delta: number): void {
  if (openIndex.value === null) return;
  const count = props.photos.length;
  openIndex.value = (openIndex.value + delta + count) % count;
}

function onKeydown(event: KeyboardEvent): void {
  if (openIndex.value === null) return;
  if (event.key === 'ArrowRight') step(1);
  else if (event.key === 'ArrowLeft') step(-1);
  else if (event.key === 'Escape') openIndex.value = null;
}

onMounted(() => window.addEventListener('keydown', onKeydown));
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown));
</script>

<template>
  <div v-if="photos.length > 0">
    <div
      v-if="variant === 'hero'"
      class="grid h-64 grid-cols-4 grid-rows-2 gap-2 overflow-hidden rounded-card sm:h-96"
    >
      <button
        v-for="(url, index) in visible"
        :key="url"
        type="button"
        class="relative overflow-hidden"
        :class="index === 0 ? 'col-span-4 row-span-2 sm:col-span-2' : 'hidden sm:block'"
        @click="openIndex = index"
      >
        <img :src="src(url)" :alt="alt" class="h-full w-full object-cover transition hover:scale-105" />
        <span
          v-if="index === visible.length - 1 && hiddenCount > 0"
          class="absolute inset-0 flex items-center justify-center bg-black/50 text-lg font-semibold text-white"
        >
          +{{ hiddenCount }}
        </span>
      </button>
    </div>

    <div v-else class="grid grid-cols-3 gap-2 sm:grid-cols-4">
      <button
        v-for="(url, index) in visible"
        :key="url"
        type="button"
        class="aspect-square overflow-hidden rounded-lg"
        @click="openIndex = index"
      >
        <img :src="src(url)" :alt="alt" loading="lazy" class="h-full w-full object-cover transition hover:scale-105" />
      </button>
    </div>

    <Teleport to="body">
      <div
        v-if="openIndex !== null"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/95 p-4"
        @click.self="openIndex = null"
      >
        <button
          type="button"
          class="absolute right-4 top-4 flex h-10 w-10 items-center justify-center rounded-full bg-white/10 text-white hover:bg-white/20"
          :aria-label="$t('gallery.close')"
          @click="openIndex = null"
        >
          <UIcon name="i-heroicons-x-mark" class="h-6 w-6" />
        </button>
        <button
          v-if="photos.length > 1"
          type="button"
          class="absolute left-2 top-1/2 flex h-12 w-12 -translate-y-1/2 items-center justify-center rounded-full bg-white/10 text-white hover:bg-white/20"
          :aria-label="$t('gallery.previous')"
          @click="step(-1)"
        >
          <UIcon name="i-heroicons-chevron-left" class="h-7 w-7" />
        </button>
        <img :src="src(photos[openIndex] ?? '')" :alt="alt" class="max-h-[85vh] max-w-[90vw] object-contain" />
        <button
          v-if="photos.length > 1"
          type="button"
          class="absolute right-2 top-1/2 flex h-12 w-12 -translate-y-1/2 items-center justify-center rounded-full bg-white/10 text-white hover:bg-white/20"
          :aria-label="$t('gallery.next')"
          @click="step(1)"
        >
          <UIcon name="i-heroicons-chevron-right" class="h-7 w-7" />
        </button>
        <span class="absolute bottom-4 text-sm text-white/70">{{ openIndex + 1 }} / {{ photos.length }}</span>
      </div>
    </Teleport>
  </div>
</template>
