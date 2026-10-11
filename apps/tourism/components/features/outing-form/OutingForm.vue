<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import IconPicker from '~/components/features/icon-picker/IconPicker.vue';
import { ApiError, useApiClient } from '~/shared/api/client';
import type { Outing, Trail } from '~/shared/api/types';
import { LEVELS, TRANSPORTS } from '~/shared/lib/trail-options';

// Organize an outing, or edit one (`outing` set).
const props = defineProps<{ outing?: Outing; trailSlug?: string }>();

const { t } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const queryClient = useQueryClient();

const trailsQuery = useQuery({
  queryKey: ['trails', 'all'],
  queryFn: () => api.get<Trail[]>('/trails?sort=name'),
});
await trailsQuery.suspense();
const trailOptions = computed(() => [
  { value: '', label: t('outings.form.noTrail') },
  ...(trailsQuery.data.value ?? []).map((tr) => ({ value: tr.id, label: tr.name })),
]);

// datetime-local works in the browser's zone; outings are planned in
// Bishkek time (UTC+6), so convert explicitly.
function toLocalInput(iso: string): string {
  const shifted = new Date(new Date(iso).getTime() + 6 * 3_600_000);
  return shifted.toISOString().slice(0, 16);
}
function fromLocalInput(value: string): string {
  return `${value}:00+06:00`;
}
function defaultStart(): string {
  // Next Saturday, 06:00.
  const now = new Date(Date.now() + 6 * 3_600_000);
  const days = (6 - now.getUTCDay() + 7) % 7 || 7;
  const sat = new Date(
    Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate() + days, 6, 0),
  );
  return sat.toISOString().slice(0, 16);
}

const o = props.outing;
const trailId = ref(o?.trail?.id ?? '');
const title = ref(o?.title ?? '');
const startsAt = ref(o ? toLocalInput(o.startsAt) : defaultStart());
const returnsAt = ref(o?.returnsAt ? toLocalInput(o.returnsAt) : '');
const meetingPoint = ref(o?.meetingPoint ?? '');
const transport = ref(o?.transport ?? 'car');
const seats = ref(o?.seats ?? 3);
const costPerPerson = ref<number | undefined>(o?.costPerPerson ?? undefined);
const level = ref(o?.level ?? 'beginner');
const description = ref(o?.description ?? '');
const contact = ref(o?.contact ?? '');
const errors = ref<Record<string, string>>({});
const submitError = ref<string | null>(null);

// Prefill from ?trail=slug once routes load.
watch(
  () => trailsQuery.data.value,
  (trails: Trail[] | undefined) => {
    if (o || !props.trailSlug || !trails) return;
    const match = trails.find((tr: Trail) => tr.slug === props.trailSlug);
    if (match && !trailId.value) {
      trailId.value = match.id;
      if (!title.value) title.value = match.name;
    }
  },
  { immediate: true },
);

const transportOptions = computed(() =>
  TRANSPORTS.map((x) => ({ ...x, label: t(`outings.transport.${x.value}`) })),
);
const levelOptions = computed(() =>
  LEVELS.map((x) => ({ ...x, label: t(`outings.level.${x.value}`) })),
);

const save = useMutation({
  mutationFn: () => {
    const body = {
      trailId: trailId.value || null,
      title: title.value,
      startsAt: fromLocalInput(startsAt.value),
      returnsAt: returnsAt.value ? fromLocalInput(returnsAt.value) : null,
      meetingPoint: meetingPoint.value,
      transport: transport.value,
      seats: Number(seats.value),
      costPerPerson:
        costPerPerson.value === undefined || (costPerPerson.value as unknown) === ''
          ? null
          : Number(costPerPerson.value),
      level: level.value,
      description: description.value.trim() || null,
      contact: contact.value.trim() || null,
    };
    return o ? api.put<Outing>(`/outings/${o.id}`, body) : api.post<Outing>('/outings', body);
  },
  onSuccess: (saved) => {
    queryClient.invalidateQueries({ queryKey: ['outings'] });
    queryClient.invalidateQueries({ queryKey: ['outing', saved.id] });
    navigateTo(localePath(`/outings/${saved.id}`));
  },
  onError: (error) => {
    submitError.value = t('outings.form.saveError');
    if (error instanceof ApiError && error.status === 400)
      submitError.value = t('outings.form.checkFields');
  },
});

function submit(): void {
  errors.value = {};
  submitError.value = null;
  if (!title.value.trim()) errors.value.title = t('login.required');
  if (!meetingPoint.value.trim()) errors.value.meetingPoint = t('login.required');
  if (!startsAt.value || new Date(fromLocalInput(startsAt.value)) <= new Date())
    errors.value.startsAt = t('outings.form.futureStart');
  if (returnsAt.value && returnsAt.value <= startsAt.value)
    errors.value.returnsAt = t('outings.form.returnAfterStart');
  if (Object.keys(errors.value).length === 0) save.mutate();
}
</script>

<template>
  <form class="mt-8 flex flex-col gap-6" @submit.prevent="submit">
    <UFormGroup :label="$t('outings.form.trail')" :hint="$t('outings.form.trailHint')">
      <USelect v-model="trailId" :options="trailOptions" option-attribute="label" searchable />
    </UFormGroup>
    <UFormGroup :label="$t('outings.form.title')" :error="errors.title">
      <UInput v-model="title" size="lg" :maxlength="160" :placeholder="$t('outings.form.titlePlaceholder')" />
    </UFormGroup>
    <div class="grid gap-4 sm:grid-cols-2">
      <UFormGroup :label="$t('outings.form.startsAt')" :error="errors.startsAt" :hint="$t('outings.form.bishkekTime')">
        <UInput v-model="startsAt" type="datetime-local" />
      </UFormGroup>
      <UFormGroup :label="$t('outings.form.returnsAt')" :error="errors.returnsAt">
        <UInput v-model="returnsAt" type="datetime-local" />
      </UFormGroup>
    </div>
    <UFormGroup :label="$t('outings.form.meetingPoint')" :error="errors.meetingPoint">
      <UInput v-model="meetingPoint" :maxlength="200" :placeholder="$t('outings.form.meetingPointPlaceholder')" />
    </UFormGroup>
    <UFormGroup :label="$t('outings.form.transport')">
      <IconPicker v-model="transport" :options="transportOptions" />
    </UFormGroup>
    <div class="grid gap-4 sm:grid-cols-2">
      <UFormGroup :label="$t('outings.form.seats')" :hint="$t('outings.form.seatsHint')">
        <UInput v-model.number="seats" type="number" min="1" max="50" />
      </UFormGroup>
      <UFormGroup :label="$t('outings.form.cost')" :hint="$t('outings.form.costHint')">
        <UInput v-model.number="costPerPerson" type="number" min="0" />
      </UFormGroup>
    </div>
    <UFormGroup :label="$t('outings.form.level')">
      <IconPicker v-model="level" :options="levelOptions" />
    </UFormGroup>
    <UFormGroup :label="$t('outings.form.description')">
      <UTextarea v-model="description" :rows="4" :maxlength="2000" :placeholder="$t('outings.form.descriptionPlaceholder')" />
    </UFormGroup>
    <UFormGroup :label="$t('outings.form.contact')" :hint="$t('outings.form.contactHint')">
      <UInput v-model="contact" :maxlength="120" placeholder="@telegram / +996…" />
    </UFormGroup>

    <p v-if="submitError" class="text-sm text-brand-600">{{ submitError }}</p>
    <UButton type="submit" color="primary" size="lg" block :loading="save.isPending.value">
      {{ outing ? $t('outings.form.save') : $t('outings.form.create') }}
    </UButton>
  </form>
</template>
