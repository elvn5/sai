<script setup lang="ts">
import PersonAvatar from '~/components/entities/achievement/PersonAvatar.vue';
import type { Outing } from '~/shared/api/types';
import { formatAmount } from '~/shared/lib/format-price';
import { LEVELS, TRANSPORTS, optionIcon } from '~/shared/lib/trail-options';

// `compact` stacks everything in one column, for narrow sidebars.
const props = defineProps<{ outing: Outing; compact?: boolean }>();
const { locale } = useI18n();
const localePath = useLocalePath();

const starts = computed(() => new Date(props.outing.startsAt));
const left = computed(() => Math.max(0, props.outing.seats - props.outing.taken));
const day = computed(() =>
  starts.value.toLocaleDateString(locale.value, { day: 'numeric', timeZone: 'Asia/Bishkek' }),
);
const month = computed(() =>
  starts.value.toLocaleDateString(locale.value, { month: 'short', timeZone: 'Asia/Bishkek' }),
);
const weekdayTime = computed(() =>
  starts.value.toLocaleString(locale.value, {
    weekday: 'short',
    hour: '2-digit',
    minute: '2-digit',
    timeZone: 'Asia/Bishkek',
  }),
);
</script>

<template>
  <NuxtLink
    :to="localePath(`/outings/${outing.id}`)"
    class="group flex rounded-card border border-border bg-surface transition hover:-translate-y-0.5 hover:border-brand-400 hover:shadow-md"
    :class="[compact ? 'gap-3 p-3' : 'gap-4 p-4', { 'opacity-60': outing.status === 'cancelled' }]"
  >
    <div class="flex w-14 shrink-0 flex-col items-center justify-center rounded-xl bg-gradient-to-b from-brand-500 to-brand-700 py-2 text-white">
      <span class="text-2xl font-black leading-none">{{ day }}</span>
      <span class="text-xs uppercase">{{ month }}</span>
    </div>
    <div class="min-w-0 flex-1">
      <p class="text-xs text-text-muted">{{ weekdayTime }}</p>
      <h3 class="truncate font-semibold text-text group-hover:text-brand-600 dark:group-hover:text-brand-400">{{ outing.title }}</h3>
      <p v-if="outing.trail" class="truncate text-sm text-text-muted">
        <UIcon name="i-mdi-map-marker-path" class="mr-0.5 inline h-4 w-4 align-text-bottom" />{{ outing.trail.name }}
      </p>
      <div v-if="!compact" class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-text-muted">
        <span class="inline-flex items-center gap-1">
          <UIcon :name="optionIcon(TRANSPORTS, outing.transport)" class="h-4 w-4" />{{ $t(`outings.transport.${outing.transport}`) }}
        </span>
        <span class="inline-flex items-center gap-1">
          <UIcon :name="optionIcon(LEVELS, outing.level)" class="h-4 w-4" />{{ $t(`outings.level.${outing.level}`) }}
        </span>
        <span v-if="outing.costPerPerson !== null">
          {{ outing.costPerPerson === 0 ? $t('outings.free') : `${formatAmount(locale, outing.costPerPerson)} ${$t('listing.currency')}` }}
        </span>
      </div>
      <p v-if="compact" class="mt-1 text-xs font-semibold" :class="left === 0 ? 'text-text-muted' : 'text-emerald-600 dark:text-emerald-400'">
        {{ outing.status === 'cancelled' ? $t('outings.cancelled') : left === 0 ? $t('outings.full') : $t('outings.seatsLeft', { count: left }) }}
      </p>
    </div>
    <div v-if="!compact" class="flex shrink-0 flex-col items-end justify-between">
      <PersonAvatar :person="outing.organizer" size="h-8 w-8 text-xs" />
      <span
        v-if="outing.status === 'cancelled'"
        class="rounded-full bg-gray-500/15 px-2 py-0.5 text-xs font-semibold text-text-muted"
      >
        {{ $t('outings.cancelled') }}
      </span>
      <span
        v-else
        class="rounded-full px-2 py-0.5 text-xs font-semibold"
        :class="left === 0 ? 'bg-gray-500/15 text-text-muted' : 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-300'"
      >
        {{ left === 0 ? $t('outings.full') : $t('outings.seatsLeft', { count: left }) }}
      </span>
    </div>
  </NuxtLink>
</template>
