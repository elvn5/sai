<script setup lang="ts">
import RegionBanner from '~/components/features/region-picker/RegionBanner.vue';
import type { GalleryRegion } from '~/shared/api/types';
import { KYRGYZSTAN_REGIONS } from '~/shared/lib/kyrgyzstan-regions';

const props = defineProps<{ modelValue: GalleryRegion | undefined }>();
const emit = defineEmits<{ 'update:modelValue': [GalleryRegion] }>();

const { t } = useI18n();

function label(region: GalleryRegion): string {
  return t(`gallery.regions.${region}`);
}
</script>

<template>
  <div class="flex flex-wrap gap-3">
    <button
      v-for="region in KYRGYZSTAN_REGIONS"
      :key="region"
      type="button"
      class="flex w-20 flex-col items-center gap-1.5 rounded-card border px-2 py-2 text-xs font-medium transition-colors"
      :class="
        props.modelValue === region
          ? 'icon-select-active border-brand-500 bg-brand-500/10 text-brand-600 dark:text-brand-400'
          : 'border-border text-text-muted hover:border-brand-500/50 hover:text-text'
      "
      :aria-pressed="props.modelValue === region"
      @click="emit('update:modelValue', region)"
    >
      <span class="h-8 w-11 overflow-hidden rounded-sm border border-black/10">
        <RegionBanner :region="region" />
      </span>
      <span class="text-center leading-tight">{{ label(region) }}</span>
    </button>
  </div>
</template>
