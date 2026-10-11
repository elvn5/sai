<script setup lang="ts">
// A 5-day forecast for a point, adjusted to a given altitude, from
// Open-Meteo (free, no key). Fetched in the browser only.
const props = defineProps<{
  latitude: number;
  longitude: number;
  elevations: { label: string; meters: number }[];
}>();

interface Daily {
  time: string[];
  weather_code: number[];
  temperature_2m_max: number[];
  temperature_2m_min: number[];
  precipitation_sum: number[];
  wind_speed_10m_max: number[];
}

const { locale, t } = useI18n();
const selected = ref(0);
const forecasts = ref<(Daily | null)[]>([]);
const failed = ref(false);

const elevation = computed(() => props.elevations[selected.value]);
const daily = computed(() => forecasts.value[selected.value] ?? null);

async function load(): Promise<void> {
  try {
    forecasts.value = await Promise.all(
      props.elevations.map(async (e) => {
        const params = new URLSearchParams({
          latitude: String(props.latitude),
          longitude: String(props.longitude),
          elevation: String(e.meters),
          daily:
            'weather_code,temperature_2m_max,temperature_2m_min,precipitation_sum,wind_speed_10m_max',
          timezone: 'Asia/Bishkek',
          forecast_days: '5',
        });
        const res = await fetch(`https://api.open-meteo.com/v1/forecast?${params}`);
        if (!res.ok) throw new Error(String(res.status));
        return ((await res.json()) as { daily: Daily }).daily;
      }),
    );
  } catch {
    failed.value = true;
  }
}

onMounted(load);

// WMO weather codes → icon and i18n key.
function weather(code: number): { icon: string; key: string } {
  if (code === 0) return { icon: 'i-mdi-weather-sunny', key: 'clear' };
  if (code <= 2) return { icon: 'i-mdi-weather-partly-cloudy', key: 'partly' };
  if (code === 3) return { icon: 'i-mdi-weather-cloudy', key: 'cloudy' };
  if (code <= 48) return { icon: 'i-mdi-weather-fog', key: 'fog' };
  if (code <= 67 || (code >= 80 && code <= 82))
    return { icon: 'i-mdi-weather-pouring', key: 'rain' };
  if (code <= 77 || code === 85 || code === 86)
    return { icon: 'i-mdi-weather-snowy-heavy', key: 'snow' };
  return { icon: 'i-mdi-weather-lightning-rainy', key: 'storm' };
}

function dayLabel(iso: string, index: number): string {
  if (index === 0) return t('trails.weather.today');
  return new Date(`${iso}T12:00:00`).toLocaleDateString(locale.value, {
    weekday: 'short',
    day: 'numeric',
  });
}
</script>

<template>
  <div class="rounded-card border border-border bg-surface p-4">
    <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
      <h3 class="flex items-center gap-2 font-semibold text-text">
        <UIcon name="i-mdi-weather-partly-snowy-rainy" class="h-5 w-5 text-brand-600 dark:text-brand-400" />
        {{ t('trails.weather.title') }}
      </h3>
      <div v-if="elevations.length > 1" class="flex gap-1">
        <UButton
          v-for="(e, i) in elevations"
          :key="e.label"
          size="xs"
          :color="selected === i ? 'primary' : 'gray'"
          :variant="selected === i ? 'solid' : 'soft'"
          @click="selected = i"
        >
          {{ e.label }} · {{ e.meters }} {{ t('trails.units.m') }}
        </UButton>
      </div>
    </div>
    <p v-if="failed" class="text-sm text-text-muted">{{ t('trails.weather.error') }}</p>
    <div v-else-if="daily" class="grid grid-cols-5 gap-2">
      <div
        v-for="(day, i) in daily.time"
        :key="day"
        class="flex flex-col items-center gap-1 rounded-xl bg-bg px-1 py-2 text-center"
        :title="t(`trails.weather.${weather(daily.weather_code[i] ?? 0).key}`)"
      >
        <span class="text-xs text-text-muted">{{ dayLabel(day, i) }}</span>
        <UIcon :name="weather(daily.weather_code[i] ?? 0).icon" class="h-7 w-7 text-text" />
        <span class="text-sm font-bold text-text">{{ Math.round(daily.temperature_2m_max[i] ?? 0) }}°</span>
        <span class="text-xs text-text-muted">{{ Math.round(daily.temperature_2m_min[i] ?? 0) }}°</span>
        <span v-if="(daily.precipitation_sum[i] ?? 0) > 0.2" class="text-[11px] text-sky-600 dark:text-sky-400">
          {{ (daily.precipitation_sum[i] ?? 0).toFixed(1) }} {{ t('trails.units.mm') }}
        </span>
        <span class="text-[11px] text-text-muted">{{ Math.round(daily.wind_speed_10m_max[i] ?? 0) }} {{ t('trails.units.kmh') }}</span>
      </div>
    </div>
    <div v-else class="h-28 animate-pulse rounded-xl bg-bg" />
    <p class="mt-2 text-[11px] text-text-muted">
      {{ t('trails.weather.source', { meters: elevation?.meters ?? 0 }) }}
      <a href="https://open-meteo.com" target="_blank" rel="noopener noreferrer" class="underline">Open-Meteo</a>
    </p>
  </div>
</template>
