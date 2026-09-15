<script setup lang="ts">
import { GalleryRegion } from '~/shared/api/types';

defineProps<{ region: GalleryRegion }>();
</script>

<template>
  <svg viewBox="0 0 40 28" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" class="h-full w-full rounded-sm">
    <template v-if="region === GalleryRegion.BATKEN">
      <rect width="40" height="28" fill="#e08a3c" />
      <circle cx="9" cy="8" r="3.5" fill="#fff3d6" />
      <path d="M0 28 L8 15 L14 22 L20 11 L27 20 L33 14 L40 28 Z" fill="#a85a1f" />
    </template>

    <template v-else-if="region === GalleryRegion.JALAL_ABAD">
      <rect width="40" height="28" fill="#2f7a3d" />
      <path
        d="M20 5 C27 9 27 19 20 24 C13 19 13 9 20 5 Z"
        fill="none"
        stroke="#eafaf0"
        stroke-width="1.6"
      />
      <line x1="20" y1="6" x2="20" y2="23" stroke="#eafaf0" stroke-width="1.2" />
    </template>

    <template v-else-if="region === GalleryRegion.ISSYK_KUL">
      <rect width="40" height="28" fill="#1f6fb2" />
      <circle cx="31" cy="7" r="3" fill="#fdeda3" />
      <path d="M0 18 Q5 14 10 18 T20 18 T30 18 T40 18 V28 H0 Z" fill="#bfe1f7" />
      <path d="M0 23 Q5 20 10 23 T20 23 T30 23 T40 23 V28 H0 Z" fill="#eaf6fd" />
    </template>

    <template v-else-if="region === GalleryRegion.NARYN">
      <rect width="40" height="28" fill="#516a82" />
      <path d="M0 28 L7 12 L13 20 L20 6 L27 20 L33 12 L40 28 Z" fill="#eef2f5" />
    </template>

    <template v-else-if="region === GalleryRegion.OSH">
      <rect width="40" height="28" fill="#c8402b" />
      <circle cx="20" cy="14" r="6.5" fill="#fbe4a0" />
    </template>

    <template v-else-if="region === GalleryRegion.TALAS">
      <rect width="40" height="28" fill="#5b4b8a" />
      <path d="M20 4 L21.6 8 L26 8 L22.4 10.6 L23.8 15 L20 12.2 L16.2 15 L17.6 10.6 L14 8 L18.4 8 Z" fill="#f4eefc" />
      <path d="M13 24 V16 A7 7 0 0 1 27 16 V24 Z" fill="#f4eefc" />
    </template>

    <template v-else-if="region === GalleryRegion.CHUY">
      <rect width="40" height="28" fill="#c9a227" />
      <path
        d="M20 24 L20 9 M20 9 L15 4 M20 9 L25 4 M20 14 L15.5 9.5 M20 14 L24.5 9.5 M20 19 L16 15 M20 19 L24 15"
        stroke="#fdf8e3"
        stroke-width="1.4"
        stroke-linecap="round"
        fill="none"
      />
    </template>
  </svg>
</template>
