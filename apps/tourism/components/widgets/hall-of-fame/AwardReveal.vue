<script setup lang="ts">
import AchievementMedal from '~/components/entities/achievement/AchievementMedal.vue';
import type { Award } from '~/shared/api/types';

// Unveils newly earned achievements one by one, with confetti.
const props = defineProps<{ awards: Award[] }>();
const emit = defineEmits<{ done: [] }>();
const { locale } = useI18n();

const index = ref(0);
const current = computed(() => props.awards[index.value]);

const CONFETTI_COLORS = ['#f59e0b', '#ef4444', '#22c55e', '#3b82f6', '#a855f7', '#ec4899'];
const confetti = Array.from({ length: 48 }, (_, i) => ({
  left: `${(i * 37) % 100}%`,
  color: CONFETTI_COLORS[i % CONFETTI_COLORS.length],
  delay: `${(i % 12) * 0.08}s`,
  duration: `${2.2 + (i % 5) * 0.35}s`,
  drift: `${((i * 53) % 120) - 60}px`,
  spin: `${(i * 97) % 720}deg`,
}));

function next(): void {
  if (index.value < props.awards.length - 1) {
    index.value += 1;
  } else {
    emit('done');
  }
}
</script>

<template>
  <Teleport to="body">
    <div class="reveal fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4 backdrop-blur-sm" @click.self="next">
      <span
        v-for="(piece, i) in confetti"
        :key="`${index}-${i}`"
        class="confetti pointer-events-none absolute top-0 h-3 w-2 rounded-sm"
        :style="{
          left: piece.left,
          background: piece.color,
          animationDelay: piece.delay,
          animationDuration: piece.duration,
          '--drift': piece.drift,
          '--spin': piece.spin,
        }"
      />
      <div v-if="current" :key="current.id" class="reveal__card relative flex max-w-sm flex-col items-center gap-4 rounded-3xl bg-surface p-8 text-center shadow-2xl">
        <span class="rays pointer-events-none absolute left-1/2 top-16 h-72 w-72 -translate-x-1/2 -translate-y-1/2" />
        <p class="relative text-xs font-bold uppercase tracking-[0.3em] text-brand-600 dark:text-brand-400">
          {{ $t('hallOfFame.reveal.heading') }}
        </p>
        <div class="reveal__medal relative">
          <AchievementMedal :icon="current.achievement.icon" :rarity="current.achievement.rarity" size="lg" fresh />
        </div>
        <h3 class="relative text-2xl font-extrabold text-text">{{ localizedText(current.achievement.title, locale) }}</h3>
        <p class="relative text-xs font-semibold uppercase tracking-widest" :class="`rarity-text--${current.achievement.rarity}`">
          {{ $t(`hallOfFame.rarity.${current.achievement.rarity}`) }} · +{{ current.achievement.points }}
        </p>
        <p v-if="localizedText(current.achievement.description, locale)" class="relative text-sm text-text-muted">
          {{ localizedText(current.achievement.description, locale) }}
        </p>
        <p v-if="current.note" class="relative rounded-xl bg-bg px-4 py-2 text-sm italic text-text">“{{ current.note }}”</p>
        <p v-if="current.awardedBy" class="relative text-xs text-text-muted">
          {{ $t('hallOfFame.awardedBy', { name: personName(current.awardedBy) }) }}
        </p>
        <UButton color="primary" size="lg" class="relative mt-2" @click="next">
          {{ index < awards.length - 1 ? $t('hallOfFame.reveal.next') : $t('hallOfFame.reveal.done') }}
        </UButton>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.reveal {
  animation: fade 0.3s ease-out;
}
.reveal__card {
  animation: pop 0.6s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.reveal__medal {
  animation: drop 0.9s cubic-bezier(0.34, 1.56, 0.64, 1) 0.15s both;
}
.rays {
  background: repeating-conic-gradient(from 0deg, rgb(var(--color-brand-500) / 0.18) 0deg 10deg, transparent 10deg 20deg);
  mask-image: radial-gradient(circle, #000 20%, transparent 70%);
  animation: spin 14s linear infinite;
}
.confetti {
  animation-name: fall;
  animation-timing-function: cubic-bezier(0.25, 0.6, 0.5, 1);
  animation-fill-mode: both;
}
.rarity-text--common {
  color: #b7793f;
}
.rarity-text--rare {
  color: #0ea5e9;
}
.rarity-text--epic {
  color: #a855f7;
}
.rarity-text--legendary {
  color: #f59e0b;
}
@keyframes fade {
  from {
    opacity: 0;
  }
}
@keyframes pop {
  from {
    opacity: 0;
    transform: scale(0.6) translateY(40px);
  }
}
@keyframes drop {
  from {
    opacity: 0;
    transform: translateY(-120px) rotate(-200deg) scale(0.3);
  }
}
@keyframes spin {
  to {
    transform: translate(-50%, -50%) rotate(360deg);
  }
}
@keyframes fall {
  from {
    transform: translate(0, -20px) rotate(0);
    opacity: 1;
  }
  to {
    transform: translate(var(--drift), 100vh) rotate(var(--spin));
    opacity: 0.2;
  }
}
@media (prefers-reduced-motion: reduce) {
  .reveal,
  .reveal__card,
  .reveal__medal,
  .rays,
  .confetti {
    animation: none;
  }
  .confetti {
    display: none;
  }
}
</style>
