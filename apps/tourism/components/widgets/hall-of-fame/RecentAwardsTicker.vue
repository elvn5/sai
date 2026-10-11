<script setup lang="ts">
import AchievementMedal from '~/components/entities/achievement/AchievementMedal.vue';
import type { Award } from '~/shared/api/types';

// The latest awards scrolling past, like a stadium ticker.
defineProps<{ awards: Award[] }>();
const { locale } = useI18n();
</script>

<template>
  <div class="ticker relative overflow-hidden rounded-card border border-border bg-surface py-3">
    <div class="ticker__track flex w-max gap-8">
      <template v-for="copy in 2" :key="copy">
        <div
          v-for="award in awards"
          :key="`${copy}-${award.id}`"
          class="flex items-center gap-3 whitespace-nowrap text-sm"
          :aria-hidden="copy === 2"
        >
          <AchievementMedal :icon="award.achievement.icon" :rarity="award.achievement.rarity" size="sm" />
          <span class="text-text">
            <span class="font-semibold">{{ personName(award.user) }}</span>
            <span class="text-text-muted"> · </span>
            {{ localizedText(award.achievement.title, locale) }}
          </span>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.ticker {
  mask-image: linear-gradient(90deg, transparent, #000 8%, #000 92%, transparent);
}
.ticker__track {
  animation: scroll 40s linear infinite;
}
.ticker:hover .ticker__track {
  animation-play-state: paused;
}
@keyframes scroll {
  to {
    transform: translateX(calc(-50% - 1rem));
  }
}
@media (prefers-reduced-motion: reduce) {
  .ticker__track {
    animation: none;
  }
}
</style>
