<script setup lang="ts">
import PersonAvatar from '~/components/entities/achievement/PersonAvatar.vue';
import type { HallEntry } from '~/shared/api/types';

// The top three, rising onto a podium: 2nd · 1st · 3rd.
const props = defineProps<{ leaders: HallEntry[] }>();

const places = computed(() =>
  [props.leaders[1], props.leaders[0], props.leaders[2]]
    .map((entry, slot) => ({ entry, slot }))
    .filter((p): p is { entry: HallEntry; slot: number } => Boolean(p.entry)),
);

const heights: Record<number, string> = { 1: 'h-36 sm:h-44', 2: 'h-28 sm:h-32', 3: 'h-20 sm:h-24' };
const colors: Record<number, string> = {
  1: 'from-amber-300 to-amber-600',
  2: 'from-slate-200 to-slate-500',
  3: 'from-orange-300 to-orange-700',
};
</script>

<template>
  <div class="flex items-end justify-center gap-3 sm:gap-6">
    <div
      v-for="{ entry, slot } in places"
      :key="entry.user.id"
      class="podium-step flex w-24 flex-col items-center sm:w-36"
      :style="{ '--delay': `${[0.3, 0, 0.6][slot]}s` }"
    >
      <div class="relative mb-2">
        <UIcon
          v-if="entry.rank === 1"
          name="i-mdi-crown"
          class="crown absolute -top-7 left-1/2 h-8 w-8 -translate-x-1/2 text-amber-400 drop-shadow"
        />
        <PersonAvatar
          :person="entry.user"
          :size="entry.rank === 1 ? 'h-20 w-20 text-2xl ring-4 ring-amber-400' : 'h-14 w-14 text-lg ring-2 ring-white/40'"
        />
      </div>
      <p class="w-full truncate text-center text-sm font-semibold text-text">{{ personName(entry.user) }}</p>
      <p class="text-xs font-medium text-text-muted">{{ $t('hallOfFame.points', { count: entry.points }) }}</p>
      <div
        class="mt-2 flex w-full items-start justify-center rounded-t-xl bg-gradient-to-b pt-3 text-3xl font-black text-white/90 shadow-lg"
        :class="[heights[entry.rank], colors[entry.rank]]"
      >
        {{ entry.rank }}
      </div>
    </div>
  </div>
</template>

<style scoped>
.podium-step {
  animation: rise 0.9s cubic-bezier(0.34, 1.56, 0.64, 1) both;
  animation-delay: var(--delay);
}
.crown {
  animation: bob 2.4s ease-in-out infinite;
}
@keyframes rise {
  from {
    opacity: 0;
    transform: translateY(60px) scale(0.9);
  }
}
@keyframes bob {
  50% {
    transform: translate(-50%, -4px) rotate(-6deg);
  }
}
@media (prefers-reduced-motion: reduce) {
  .podium-step,
  .crown {
    animation: none;
  }
}
</style>
