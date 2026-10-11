<script setup lang="ts">
import { useMutation, useQueryClient } from '@tanstack/vue-query';
import TiptapEditor from '~/components/features/guide-editor/TiptapEditor.vue';
import IconPicker from '~/components/features/icon-picker/IconPicker.vue';
import PhotoUploader from '~/components/features/photo-uploader/PhotoUploader.vue';
import ElevationProfile from '~/components/widgets/trail-map/ElevationProfile.vue';
import { ApiError, useApiClient } from '~/shared/api/client';
import type { TrackPoint, Trail, TrailSegment } from '~/shared/api/types';
import { parseGpx } from '~/shared/lib/gpx';
import { AREAS, GEAR, GRADES, KINDS, SEASONS, formatGrade } from '~/shared/lib/trail-options';

// Publish a route, or edit one (`trail` set). Partners only.
const props = defineProps<{ trail?: Trail }>();

const { t } = useI18n();
const localePath = useLocalePath();
const api = useApiClient();
const queryClient = useQueryClient();

const tr = props.trail;
const name = ref(tr?.name ?? '');
const summary = ref(tr?.summary ?? '');
const description = ref(tr?.description ?? '');
const area = ref(tr?.area ?? 'ala_archa');
const kind = ref(tr?.kind ?? 'hike');
const grade = ref(tr?.grade ?? 'trail');
const seasons = ref<string[]>(tr?.seasons ?? ['summer']);
const gear = ref<string[]>(tr?.gear ?? []);
const summitElevation = ref<number | undefined>(tr?.summitElevation ?? undefined);
const startElevation = ref<number | undefined>(tr?.startElevation ?? undefined);
const elevationGain = ref<number | undefined>(tr?.elevationGain ?? undefined);
const distanceKm = ref<number | undefined>(tr?.distanceKm ?? undefined);
const durationHours = ref<number | undefined>(tr?.durationHours ?? undefined);
const days = ref(tr?.days ?? 1);
const segments = ref<TrailSegment[]>(tr?.segments.map((s) => ({ ...s })) ?? []);
const gettingThere = ref(tr?.gettingThere ?? '');
const track = ref<TrackPoint[]>(tr?.track ?? []);
const gpxUrl = ref<string | null>(tr?.gpxUrl ?? null);
const photos = ref<string[]>(
  tr
    ? [
        ...(tr.coverImageUrl ? [tr.coverImageUrl] : []),
        ...tr.photos.filter((p) => p !== tr.coverImageUrl),
      ]
    : [],
);
const isPublished = ref(tr?.isPublished ?? true);
const gpxError = ref<string | null>(null);
const gpxUploading = ref(false);
const errors = ref<Record<string, string>>({});
const submitError = ref<string | null>(null);

const kindOptions = computed(() =>
  KINDS.map((k) => ({ ...k, label: t(`trails.kind.${k.value}`) })),
);
const seasonOptions = computed(() =>
  SEASONS.map((s) => ({ ...s, label: t(`trails.season.${s.value}`) })),
);
const gearOptions = computed(() => GEAR.map((g) => ({ ...g, label: t(`trails.gear.${g.value}`) })));
const areaOptions = computed(() => AREAS.map((a) => ({ value: a, label: t(`trails.area.${a}`) })));
const gradeOptions = computed(() =>
  GRADES.map((g) => ({
    value: g,
    label: g === 'trail' ? t('trails.grade.trail') : formatGrade(g),
  })),
);

async function onGpx(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = '';
  if (!file) return;
  gpxError.value = null;
  try {
    const points = parseGpx(await file.text());
    if (points.length < 2) throw new Error('empty');
    track.value = points;
    const eles = points.map((p) => p[2]).filter((e): e is number => e !== null);
    if (eles.length > 0) {
      startElevation.value ??= eles[0];
      summitElevation.value ??= Math.max(...eles);
    }
    gpxUploading.value = true;
    const formData = new FormData();
    formData.append('file', file);
    gpxUrl.value = (await api.upload<{ url: string }>('/uploads/file', formData)).url;
  } catch {
    gpxError.value = t('trails.form.gpxError');
  } finally {
    gpxUploading.value = false;
  }
}

function clearTrack(): void {
  track.value = [];
  gpxUrl.value = null;
}

function addSegment(): void {
  const last = segments.value.at(-1);
  segments.value.push({ from: last?.to ?? '', to: '', hours: 1 });
}

function num(v: number | undefined | string): number | null {
  return v === undefined || v === '' || Number.isNaN(Number(v)) ? null : Number(v);
}

const save = useMutation({
  mutationFn: () => {
    const body = {
      name: name.value,
      summary: summary.value.trim() || null,
      description: description.value,
      area: area.value,
      kind: kind.value,
      grade: grade.value,
      seasons: seasons.value,
      gear: gear.value,
      summitElevation: num(summitElevation.value),
      startElevation: num(startElevation.value),
      elevationGain: num(elevationGain.value),
      distanceKm: num(distanceKm.value),
      durationHours: num(durationHours.value),
      days: Number(days.value) || 1,
      segments: segments.value
        .filter((s: TrailSegment) => s.from.trim() && s.to.trim() && s.hours > 0)
        .map((s: TrailSegment) => ({ ...s, hours: Number(s.hours) })),
      gettingThere: gettingThere.value.trim() || null,
      track: track.value,
      gpxUrl: gpxUrl.value,
      coverImageUrl: photos.value[0] ?? null,
      photos: photos.value,
      isPublished: isPublished.value,
    };
    if (tr) return api.put<Trail>(`/trails/${tr.id}`, body);
    // slugify adds a random suffix, so names can repeat.
    return api.post<Trail>('/trails', { ...body, slug: slugify(name.value) });
  },
  onSuccess: (saved) => {
    queryClient.invalidateQueries({ queryKey: ['trails'] });
    queryClient.invalidateQueries({ queryKey: ['trail', saved.slug] });
    queryClient.invalidateQueries({ queryKey: ['my-trails'] });
    navigateTo(localePath(`/trails/${saved.slug}`));
  },
  onError: (error) => {
    submitError.value =
      error instanceof ApiError && error.status === 400
        ? t('outings.form.checkFields')
        : t('trails.form.saveError');
  },
});

function submit(): void {
  errors.value = {};
  submitError.value = null;
  if (!name.value.trim()) errors.value.name = t('login.required');
  if (Object.keys(errors.value).length === 0) save.mutate();
}
</script>

<template>
  <form class="mt-8 flex flex-col gap-8" @submit.prevent="submit">
    <section class="flex flex-col gap-5">
      <UFormGroup :label="$t('trails.form.name')" :error="errors.name">
        <UInput v-model="name" size="lg" :maxlength="160" :placeholder="$t('trails.form.namePlaceholder')" />
      </UFormGroup>
      <UFormGroup :label="$t('trails.form.summary')">
        <UInput v-model="summary" :maxlength="300" :placeholder="$t('trails.form.summaryPlaceholder')" />
      </UFormGroup>
      <div class="grid gap-4 sm:grid-cols-2">
        <UFormGroup :label="$t('trails.form.area')">
          <USelect v-model="area" :options="areaOptions" option-attribute="label" />
        </UFormGroup>
        <UFormGroup :label="$t('trails.form.grade')">
          <USelect v-model="grade" :options="gradeOptions" option-attribute="label" />
        </UFormGroup>
      </div>
      <UFormGroup :label="$t('trails.form.kind')">
        <IconPicker v-model="kind" :options="kindOptions" />
      </UFormGroup>
    </section>

    <section class="flex flex-col gap-4 rounded-card border border-border bg-surface p-4">
      <h2 class="font-semibold text-text">{{ $t('trails.form.trackTitle') }}</h2>
      <p class="text-sm text-text-muted">{{ $t('trails.form.trackHint') }}</p>
      <div class="flex flex-wrap items-center gap-3">
        <label class="cursor-pointer">
          <input type="file" accept=".gpx,application/gpx+xml" class="hidden" @change="onGpx" />
          <span class="inline-flex items-center gap-2 rounded-md bg-brand-600 px-3 py-2 text-sm font-medium text-white hover:bg-brand-700">
            <UIcon name="i-mdi-upload" class="h-4 w-4" />{{ $t('trails.form.uploadGpx') }}
          </span>
        </label>
        <span v-if="gpxUploading" class="text-sm text-text-muted">{{ $t('trails.form.uploading') }}</span>
        <span v-else-if="track.length > 0" class="text-sm text-text">
          {{ $t('trails.form.trackPoints', { count: track.length }) }}
          <button type="button" class="ml-2 text-brand-600 hover:underline" @click="clearTrack">{{ $t('trails.form.removeTrack') }}</button>
        </span>
      </div>
      <p v-if="gpxError" class="text-sm text-brand-600">{{ gpxError }}</p>
      <ElevationProfile v-if="track.length > 1" :track="track" />
    </section>

    <section class="grid gap-4 sm:grid-cols-3">
      <UFormGroup :label="$t('trails.form.summit')"><UInput v-model.number="summitElevation" type="number" min="500" max="7500" /></UFormGroup>
      <UFormGroup :label="$t('trails.form.start')"><UInput v-model.number="startElevation" type="number" min="300" max="7500" /></UFormGroup>
      <UFormGroup :label="$t('trails.form.gain')"><UInput v-model.number="elevationGain" type="number" min="0" max="7000" /></UFormGroup>
      <UFormGroup :label="$t('trails.form.distance')"><UInput v-model.number="distanceKm" type="number" step="0.1" min="0" /></UFormGroup>
      <UFormGroup :label="$t('trails.form.duration')"><UInput v-model.number="durationHours" type="number" step="0.5" min="0" /></UFormGroup>
      <UFormGroup :label="$t('trails.form.days')"><UInput v-model.number="days" type="number" min="1" max="60" /></UFormGroup>
    </section>

    <section class="flex flex-col gap-3">
      <div class="flex items-center justify-between">
        <h2 class="font-semibold text-text">{{ $t('trails.form.segments') }}</h2>
        <UButton size="xs" color="gray" icon="i-heroicons-plus" @click="addSegment">{{ $t('trails.form.addSegment') }}</UButton>
      </div>
      <div v-for="(segment, i) in segments" :key="i" class="grid grid-cols-[1fr_1fr_80px_auto] items-center gap-2">
        <UInput v-model="segment.from" :placeholder="$t('trails.form.from')" :maxlength="100" />
        <UInput v-model="segment.to" :placeholder="$t('trails.form.to')" :maxlength="100" />
        <UInput v-model.number="segment.hours" type="number" step="0.5" min="0.5" :placeholder="$t('trails.form.hours')" />
        <UButton color="gray" variant="ghost" icon="i-heroicons-x-mark" @click="segments.splice(i, 1)" />
      </div>
      <p v-if="segments.length === 0" class="text-sm text-text-muted">{{ $t('trails.form.segmentsHint') }}</p>
    </section>

    <UFormGroup :label="$t('trails.form.seasons')">
      <IconPicker v-model="seasons" :options="seasonOptions" multiple />
    </UFormGroup>
    <UFormGroup :label="$t('trails.form.gear')">
      <IconPicker v-model="gear" :options="gearOptions" multiple />
    </UFormGroup>
    <UFormGroup :label="$t('trails.form.gettingThere')">
      <UTextarea v-model="gettingThere" :rows="3" :maxlength="2000" :placeholder="$t('trails.form.gettingTherePlaceholder')" />
    </UFormGroup>
    <UFormGroup :label="$t('trails.form.description')">
      <TiptapEditor v-model="description" :placeholder="$t('trails.form.descriptionPlaceholder')" />
    </UFormGroup>
    <UFormGroup :label="$t('trails.form.photos')" :hint="$t('trails.form.photosHint')">
      <PhotoUploader v-model="photos" :max="12" folder="trails" show-cover-badge />
    </UFormGroup>
    <label class="inline-flex items-center gap-3 text-sm text-text"><UToggle v-model="isPublished" />{{ $t('trails.form.publish') }}</label>

    <p v-if="submitError" class="text-sm text-brand-600">{{ submitError }}</p>
    <UButton type="submit" color="primary" size="lg" block :loading="save.isPending.value">
      {{ trail ? $t('trails.form.save') : $t('trails.form.create') }}
    </UButton>
  </form>
</template>
