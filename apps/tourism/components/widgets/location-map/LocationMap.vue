<script setup lang="ts">
import { mapsLink } from '~/shared/lib/contact-links';

const props = defineProps<{
  latitude: number | null;
  longitude: number | null;
  address: string | null;
}>();

// OpenStreetMap's embed needs no API key; ~1km box around the point.
const BBOX_DELTA = 0.01;

const embedUrl = computed(() => {
  if (props.latitude === null || props.longitude === null) return null;
  const { latitude: lat, longitude: lng } = props;
  const bbox = [lng - BBOX_DELTA, lat - BBOX_DELTA, lng + BBOX_DELTA, lat + BBOX_DELTA].join(',');
  return `https://www.openstreetmap.org/export/embed.html?bbox=${bbox}&layer=mapnik&marker=${lat},${lng}`;
});

const externalUrl = computed(() => mapsLink(props.latitude, props.longitude, props.address));
</script>

<template>
  <div v-if="embedUrl || address" class="flex flex-col gap-3">
    <p v-if="address" class="flex items-start gap-2 text-text">
      <UIcon name="i-heroicons-map-pin" class="mt-0.5 h-5 w-5 shrink-0 text-brand-600" />
      {{ address }}
    </p>
    <iframe
      v-if="embedUrl"
      :src="embedUrl"
      :title="address ?? $t('listing.map.title')"
      loading="lazy"
      class="h-64 w-full rounded-card border border-border"
    />
    <a
      v-if="externalUrl"
      :href="externalUrl"
      target="_blank"
      rel="noopener noreferrer"
      class="inline-flex items-center gap-1 text-sm text-primary-600 hover:underline dark:text-primary-400"
    >
      {{ $t('listing.map.openInGoogle') }}
      <UIcon name="i-heroicons-arrow-top-right-on-square" class="h-4 w-4" />
    </a>
  </div>
</template>
