<script setup lang="ts">
import { AchievementRarity } from '~/shared/api/types';

// A round medal for an achievement: the rarer, the brighter it glows.
// `locked` draws an unearned one as a silhouette; `fresh` adds sparkles.
const props = withDefaults(
  defineProps<{
    icon: string;
    rarity: AchievementRarity;
    size?: 'sm' | 'md' | 'lg';
    locked?: boolean;
    fresh?: boolean;
  }>(),
  { size: 'md', locked: false, fresh: false },
);

const sizeClass = computed(
  () => ({ sm: 'h-10 w-10', md: 'h-16 w-16', lg: 'h-28 w-28' })[props.size],
);
const iconClass = computed(() => ({ sm: 'h-5 w-5', md: 'h-8 w-8', lg: 'h-14 w-14' })[props.size]);
</script>

<template>
  <span
    class="medal relative inline-flex shrink-0 items-center justify-center rounded-full"
    :class="[sizeClass, locked ? 'medal--locked' : `medal--${rarity}`, { 'medal--fresh': fresh && !locked }]"
  >
    <span class="medal__ring absolute inset-0 rounded-full" />
    <span class="medal__face absolute inset-[12%] rounded-full" />
    <span v-if="!locked && (rarity === AchievementRarity.EPIC || rarity === AchievementRarity.LEGENDARY)" class="medal__shine absolute inset-0 overflow-hidden rounded-full" />
    <UIcon :name="locked ? 'i-heroicons-lock-closed' : icon" class="relative" :class="[iconClass, locked ? 'text-text-muted' : 'text-white drop-shadow']" />
    <template v-if="fresh && !locked">
      <span v-for="n in 4" :key="n" class="medal__spark absolute" :style="{ '--i': n }" />
    </template>
  </span>
</template>

<style scoped>
.medal {
  --c1: #94a3b8;
  --c2: #475569;
  --glow: transparent;
  box-shadow: 0 0 0 0 var(--glow);
  transition: transform 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.medal:hover {
  transform: translateY(-3px) rotate(-4deg) scale(1.06);
}
.medal__ring {
  background: conic-gradient(from 0deg, var(--c1), var(--c2), var(--c1), var(--c2), var(--c1));
}
.medal__face {
  background: radial-gradient(circle at 30% 25%, var(--c1), var(--c2) 70%);
  box-shadow: inset 0 2px 6px rgb(255 255 255 / 0.35), inset 0 -4px 8px rgb(0 0 0 / 0.25);
}
.medal--common {
  --c1: #d6a26b;
  --c2: #8a5a2b;
}
.medal--rare {
  --c1: #7dd3fc;
  --c2: #0369a1;
  --glow: rgb(56 189 248 / 0.35);
  animation: glow 3s ease-in-out infinite;
}
.medal--epic {
  --c1: #d8b4fe;
  --c2: #6d28d9;
  --glow: rgb(168 85 247 / 0.45);
  animation: glow 2.4s ease-in-out infinite;
}
.medal--legendary {
  --c1: #fde68a;
  --c2: #d97706;
  --glow: rgb(251 191 36 / 0.6);
  animation: glow 1.8s ease-in-out infinite;
}
.medal--legendary .medal__ring {
  animation: spin 6s linear infinite;
}
.medal--locked {
  --c1: rgb(148 163 184 / 0.25);
  --c2: rgb(71 85 105 / 0.25);
  filter: grayscale(1);
}
.medal--locked .medal__face {
  box-shadow: inset 0 0 0 1px rgb(148 163 184 / 0.3);
}
.medal__shine::after {
  content: '';
  position: absolute;
  inset: -50%;
  background: linear-gradient(115deg, transparent 40%, rgb(255 255 255 / 0.55) 50%, transparent 60%);
  animation: shine 3.2s ease-in-out infinite;
}
.medal__spark {
  width: 6px;
  height: 6px;
  border-radius: 9999px;
  background: #fff;
  box-shadow: 0 0 8px 2px var(--c1);
  animation: spark 1.6s ease-out infinite;
  animation-delay: calc(var(--i) * 0.4s);
  top: 50%;
  left: 50%;
}
@keyframes glow {
  50% {
    box-shadow: 0 0 22px 6px var(--glow);
  }
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
@keyframes shine {
  0%,
  60% {
    transform: translateX(-60%);
  }
  100% {
    transform: translateX(60%);
  }
}
@keyframes spark {
  0% {
    transform: translate(-50%, -50%) rotate(calc(var(--i) * 90deg)) translateY(0) scale(1);
    opacity: 1;
  }
  100% {
    transform: translate(-50%, -50%) rotate(calc(var(--i) * 90deg)) translateY(-160%) scale(0);
    opacity: 0;
  }
}
@media (prefers-reduced-motion: reduce) {
  .medal,
  .medal__ring,
  .medal__shine::after,
  .medal__spark {
    animation: none !important;
  }
}
</style>
