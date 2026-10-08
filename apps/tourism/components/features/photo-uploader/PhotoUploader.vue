<script setup lang="ts">
import { useImageUpload } from '~/shared/lib/image-upload';
import { resolveMediaUrl } from '~/shared/lib/resolve-media-url';

const props = withDefaults(
  defineProps<{ max: number; folder: string; showCoverBadge?: boolean }>(),
  { showCoverBadge: false },
);
const model = defineModel<string[]>({ required: true });

const { t } = useI18n();
const { uploadImage } = useImageUpload();
const config = useRuntimeConfig();
const apiBaseUrl = config.public.apiBaseUrl;

const uploadingCount = ref(0);
const error = ref<string | null>(null);
const input = ref<HTMLInputElement | null>(null);

const remaining = computed(() => props.max - model.value.length - uploadingCount.value);

async function onFilesSelected(event: Event): Promise<void> {
  const target = event.target as HTMLInputElement;
  const files = Array.from(target.files ?? []).slice(0, Math.max(remaining.value, 0));
  target.value = '';
  error.value = null;

  uploadingCount.value += files.length;
  // Uploaded in parallel; each URL is appended as soon as it's ready so a
  // single failure doesn't discard the rest.
  await Promise.all(
    files.map(async (file) => {
      try {
        const url = await uploadImage(file, props.folder);
        model.value = [...model.value, url];
      } catch (uploadError) {
        error.value =
          uploadError instanceof Error && uploadError.message === 'unsupported-file-type'
            ? t('listing.photos.unsupportedFileType')
            : t('listing.photos.uploadError');
      } finally {
        uploadingCount.value -= 1;
      }
    }),
  );
}

function remove(index: number): void {
  model.value = model.value.filter((_, i) => i !== index);
}

function makeFirst(index: number): void {
  const next = [...model.value];
  const [photo] = next.splice(index, 1);
  if (photo) model.value = [photo, ...next];
}
</script>

<template>
  <div>
    <div class="grid grid-cols-3 gap-2 sm:grid-cols-4">
      <div
        v-for="(url, index) in model"
        :key="url"
        class="group relative aspect-square overflow-hidden rounded-lg border border-border"
      >
        <img :src="resolveMediaUrl(apiBaseUrl, url)" alt="" class="h-full w-full object-cover" />
        <span
          v-if="showCoverBadge && index === 0"
          class="absolute left-1 top-1 rounded bg-black/60 px-1.5 py-0.5 text-xs text-white"
        >
          {{ $t('listing.photos.cover') }}
        </span>
        <div class="absolute right-1 top-1 flex gap-1">
          <button
            v-if="showCoverBadge && index > 0"
            type="button"
            class="flex h-7 w-7 items-center justify-center rounded-full bg-black/60 text-white hover:bg-black/80"
            :aria-label="$t('listing.photos.makeCover')"
            :title="$t('listing.photos.makeCover')"
            @click="makeFirst(index)"
          >
            <UIcon name="i-heroicons-star" class="h-4 w-4" />
          </button>
          <button
            type="button"
            class="flex h-7 w-7 items-center justify-center rounded-full bg-black/60 text-white hover:bg-black/80"
            :aria-label="$t('listing.photos.remove')"
            @click="remove(index)"
          >
            <UIcon name="i-heroicons-x-mark" class="h-4 w-4" />
          </button>
        </div>
      </div>

      <div
        v-for="n in uploadingCount"
        :key="`uploading-${n}`"
        class="flex aspect-square items-center justify-center rounded-lg border border-dashed border-border"
      >
        <UIcon name="i-heroicons-arrow-path" class="h-6 w-6 animate-spin text-text-muted" />
      </div>

      <button
        v-if="remaining > 0"
        type="button"
        class="flex aspect-square flex-col items-center justify-center gap-1 rounded-lg border border-dashed border-border text-text-muted transition-colors hover:border-brand-500 hover:text-brand-600"
        @click="input?.click()"
      >
        <UIcon name="i-heroicons-photo" class="h-6 w-6" />
        <span class="text-xs">{{ $t('listing.photos.add') }}</span>
      </button>
    </div>

    <input
      ref="input"
      type="file"
      accept="image/jpeg,image/png,image/webp"
      :multiple="max > 1"
      class="hidden"
      @change="onFilesSelected"
    />
    <p v-if="error" class="mt-2 text-sm text-red-600">{{ error }}</p>
  </div>
</template>
