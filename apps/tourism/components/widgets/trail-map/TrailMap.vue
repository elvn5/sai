<script setup lang="ts">
import type { Map as LeafletMap } from 'leaflet';
import 'leaflet/dist/leaflet.css';
import type { TrackPoint } from '~/shared/api/types';

// A topographic map with the route's track (or just its start point).
// Leaflet touches `window`, so it loads only in the browser.
const props = defineProps<{
  track: TrackPoint[];
  startLatitude: number | null;
  startLongitude: number | null;
  // Index into the track to mark (from the elevation profile hover).
  highlight?: number | null;
}>();

const container = ref<HTMLDivElement | null>(null);
let map: LeafletMap | null = null;
let marker: import('leaflet').CircleMarker | null = null;

onMounted(async () => {
  const L = await import('leaflet');
  if (!container.value) return;
  map = L.map(container.value, { scrollWheelZoom: false });
  L.tileLayer('https://{s}.tile.opentopomap.org/{z}/{x}/{y}.png', {
    maxZoom: 17,
    attribution:
      '© <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>, SRTM | © <a href="https://opentopomap.org">OpenTopoMap</a> (CC-BY-SA)',
  }).addTo(map);

  const latlngs = props.track.map((p) => [p[0], p[1]] as [number, number]);
  if (latlngs.length > 1) {
    const line = L.polyline(latlngs, { color: '#ef4444', weight: 4, opacity: 0.9 }).addTo(map);
    L.circleMarker(latlngs[0] as [number, number], {
      radius: 7,
      color: '#fff',
      weight: 2,
      fillColor: '#16a34a',
      fillOpacity: 1,
    }).addTo(map);
    L.circleMarker(latlngs[latlngs.length - 1] as [number, number], {
      radius: 7,
      color: '#fff',
      weight: 2,
      fillColor: '#dc2626',
      fillOpacity: 1,
    }).addTo(map);
    map.fitBounds(line.getBounds(), { padding: [24, 24] });
  } else if (props.startLatitude !== null && props.startLongitude !== null) {
    const start: [number, number] = [props.startLatitude, props.startLongitude];
    L.circleMarker(start, {
      radius: 8,
      color: '#fff',
      weight: 2,
      fillColor: '#16a34a',
      fillOpacity: 1,
    }).addTo(map);
    map.setView(start, 13);
  }
  marker = L.circleMarker([0, 0], {
    radius: 6,
    color: '#fff',
    weight: 2,
    fillColor: '#2563eb',
    fillOpacity: 1,
  });
});

watch(
  () => props.highlight,
  (index: number | null | undefined) => {
    if (!map || !marker) return;
    const point = index !== null && index !== undefined ? props.track[index] : undefined;
    if (point) {
      marker.setLatLng([point[0], point[1]]).addTo(map);
    } else {
      marker.remove();
    }
  },
);

onBeforeUnmount(() => {
  map?.remove();
  map = null;
});
</script>

<template>
  <div ref="container" class="trail-map h-80 w-full overflow-hidden rounded-card border border-border sm:h-96" />
</template>

<style scoped>
.trail-map {
  z-index: 0;
}
</style>
