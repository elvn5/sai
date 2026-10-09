<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import AchievementMedal from '~/components/entities/achievement/AchievementMedal.vue';
import PersonAvatar from '~/components/entities/achievement/PersonAvatar.vue';
import { ApiError, useApiClient } from '~/shared/api/client';
import type { Achievement, Award, AwardRecipient } from '~/shared/api/types';

// Partners hand achievements to travelers here, by the code (or QR link)
// the traveler shows them.
const { t, locale } = useI18n();
const route = useRoute();
const api = useApiClient();
const auth = useAuth();
const queryClient = useQueryClient();

useRequirePartner();

const codeInput = ref(typeof route.query.code === 'string' ? route.query.code : '');
const recipient = ref<AwardRecipient | null>(null);
const lookupError = ref<string | null>(null);
const selectedId = ref<string | null>(null);
const note = ref('');
const awardError = ref<string | null>(null);
const justAwarded = ref<Award | null>(null);

const catalogQuery = useQuery({
  queryKey: ['achievements'],
  queryFn: () => api.get<Achievement[]>('/achievements'),
});
const givenQuery = useQuery({
  queryKey: ['awards-given'],
  queryFn: () => api.get<Award[]>('/achievements/awards/given'),
  enabled: auth.isPartner,
});

const catalog = computed(() => catalogQuery.data.value ?? []);
const earned = computed(() => new Set(recipient.value?.earnedAchievementIds ?? []));
const selected = computed(
  () => catalog.value.find((a: Achievement) => a.id === selectedId.value) ?? null,
);
const normalizedCode = computed(() => codeInput.value.replace(/[\s-]/g, '').toUpperCase());

const lookupMutation = useMutation({
  mutationFn: (code: string) =>
    api.get<AwardRecipient>(`/achievements/recipients/${encodeURIComponent(code)}`),
  onSuccess: (res) => {
    recipient.value = res;
    selectedId.value = null;
    justAwarded.value = null;
  },
  onError: (error) => {
    recipient.value = null;
    lookupError.value =
      error instanceof ApiError && error.status === 404
        ? t('award.unknownCode')
        : t('award.lookupError');
  },
});

const awardMutation = useMutation({
  mutationFn: () =>
    api.post<Award>('/achievements/awards', {
      code: normalizedCode.value,
      achievementId: selectedId.value,
      note: note.value.trim() || null,
    }),
  onSuccess: (award) => {
    justAwarded.value = award;
    recipient.value?.earnedAchievementIds.push(award.achievement.id);
    selectedId.value = null;
    note.value = '';
    queryClient.invalidateQueries({ queryKey: ['awards-given'] });
    queryClient.invalidateQueries({ queryKey: ['hall-of-fame'] });
  },
  onError: (error) => {
    const status = error instanceof ApiError ? error.status : 0;
    awardError.value =
      status === 409
        ? t('award.duplicate')
        : status === 400
          ? t('award.self')
          : t('award.awardError');
  },
});

function lookup(): void {
  lookupError.value = null;
  if (normalizedCode.value.length !== 8) {
    lookupError.value = t('award.codeFormat');
    return;
  }
  lookupMutation.mutate(normalizedCode.value);
}

function giveAward(): void {
  awardError.value = null;
  awardMutation.mutate();
}

onMounted(() => {
  if (codeInput.value) lookup();
});

useSeoMeta({ title: () => t('award.title'), robots: 'noindex' });
</script>

<template>
  <section class="mx-auto max-w-4xl px-4 py-12">
    <div class="flex items-center gap-3">
      <UIcon name="i-mdi-medal" class="h-9 w-9 text-amber-500" />
      <h1 class="text-3xl font-extrabold text-text">{{ $t('award.title') }}</h1>
    </div>
    <p class="mt-2 text-text-muted">{{ $t('award.subtitle') }}</p>

    <form class="mt-8 flex flex-col gap-3 sm:flex-row" @submit.prevent="lookup">
      <UInput
        v-model="codeInput"
        size="xl"
        class="flex-1"
        :placeholder="$t('award.codePlaceholder')"
        input-class="font-mono uppercase tracking-[0.2em]"
        icon="i-mdi-qrcode-scan"
        autocomplete="off"
      />
      <UButton type="submit" size="xl" color="primary" :loading="lookupMutation.isPending.value">
        {{ $t('award.find') }}
      </UButton>
    </form>
    <p v-if="lookupError" class="mt-2 text-sm text-brand-600">{{ lookupError }}</p>

    <div v-if="recipient" class="recipient mt-8 flex flex-col gap-6">
      <div class="flex items-center gap-4 rounded-card border border-border bg-surface p-4">
        <PersonAvatar :person="recipient.user" size="h-14 w-14 text-xl" />
        <div>
          <p class="text-lg font-bold text-text">{{ personName(recipient.user) }}</p>
          <p class="text-sm text-text-muted">{{ $t('award.alreadyHas', { count: recipient.earnedAchievementIds.length }) }}</p>
        </div>
      </div>

      <div
        v-if="justAwarded"
        class="success flex items-center gap-4 rounded-card border border-amber-400/50 bg-amber-400/10 p-4"
      >
        <AchievementMedal :icon="justAwarded.achievement.icon" :rarity="justAwarded.achievement.rarity" fresh />
        <p class="font-semibold text-text">
          {{ $t('award.success', { name: personName(justAwarded.user), title: localizedText(justAwarded.achievement.title, locale) }) }}
        </p>
      </div>

      <div>
        <h2 class="mb-3 font-semibold text-text">{{ $t('award.choose') }}</h2>
        <div class="grid grid-cols-2 gap-3 sm:grid-cols-3">
          <button
            v-for="achievement in catalog"
            :key="achievement.id"
            type="button"
            :disabled="earned.has(achievement.id)"
            class="flex flex-col items-center gap-2 rounded-card border bg-surface p-4 text-center transition disabled:cursor-not-allowed disabled:opacity-40"
            :class="selectedId === achievement.id ? 'border-brand-500 ring-2 ring-brand-500' : 'border-border hover:border-brand-400'"
            @click="selectedId = achievement.id"
          >
            <AchievementMedal :icon="achievement.icon" :rarity="achievement.rarity" />
            <span class="text-sm font-semibold text-text">{{ localizedText(achievement.title, locale) }}</span>
            <span class="text-xs text-text-muted">
              {{ earned.has(achievement.id) ? $t('award.earned') : `${$t(`hallOfFame.rarity.${achievement.rarity}`)} · +${achievement.points}` }}
            </span>
          </button>
        </div>
        <p v-if="catalog.length === 0" class="text-sm text-text-muted">{{ $t('award.emptyCatalog') }}</p>
      </div>

      <div v-if="selected" class="flex flex-col gap-3">
        <UFormGroup :label="$t('award.note')" :hint="$t('award.noteHint')">
          <UTextarea v-model="note" :maxlength="300" :rows="2" />
        </UFormGroup>
        <p v-if="awardError" class="text-sm text-brand-600">{{ awardError }}</p>
        <UButton size="xl" color="primary" block icon="i-mdi-medal" :loading="awardMutation.isPending.value" @click="giveAward">
          {{ $t('award.give', { title: localizedText(selected.title, locale) }) }}
        </UButton>
      </div>
    </div>

    <div v-if="(givenQuery.data.value ?? []).length > 0" class="mt-12">
      <h2 class="mb-3 font-semibold text-text">{{ $t('award.history') }}</h2>
      <ul class="flex flex-col gap-2">
        <li
          v-for="award in givenQuery.data.value"
          :key="award.id"
          class="flex items-center gap-3 rounded-card border border-border bg-surface px-4 py-2"
        >
          <AchievementMedal :icon="award.achievement.icon" :rarity="award.achievement.rarity" size="sm" />
          <span class="flex-1 text-sm text-text">
            <span class="font-semibold">{{ personName(award.user) }}</span> · {{ localizedText(award.achievement.title, locale) }}
          </span>
          <span class="text-xs text-text-muted">{{ new Date(award.awardedAt).toLocaleDateString(locale) }}</span>
        </li>
      </ul>
    </div>
  </section>
</template>

<style scoped>
.recipient,
.success {
  animation: enter 0.5s cubic-bezier(0.22, 1, 0.36, 1) both;
}
@keyframes enter {
  from {
    opacity: 0;
    transform: translateY(12px);
  }
}
@media (prefers-reduced-motion: reduce) {
  .recipient,
  .success {
    animation: none;
  }
}
</style>
