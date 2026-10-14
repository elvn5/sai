<script setup lang="ts">
// One measure per day as thin bars on a zero baseline, with a tooltip per
// day. Two measures get two charts — never a second axis.
const props = defineProps<{
  title: string;
  points: { day: string; value: number }[];
}>();

const { locale } = useI18n();

const WIDTH = 600;
const HEIGHT = 120;
const GAP = 2;

const max = computed(() => Math.max(1, ...props.points.map((p) => p.value)));
const total = computed(() => props.points.reduce((sum, p) => sum + p.value, 0));
const step = computed(() => WIDTH / Math.max(1, props.points.length));
const hovered = ref<number | null>(null);

function barHeight(value: number): number {
  return value === 0 ? 0 : Math.max(3, (value / max.value) * (HEIGHT - 4));
}

function formatDay(day: string): string {
  return new Date(`${day}T12:00:00`).toLocaleDateString(locale.value, {
    day: 'numeric',
    month: 'short',
  });
}

const tooltip = computed(() => {
  if (hovered.value === null) return null;
  const point = props.points[hovered.value];
  if (!point) return null;
  return {
    text: `${formatDay(point.day)}: ${point.value}`,
    left: `${((hovered.value + 0.5) / props.points.length) * 100}%`,
  };
});
</script>

<template>
  <figure class="flex flex-col gap-2">
    <figcaption class="flex items-baseline justify-between gap-2">
      <span class="text-sm font-semibold text-text">{{ title }}</span>
      <span class="text-sm tabular-nums text-text-muted">{{ total }}</span>
    </figcaption>
    <div class="relative" @mouseleave="hovered = null">
      <svg
        :viewBox="`0 0 ${WIDTH} ${HEIGHT}`"
        preserveAspectRatio="none"
        class="h-28 w-full"
        role="img"
        :aria-label="`${title}: ${total}`"
      >
        <line :x1="0" :x2="WIDTH" :y1="HEIGHT - 0.5" :y2="HEIGHT - 0.5" class="stroke-border" stroke-width="1" />
        <g v-for="(point, index) in points" :key="point.day">
          <rect
            v-if="point.value > 0"
            :x="index * step + GAP / 2"
            :y="HEIGHT - barHeight(point.value)"
            :width="Math.max(1, step - GAP)"
            :height="barHeight(point.value)"
            rx="2"
            class="transition-opacity"
            :class="hovered === null || hovered === index ? 'fill-brand-500' : 'fill-brand-500 opacity-40'"
          />
          <!-- The hit target spans the whole column, bigger than the bar. -->
          <rect
            :x="index * step"
            y="0"
            :width="step"
            :height="HEIGHT"
            fill="transparent"
            @mouseenter="hovered = index"
            @touchstart.passive="hovered = index"
          />
        </g>
      </svg>
      <div
        v-if="tooltip"
        class="pointer-events-none absolute -top-8 -translate-x-1/2 whitespace-nowrap rounded-md border border-border bg-surface px-2 py-1 text-xs font-medium text-text shadow-sm"
        :style="{ left: tooltip.left }"
      >
        {{ tooltip.text }}
      </div>
    </div>
    <div v-if="points.length > 0" class="flex justify-between text-xs text-text-muted">
      <span>{{ formatDay(points[0]!.day) }}</span>
      <span>{{ formatDay(points[points.length - 1]!.day) }}</span>
    </div>
  </figure>
</template>
