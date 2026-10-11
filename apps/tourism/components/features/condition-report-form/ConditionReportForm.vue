<script setup lang="ts">
import { useMutation, useQueryClient } from '@tanstack/vue-query';
import IconPicker from '~/components/features/icon-picker/IconPicker.vue';
import PhotoUploader from '~/components/features/photo-uploader/PhotoUploader.vue';
import { useApiClient } from '~/shared/api/client';
import type { ConditionReport } from '~/shared/api/types';
import { AVALANCHE_RISKS, TRACTIONS, TRAIL_STATES } from '~/shared/lib/trail-options';

// "What did the route look like?" — a quick report anyone signed in can
// file after a trip.
const props = defineProps<{ trailId: string; trailSlug: string }>();
const open = defineModel<boolean>('open', { required: true });

const { t } = useI18n();
const api = useApiClient();
const queryClient = useQueryClient();

function today(): string {
  return new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Bishkek' });
}
function daysBack(days: number): string {
  return new Date(Date.now() - days * 86_400_000).toLocaleDateString('sv-SE', {
    timeZone: 'Asia/Bishkek',
  });
}

const observedOn = ref(today());
const trailState = ref('clear');
const traction = ref('none');
const avalancheRisk = ref('');
const snowLine = ref<number | undefined>();
const reached = ref('');
const text = ref('');
const photos = ref<string[]>([]);
const error = ref<string | null>(null);

const stateOptions = computed(() =>
  TRAIL_STATES.map((o) => ({ ...o, label: t(`trails.state.${o.value}`) })),
);
const tractionOptions = computed(() =>
  TRACTIONS.map((o) => ({ ...o, label: t(`trails.traction.${o.value}`) })),
);
const avalancheOptions = computed(() => [
  { value: '', label: t('trails.conditions.avalancheUnknown') },
  ...AVALANCHE_RISKS.map((r) => ({ value: r, label: t(`trails.avalanche.${r}`) })),
]);

const submit = useMutation({
  mutationFn: () =>
    api.post<ConditionReport>(`/trails/${props.trailId}/conditions`, {
      observedOn: observedOn.value,
      trailState: trailState.value,
      traction: traction.value,
      avalancheRisk: avalancheRisk.value || null,
      snowLine: snowLine.value || null,
      reached: reached.value.trim() || null,
      text: text.value.trim() || null,
      photos: photos.value,
    }),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ['trail-conditions', props.trailSlug] });
    queryClient.invalidateQueries({ queryKey: ['trail', props.trailSlug] });
    queryClient.invalidateQueries({ queryKey: ['conditions'] });
    queryClient.invalidateQueries({ queryKey: ['trails'] });
    text.value = '';
    photos.value = [];
    open.value = false;
  },
  onError: () => {
    error.value = t('trails.conditions.saveError');
  },
});
</script>

<template>
  <UModal v-model="open" :ui="{ width: 'sm:max-w-xl' }">
    <form class="flex flex-col gap-5 p-6" @submit.prevent="submit.mutate()">
      <div>
        <h2 class="text-lg font-bold text-text">{{ $t('trails.conditions.formTitle') }}</h2>
        <p class="text-sm text-text-muted">{{ $t('trails.conditions.formHint') }}</p>
      </div>

      <UFormGroup :label="$t('trails.conditions.date')">
        <UInput v-model="observedOn" type="date" :min="daysBack(30)" :max="today()" />
      </UFormGroup>
      <UFormGroup :label="$t('trails.conditions.trailState')">
        <IconPicker v-model="trailState" :options="stateOptions" />
      </UFormGroup>
      <UFormGroup :label="$t('trails.conditions.traction')">
        <IconPicker v-model="traction" :options="tractionOptions" />
      </UFormGroup>
      <div class="grid gap-4 sm:grid-cols-2">
        <UFormGroup :label="$t('trails.conditions.avalanche')">
          <USelect v-model="avalancheRisk" :options="avalancheOptions" option-attribute="label" />
        </UFormGroup>
        <UFormGroup :label="$t('trails.conditions.snowLine')">
          <UInput v-model.number="snowLine" type="number" min="500" max="7500" :placeholder="$t('trails.conditions.snowLinePlaceholder')" />
        </UFormGroup>
      </div>
      <UFormGroup :label="$t('trails.conditions.reached')">
        <UInput v-model="reached" :maxlength="120" :placeholder="$t('trails.conditions.reachedPlaceholder')" />
      </UFormGroup>
      <UFormGroup :label="$t('trails.conditions.text')">
        <UTextarea v-model="text" :rows="3" :maxlength="2000" :placeholder="$t('trails.conditions.textPlaceholder')" />
      </UFormGroup>
      <UFormGroup :label="$t('trails.conditions.photos')">
        <PhotoUploader v-model="photos" :max="6" folder="conditions" />
      </UFormGroup>

      <p v-if="error" class="text-sm text-brand-600">{{ error }}</p>
      <div class="flex justify-end gap-2">
        <UButton color="gray" variant="ghost" @click="open = false">{{ $t('trails.cancel') }}</UButton>
        <UButton type="submit" color="primary" :loading="submit.isPending.value">{{ $t('trails.conditions.submit') }}</UButton>
      </div>
    </form>
  </UModal>
</template>
