<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import AchievementMedal from '~/components/entities/achievement/AchievementMedal.vue';
import PersonAvatar from '~/components/entities/achievement/PersonAvatar.vue';
import AwardCodeCard from '~/components/features/award-code/AwardCodeCard.vue';
import AwardReveal from '~/components/widgets/hall-of-fame/AwardReveal.vue';
import HallPodium from '~/components/widgets/hall-of-fame/HallPodium.vue';
import RecentAwardsTicker from '~/components/widgets/hall-of-fame/RecentAwardsTicker.vue';
import { useApiClient } from '~/shared/api/client';
import type { Achievement, Award, HallEntry, HallOfFame, MyAchievements } from '~/shared/api/types';

const { t, locale } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const auth = useAuth();
const queryClient = useQueryClient();
const { isTma, getInitData } = useTelegramWebApp();

const hallQuery = useQuery({
  queryKey: ['hall-of-fame'],
  queryFn: () => api.get<HallOfFame>('/hall-of-fame'),
});
const catalogQuery = useQuery({
  queryKey: ['achievements'],
  queryFn: () => api.get<Achievement[]>('/achievements'),
});
await Promise.all([hallQuery.suspense(), catalogQuery.suspense()]);

const mineQuery = useQuery({
  queryKey: ['my-achievements'],
  queryFn: () => api.get<MyAchievements>('/achievements/me'),
  enabled: computed(() => Boolean(auth.user.value)),
});

const leaders = computed(() => hallQuery.data.value?.leaders ?? []);
const recent = computed(() => hallQuery.data.value?.recent ?? []);
const mine = computed(() => mineQuery.data.value ?? null);
const myRank = computed(
  () => leaders.value.find((l: HallEntry) => l.user.id === auth.user.value?.id)?.rank ?? null,
);
const earnedIds = computed(
  () => new Set(mine.value?.awards.map((a: Award) => a.achievement.id) ?? []),
);
const locked = computed(() =>
  (catalogQuery.data.value ?? []).filter((a: Achievement) => !earnedIds.value.has(a.id)),
);
const freshAwards = computed(() => mine.value?.awards.filter((a: Award) => a.isNew) ?? []);
const revealDismissed = ref(false);

const seenMutation = useMutation({
  mutationFn: () => api.post('/achievements/me/seen'),
  onSuccess: () => queryClient.invalidateQueries({ queryKey: ['my-achievements'] }),
});

function finishReveal(): void {
  revealDismissed.value = true;
  seenMutation.mutate();
}

useSeoMeta({
  title: () => t('hallOfFame.title'),
  description: () => t('hallOfFame.subtitle'),
  ogTitle: () => t('hallOfFame.title'),
  ogDescription: () => t('hallOfFame.subtitle'),
});

onMounted(() => {
  if (!auth.checked.value) {
    void auth.ensureSession({ isTma: isTma.value, initData: getInitData() });
  }
});
</script>

<template>
  <div>
    <section class="hero relative isolate overflow-hidden px-4 pb-16 pt-20 text-center">
      <div class="hero__bg absolute inset-0 -z-10" />
      <span v-for="n in 14" :key="n" class="particle absolute -z-10 rounded-full" :style="{ '--n': n }" />
      <div class="trophy relative mx-auto mb-6 flex h-28 w-28 items-center justify-center">
        <span class="trophy__rays absolute inset-[-60%]" />
        <UIcon name="i-mdi-trophy" class="relative h-24 w-24 text-amber-300 drop-shadow-[0_0_25px_rgba(251,191,36,0.6)]" />
      </div>
      <h1 class="text-4xl font-black tracking-tight text-white sm:text-5xl">{{ $t('hallOfFame.title') }}</h1>
      <p class="mx-auto mt-3 max-w-xl text-white/80">{{ $t('hallOfFame.subtitle') }}</p>
    </section>

    <div class="mx-auto -mt-6 max-w-5xl px-4">
      <RecentAwardsTicker v-if="recent.length > 0" :awards="recent" />
    </div>

    <section v-if="auth.user.value && mine" class="mx-auto max-w-5xl px-4 pt-12">
      <div class="flex flex-wrap items-end justify-between gap-4">
        <h2 class="text-2xl font-extrabold text-text">{{ $t('hallOfFame.mine.heading') }}</h2>
        <div class="flex gap-3">
          <div class="stat rounded-card border border-border bg-surface px-4 py-2 text-center">
            <p class="text-2xl font-black text-amber-500">{{ mine.points }}</p>
            <p class="text-xs text-text-muted">{{ $t('hallOfFame.mine.points') }}</p>
          </div>
          <div class="stat rounded-card border border-border bg-surface px-4 py-2 text-center">
            <p class="text-2xl font-black text-text">{{ mine.awards.length }}</p>
            <p class="text-xs text-text-muted">{{ $t('hallOfFame.mine.earned') }}</p>
          </div>
          <div v-if="myRank" class="stat rounded-card border border-border bg-surface px-4 py-2 text-center">
            <p class="text-2xl font-black text-brand-600 dark:text-brand-400">#{{ myRank }}</p>
            <p class="text-xs text-text-muted">{{ $t('hallOfFame.mine.rank') }}</p>
          </div>
        </div>
      </div>

      <div class="mt-6 grid gap-6 lg:grid-cols-[1fr_340px]">
        <div class="shelf rounded-card border border-border bg-surface p-5">
          <div v-if="mine.awards.length > 0 || locked.length > 0" class="grid grid-cols-2 gap-5 sm:grid-cols-3 md:grid-cols-4">
            <div
              v-for="(award, i) in mine.awards"
              :key="award.id"
              class="shelf__item flex flex-col items-center gap-2 text-center"
              :style="{ '--i': i }"
            >
              <AchievementMedal :icon="award.achievement.icon" :rarity="award.achievement.rarity" :fresh="award.isNew" />
              <p class="text-sm font-semibold leading-tight text-text">{{ localizedText(award.achievement.title, locale) }}</p>
              <p v-if="award.awardedBy" class="text-xs text-text-muted">
                {{ $t('hallOfFame.awardedBy', { name: personName(award.awardedBy) }) }}
              </p>
            </div>
            <div
              v-for="(achievement, i) in locked"
              :key="achievement.id"
              class="shelf__item flex flex-col items-center gap-2 text-center opacity-70"
              :style="{ '--i': mine.awards.length + i }"
              :title="localizedText(achievement.description, locale)"
            >
              <AchievementMedal :icon="achievement.icon" :rarity="achievement.rarity" locked />
              <p class="text-sm font-medium leading-tight text-text-muted">{{ localizedText(achievement.title, locale) }}</p>
              <p class="text-xs text-text-muted">+{{ achievement.points }}</p>
            </div>
          </div>
          <p v-else class="py-8 text-center text-text-muted">{{ $t('hallOfFame.mine.empty') }}</p>
        </div>

        <div class="flex flex-col gap-3">
          <AwardCodeCard :code="mine.code" />
          <p class="text-sm text-text-muted">{{ $t('hallOfFame.mine.howTo') }}</p>
        </div>
      </div>
    </section>

    <section v-else-if="auth.checked.value && !auth.user.value" class="mx-auto max-w-5xl px-4 pt-12">
      <div class="flex flex-col items-center gap-4 rounded-card border border-dashed border-border bg-surface px-6 py-10 text-center">
        <div class="flex -space-x-3">
          <AchievementMedal
            v-for="achievement in (catalogQuery.data.value ?? []).slice(0, 5)"
            :key="achievement.id"
            :icon="achievement.icon"
            :rarity="achievement.rarity"
          />
        </div>
        <p class="max-w-md text-text">{{ $t('hallOfFame.guest.pitch') }}</p>
        <UButton color="primary" :to="localePath('/login')">{{ $t('hallOfFame.guest.cta') }}</UButton>
      </div>
    </section>

    <section class="mx-auto max-w-5xl px-4 py-14">
      <h2 class="mb-10 text-center text-2xl font-extrabold text-text">{{ $t('hallOfFame.leaders.heading') }}</h2>
      <template v-if="leaders.length > 0">
        <HallPodium :leaders="leaders" />
        <ol class="mt-10 flex flex-col gap-2">
          <li
            v-for="(entry, i) in leaders.slice(3)"
            :key="entry.user.id"
            class="leader-row flex items-center gap-4 rounded-card border border-border bg-surface px-4 py-3"
            :class="{ 'ring-2 ring-brand-500': entry.user.id === auth.user.value?.id }"
            :style="{ '--i': i }"
          >
            <span class="w-8 text-center text-lg font-black text-text-muted">{{ entry.rank }}</span>
            <PersonAvatar :person="entry.user" />
            <div class="min-w-0 flex-1">
              <p class="truncate font-semibold text-text">{{ personName(entry.user) }}</p>
              <div class="mt-1 flex gap-1">
                <AchievementMedal
                  v-for="achievement in entry.achievements.slice(0, 6)"
                  :key="achievement.id"
                  :icon="achievement.icon"
                  :rarity="achievement.rarity"
                  size="sm"
                />
              </div>
            </div>
            <span class="text-right text-sm font-bold text-amber-500">{{ $t('hallOfFame.points', { count: entry.points }) }}</span>
          </li>
        </ol>
      </template>
      <p v-else class="text-center text-text-muted">{{ $t('hallOfFame.leaders.empty') }}</p>
    </section>

    <AwardReveal v-if="freshAwards.length > 0 && !revealDismissed" :awards="freshAwards" @done="finishReveal" />
  </div>
</template>

<style scoped>
.hero__bg {
  background:
    radial-gradient(ellipse at 50% 0%, rgb(251 191 36 / 0.25), transparent 60%),
    linear-gradient(160deg, rgb(var(--color-brand-900)), #0b1020 70%);
}
.particle {
  width: calc(3px + (var(--n) * 7 % 5) * 1px);
  height: calc(3px + (var(--n) * 7 % 5) * 1px);
  left: calc(var(--n) * 7%);
  bottom: -10px;
  background: rgb(253 230 138 / 0.8);
  box-shadow: 0 0 10px 2px rgb(251 191 36 / 0.5);
  animation: float calc(6s + var(--n) * 0.5s) linear infinite;
  animation-delay: calc(var(--n) * -0.8s);
}
.trophy {
  animation: hover 3.5s ease-in-out infinite;
}
.trophy__rays {
  background: repeating-conic-gradient(from 0deg, rgb(251 191 36 / 0.22) 0deg 8deg, transparent 8deg 18deg);
  mask-image: radial-gradient(circle, #000 15%, transparent 65%);
  animation: rotate 20s linear infinite;
}
.shelf__item,
.leader-row {
  animation: enter 0.6s cubic-bezier(0.22, 1, 0.36, 1) both;
  animation-delay: calc(var(--i) * 60ms);
}
@keyframes float {
  from {
    transform: translateY(0);
    opacity: 0;
  }
  15% {
    opacity: 1;
  }
  to {
    transform: translateY(-520px);
    opacity: 0;
  }
}
@keyframes hover {
  50% {
    transform: translateY(-10px);
  }
}
@keyframes rotate {
  to {
    transform: rotate(360deg);
  }
}
@keyframes enter {
  from {
    opacity: 0;
    transform: translateY(16px);
  }
}
@media (prefers-reduced-motion: reduce) {
  .particle,
  .trophy,
  .trophy__rays,
  .shelf__item,
  .leader-row {
    animation: none;
  }
}
</style>
