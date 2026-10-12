<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import PersonAvatar from '~/components/entities/achievement/PersonAvatar.vue';
import TelegramNotice from '~/components/features/telegram-notice/TelegramNotice.vue';
import { ApiError, useApiClient } from '~/shared/api/client';
import type { Outing } from '~/shared/api/types';
import { formatAmount } from '~/shared/lib/format-price';
import { LEVELS, TRANSPORTS, optionIcon } from '~/shared/lib/trail-options';

const route = useRoute();
const { t, locale } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const auth = useAuth();
const queryClient = useQueryClient();
const id = String(route.params.id);

const outingQuery = useQuery({
  queryKey: ['outing', id],
  queryFn: async () => {
    try {
      return await api.get<Outing>(`/outings/${id}`);
    } catch {
      return null;
    }
  },
});
await outingQuery.suspense();
if (!outingQuery.data.value) {
  throw createError({ statusCode: 404, statusMessage: 'Outing not found' });
}

const outing = computed(() => outingQuery.data.value as Outing);
const left = computed(() => Math.max(0, outing.value.seats - outing.value.taken));
const started = computed(() => new Date(outing.value.startsAt) <= new Date());
const closed = computed(() => outing.value.status === 'cancelled' || started.value);
const actionError = ref<string | null>(null);

function fmt(iso: string): string {
  return new Date(iso).toLocaleString(locale.value, {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    hour: '2-digit',
    minute: '2-digit',
    timeZone: 'Asia/Bishkek',
  });
}

function onDone(updated: Outing): void {
  queryClient.setQueryData(['outing', id], updated);
  queryClient.invalidateQueries({ queryKey: ['outings'] });
}
function onError(error: Error): void {
  actionError.value =
    error instanceof ApiError && error.status === 409
      ? t('outings.errors.conflict')
      : t('outings.errors.generic');
  void outingQuery.refetch();
}

const join = useMutation({
  mutationFn: () => api.post<Outing>(`/outings/${id}/join`),
  onSuccess: onDone,
  onError,
});
const leave = useMutation({
  mutationFn: () => api.delete<Outing>(`/outings/${id}/join`),
  onSuccess: onDone,
  onError,
});
const cancel = useMutation({
  mutationFn: () => api.post<Outing>(`/outings/${id}/cancel`),
  onSuccess: onDone,
  onError,
});

function onJoin(): void {
  actionError.value = null;
  if (!auth.user.value) {
    void navigateTo(localePath('/login'));
    return;
  }
  join.mutate();
}
function onCancel(): void {
  if (window.confirm(t('outings.cancelConfirm'))) cancel.mutate();
}

// Re-read once the session is known: contact and "joined" depend on it.
onMounted(() => {
  void outingQuery.refetch();
});

useSeoMeta({
  title: () => outing.value.title,
  description: () => `${fmt(outing.value.startsAt)} · ${outing.value.meetingPoint}`,
  robots: 'noindex',
});
</script>

<template>
  <section class="mx-auto max-w-3xl px-4 py-12">
    <NuxtLink :to="localePath('/outings')" class="text-sm text-text-muted hover:text-text">← {{ $t('outings.title') }}</NuxtLink>

    <div class="mt-4 overflow-hidden rounded-card border border-border bg-surface">
      <div class="bg-gradient-to-br from-brand-600 to-secondary-700 px-6 py-6 text-white">
        <span v-if="outing.status === 'cancelled'" class="mb-2 inline-block rounded-full bg-black/30 px-2.5 py-1 text-xs font-semibold">
          {{ $t('outings.cancelled') }}
        </span>
        <h1 class="text-2xl font-black sm:text-3xl">{{ outing.title }}</h1>
        <p class="mt-2 flex items-center gap-2 text-white/90">
          <UIcon name="i-mdi-calendar-clock" class="h-5 w-5" />{{ fmt(outing.startsAt) }}
        </p>
        <p v-if="outing.returnsAt" class="mt-1 flex items-center gap-2 text-sm text-white/75">
          <UIcon name="i-mdi-home-import-outline" class="h-5 w-5" />{{ $t('outings.returns', { when: fmt(outing.returnsAt) }) }}
        </p>
      </div>

      <div class="grid gap-4 p-6 sm:grid-cols-2">
        <div class="flex items-start gap-3">
          <UIcon name="i-mdi-map-marker-radius" class="mt-0.5 h-5 w-5 shrink-0 text-brand-600 dark:text-brand-400" />
          <div>
            <p class="text-xs text-text-muted">{{ $t('outings.form.meetingPoint') }}</p>
            <p class="font-medium text-text">{{ outing.meetingPoint }}</p>
          </div>
        </div>
        <NuxtLink v-if="outing.trail" :to="localePath(`/trails/${outing.trail.slug}`)" class="flex items-start gap-3 hover:underline">
          <UIcon name="i-mdi-map-marker-path" class="mt-0.5 h-5 w-5 shrink-0 text-brand-600 dark:text-brand-400" />
          <div>
            <p class="text-xs text-text-muted">{{ $t('outings.form.trail') }}</p>
            <p class="font-medium text-text">{{ outing.trail.name }}</p>
          </div>
        </NuxtLink>
        <div class="flex items-start gap-3">
          <UIcon :name="optionIcon(TRANSPORTS, outing.transport)" class="mt-0.5 h-5 w-5 shrink-0 text-brand-600 dark:text-brand-400" />
          <div>
            <p class="text-xs text-text-muted">{{ $t('outings.form.transport') }}</p>
            <p class="font-medium text-text">{{ $t(`outings.transport.${outing.transport}`) }}</p>
          </div>
        </div>
        <div class="flex items-start gap-3">
          <UIcon :name="optionIcon(LEVELS, outing.level)" class="mt-0.5 h-5 w-5 shrink-0 text-brand-600 dark:text-brand-400" />
          <div>
            <p class="text-xs text-text-muted">{{ $t('outings.form.level') }}</p>
            <p class="font-medium text-text">{{ $t(`outings.level.${outing.level}`) }}</p>
          </div>
        </div>
        <div v-if="outing.costPerPerson !== null" class="flex items-start gap-3">
          <UIcon name="i-mdi-cash-multiple" class="mt-0.5 h-5 w-5 shrink-0 text-brand-600 dark:text-brand-400" />
          <div>
            <p class="text-xs text-text-muted">{{ $t('outings.form.cost') }}</p>
            <p class="font-medium text-text">
              {{ outing.costPerPerson === 0 ? $t('outings.free') : `${formatAmount(locale, outing.costPerPerson)} ${$t('listing.currency')}` }}
            </p>
          </div>
        </div>
      </div>

      <p v-if="outing.description" class="whitespace-pre-line border-t border-border px-6 py-5 text-text">{{ outing.description }}</p>

      <div class="border-t border-border px-6 py-5">
        <div class="mb-2 flex items-center justify-between text-sm">
          <span class="font-semibold text-text">{{ $t('outings.participants') }}</span>
          <span class="text-text-muted">{{ $t('outings.seatsTaken', { taken: outing.taken, seats: outing.seats }) }}</span>
        </div>
        <div class="mb-4 h-2 overflow-hidden rounded-full bg-bg">
          <div class="h-full rounded-full bg-brand-500 transition-all duration-500" :style="{ width: `${Math.min(100, (outing.taken / outing.seats) * 100)}%` }" />
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <div class="flex items-center gap-2 rounded-full bg-bg py-1 pl-1 pr-3">
            <PersonAvatar :person="outing.organizer" size="h-7 w-7 text-xs" />
            <span class="text-sm text-text">{{ personName(outing.organizer) }}</span>
            <span class="text-xs text-brand-600 dark:text-brand-400">{{ $t('outings.organizer') }}</span>
          </div>
          <div v-for="person in outing.participants ?? []" :key="person.id" class="flex items-center gap-2 rounded-full bg-bg py-1 pl-1 pr-3">
            <PersonAvatar :person="person" size="h-7 w-7 text-xs" />
            <span class="text-sm text-text">{{ personName(person) }}</span>
          </div>
        </div>
      </div>

      <div v-if="outing.contact" class="flex items-center gap-3 border-t border-border bg-emerald-500/10 px-6 py-4">
        <UIcon name="i-mdi-message-text-outline" class="h-5 w-5 text-emerald-600" />
        <div>
          <p class="text-xs text-text-muted">{{ $t('outings.contact') }}</p>
          <p class="font-semibold text-text">{{ outing.contact }}</p>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-3 border-t border-border px-6 py-5">
        <template v-if="outing.isOrganizer">
          <UButton v-if="!closed" color="gray" icon="i-heroicons-pencil-square" :to="localePath(`/outings/${outing.id}/edit`)">{{ $t('listing.edit') }}</UButton>
          <UButton v-if="!closed" color="red" variant="soft" :loading="cancel.isPending.value" @click="onCancel">{{ $t('outings.cancel') }}</UButton>
        </template>
        <template v-else-if="!closed">
          <UButton v-if="outing.isJoined" color="gray" :loading="leave.isPending.value" @click="leave.mutate()">{{ $t('outings.leave') }}</UButton>
          <UButton v-else color="primary" size="lg" icon="i-mdi-hand-wave" :disabled="left === 0" :loading="join.isPending.value" @click="onJoin">
            {{ left === 0 ? $t('outings.full') : $t('outings.join') }}
          </UButton>
          <span v-if="!outing.isJoined && left > 0" class="text-sm text-text-muted">{{ $t('outings.joinHint') }}</span>
        </template>
        <UButton
          v-if="started && outing.trail && (outing.isJoined || outing.isOrganizer)"
          color="primary"
          variant="soft"
          icon="i-mdi-clipboard-text-outline"
          :to="localePath(`/trails/${outing.trail.slug}`)"
        >
          {{ $t('outings.reportAfter') }}
        </UButton>
        <p v-if="actionError" class="w-full text-sm text-brand-600">{{ actionError }}</p>
      </div>

      <div v-if="!closed && (outing.isJoined || outing.isOrganizer)" class="border-t border-border px-6 py-4">
        <TelegramNotice />
      </div>
    </div>
  </section>
</template>
