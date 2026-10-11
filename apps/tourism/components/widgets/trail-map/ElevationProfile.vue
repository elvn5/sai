<script setup lang="ts">
import type { TrackPoint } from '~/shared/api/types';
import { cumulativeKm } from '~/shared/lib/gpx';

// An SVG elevation profile; hovering reports the point to the map.
const props = defineProps<{ track: TrackPoint[] }>();
const emit = defineEmits<{ hover: [number | null] }>();
const { t } = useI18n();

const WIDTH = 800;
const HEIGHT = 200;
const PAD = { top: 12, right: 12, bottom: 24, left: 44 };

interface ProfilePoint {
  i: number;
  km: number;
  ele: number;
}

const points = computed((): ProfilePoint[] => {
  const km = cumulativeKm(props.track);
  return props.track
    .map((p, i) => ({ i, km: km[i] as number, ele: p[2] }))
    .filter((p): p is ProfilePoint => p.ele !== null);
});
const hasProfile = computed(() => points.value.length > 1);
const totalKm = computed(() => points.value.at(-1)?.km ?? 0);
const minEle = computed(() => Math.min(...points.value.map((p: ProfilePoint) => p.ele)));
const maxEle = computed(() => Math.max(...points.value.map((p: ProfilePoint) => p.ele)));

function x(km: number): number {
  return PAD.left + (km / (totalKm.value || 1)) * (WIDTH - PAD.left - PAD.right);
}
function y(ele: number): number {
  const span = maxEle.value - minEle.value || 1;
  return PAD.top + (1 - (ele - minEle.value) / span) * (HEIGHT - PAD.top - PAD.bottom);
}

const linePath = computed(() =>
  points.value
    .map(
      (p: ProfilePoint, n: number) =>
        `${n === 0 ? 'M' : 'L'}${x(p.km).toFixed(1)},${y(p.ele).toFixed(1)}`,
    )
    .join(''),
);
const areaPath = computed(() =>
  hasProfile.value
    ? `${linePath.value}L${x(totalKm.value).toFixed(1)},${HEIGHT - PAD.bottom}L${PAD.left},${HEIGHT - PAD.bottom}Z`
    : '',
);
const ticks = computed(() => {
  const step = Math.max(100, Math.ceil((maxEle.value - minEle.value) / 4 / 100) * 100);
  const out: number[] = [];
  for (let e = Math.ceil(minEle.value / step) * step; e <= maxEle.value; e += step) out.push(e);
  return out;
});

const hovered = ref<ProfilePoint | null>(null);

function onMove(event: MouseEvent | TouchEvent): void {
  const svg = event.currentTarget as SVGSVGElement;
  const rect = svg.getBoundingClientRect();
  const clientX = 'touches' in event ? (event.touches[0]?.clientX ?? 0) : event.clientX;
  const km =
    ((((clientX - rect.left) / rect.width) * WIDTH - PAD.left) / (WIDTH - PAD.left - PAD.right)) *
    totalKm.value;
  let best = points.value[0];
  for (const p of points.value) {
    if (best && Math.abs(p.km - km) < Math.abs(best.km - km)) best = p;
  }
  hovered.value = best ?? null;
  emit('hover', best?.i ?? null);
}

function onLeave(): void {
  hovered.value = null;
  emit('hover', null);
}
</script>

<template>
  <div v-if="hasProfile" class="rounded-card border border-border bg-surface p-3">
    <svg
      :viewBox="`0 0 ${WIDTH} ${HEIGHT}`"
      class="h-auto w-full touch-none select-none"
      role="img"
      :aria-label="t('trails.detail.profile')"
      @mousemove="onMove"
      @touchmove.passive="onMove"
      @mouseleave="onLeave"
      @touchend="onLeave"
    >
      <defs>
        <linearGradient id="profile-fill" x1="0" x2="0" y1="0" y2="1">
          <stop offset="0%" stop-color="rgb(var(--color-brand-500))" stop-opacity="0.45" />
          <stop offset="100%" stop-color="rgb(var(--color-brand-500))" stop-opacity="0.03" />
        </linearGradient>
      </defs>
      <g v-for="tick in ticks" :key="tick">
        <line :x1="PAD.left" :x2="WIDTH - PAD.right" :y1="y(tick)" :y2="y(tick)" stroke="currentColor" class="text-border" stroke-dasharray="3 4" />
        <text :x="PAD.left - 6" :y="y(tick) + 4" text-anchor="end" class="fill-current text-[11px] text-text-muted">{{ tick }}</text>
      </g>
      <path :d="areaPath" fill="url(#profile-fill)" />
      <path :d="linePath" fill="none" stroke="rgb(var(--color-brand-500))" stroke-width="2.5" stroke-linejoin="round" />
      <text :x="WIDTH - PAD.right" :y="HEIGHT - 6" text-anchor="end" class="fill-current text-[11px] text-text-muted">
        {{ totalKm.toFixed(1) }} {{ t('trails.units.km') }}
      </text>
      <g v-if="hovered">
        <line :x1="x(hovered.km)" :x2="x(hovered.km)" :y1="PAD.top" :y2="HEIGHT - PAD.bottom" stroke="currentColor" class="text-text-muted" />
        <circle :cx="x(hovered.km)" :cy="y(hovered.ele)" r="5" fill="#2563eb" stroke="#fff" stroke-width="2" />
        <text
          :x="Math.min(x(hovered.km) + 8, WIDTH - 130)"
          :y="PAD.top + 14"
          class="fill-current text-[12px] font-semibold text-text"
        >
          {{ Math.round(hovered.ele) }} {{ t('trails.units.m') }} · {{ hovered.km.toFixed(1) }} {{ t('trails.units.km') }}
        </text>
      </g>
    </svg>
  </div>
</template>
