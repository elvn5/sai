import { injectLang } from '@/core/i18n/current-lang';
import { formatDate } from '@/core/lib/format';
import {
  ChangeDetectionStrategy,
  Component,
  afterNextRender,
  computed,
  input,
  signal,
} from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

export interface WeatherPoint {
  label: string;
  meters: number;
}

interface Daily {
  time: string[];
  weather_code: number[];
  temperature_2m_max: number[];
  temperature_2m_min: number[];
  precipitation_sum: number[];
  wind_speed_10m_max: number[];
}

interface DayForecast {
  date: string;
  icon: string;
  // Translation key of the sky.
  sky: string;
  max: number;
  min: number;
  rain: number;
  wind: number;
}

// WMO weather codes → icon and translation key.
function sky(code: number): { icon: string; key: string } {
  if (code === 0) return { icon: 'icon-[mdi--weather-sunny]', key: 'clear' };
  if (code <= 2) return { icon: 'icon-[mdi--weather-partly-cloudy]', key: 'partly' };
  if (code === 3) return { icon: 'icon-[mdi--weather-cloudy]', key: 'cloudy' };
  if (code <= 48) return { icon: 'icon-[mdi--weather-fog]', key: 'fog' };
  if (code <= 67 || (code >= 80 && code <= 82))
    return { icon: 'icon-[mdi--weather-pouring]', key: 'rain' };
  if (code <= 77 || code === 85 || code === 86)
    return { icon: 'icon-[mdi--weather-snowy-heavy]', key: 'snow' };
  return { icon: 'icon-[mdi--weather-lightning-rainy]', key: 'storm' };
}

function days(daily: Daily): DayForecast[] {
  return daily.time.map((date, i) => {
    const { icon, key } = sky(daily.weather_code[i] ?? 0);
    return {
      date,
      icon,
      sky: `trails.weather.${key}`,
      max: Math.round(daily.temperature_2m_max[i] ?? 0),
      min: Math.round(daily.temperature_2m_min[i] ?? 0),
      rain: daily.precipitation_sum[i] ?? 0,
      wind: Math.round(daily.wind_speed_10m_max[i] ?? 0),
    };
  });
}

// A 5-day forecast for the route's start, adjusted to the start's and the
// summit's height, from Open-Meteo (free, no key). Fetched in the browser:
// it changes hourly, so it stays out of the cached page.
@Component({
  selector: 'app-mountain-weather',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <section class="ui-card p-4" aria-labelledby="weather-title">
      <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h2 id="weather-title" class="flex items-center gap-2 font-semibold text-text">
          <span class="icon-[mdi--weather-partly-snowy-rainy] size-5 text-brand-600 dark:text-brand-300" aria-hidden="true"></span>
          {{ 'trails.weather.title' | transloco }}
        </h2>
        @if (elevations().length > 1) {
          <div class="flex gap-1" role="group">
            @for (point of elevations(); track point.label; let i = $index) {
              <button type="button" class="ui-pill px-2.5 py-0.5 text-xs" [attr.aria-pressed]="selected() === i" (click)="selected.set(i)">
                {{ point.label }} · {{ point.meters }} {{ 'trails.units.m' | transloco }}
              </button>
            }
          </div>
        }
      </div>
      @if (failed()) {
        <p class="text-sm text-text-muted">{{ 'trails.weather.error' | transloco }}</p>
      } @else if (forecast(); as list) {
        <ol class="grid grid-cols-5 gap-2">
          @for (day of list; track day.date; let i = $index) {
            <li class="flex flex-col items-center gap-1 rounded-xl bg-text/5 px-1 py-2 text-center">
              <span class="text-xs text-text-muted">
                @if (i === 0) { {{ 'trails.weather.today' | transloco }} } @else { {{ dayLabel(day.date) }} }
              </span>
              <span [class]="day.icon + ' size-7 text-text'" role="img" [attr.aria-label]="day.sky | transloco"></span>
              <span class="text-sm font-bold text-text">{{ day.max }}°</span>
              <span class="text-xs text-text-muted">{{ day.min }}°</span>
              @if (day.rain > 0.2) {
                <span class="text-[11px] text-sky-700 dark:text-sky-300">{{ day.rain.toFixed(1) }} {{ 'trails.units.mm' | transloco }}</span>
              }
              <span class="text-[11px] text-text-muted">{{ day.wind }} {{ 'trails.units.kmh' | transloco }}</span>
            </li>
          }
        </ol>
      } @else {
        <div class="h-28 animate-pulse rounded-xl bg-text/5"></div>
      }
      <p class="mt-2 text-[11px] text-text-muted">
        {{ 'trails.weather.source' | transloco: { meters: elevations()[selected()]?.meters ?? 0 } }}
        <a href="https://open-meteo.com" target="_blank" rel="noopener noreferrer" class="underline">Open-Meteo</a>
      </p>
    </section>
  `,
})
export class MountainWeather {
  readonly latitude = input.required<number>();
  readonly longitude = input.required<number>();
  readonly elevations = input.required<WeatherPoint[]>();

  private readonly lang = injectLang();
  protected readonly selected = signal(0);
  private readonly forecasts = signal<DayForecast[][] | null>(null);
  protected readonly failed = signal(false);
  protected readonly forecast = computed(() => this.forecasts()?.[this.selected()] ?? null);

  constructor() {
    afterNextRender(() => void this.load());
  }

  protected dayLabel(date: string): string {
    return formatDate(this.lang(), date, { weekday: 'short', day: 'numeric' });
  }

  private async load(): Promise<void> {
    try {
      this.forecasts.set(
        await Promise.all(this.elevations().map((point) => this.fetchDays(point.meters))),
      );
    } catch {
      this.failed.set(true);
    }
  }

  private async fetchDays(meters: number): Promise<DayForecast[]> {
    const params = new URLSearchParams({
      latitude: String(this.latitude()),
      longitude: String(this.longitude()),
      elevation: String(meters),
      daily:
        'weather_code,temperature_2m_max,temperature_2m_min,precipitation_sum,wind_speed_10m_max',
      timezone: 'Asia/Bishkek',
      forecast_days: '5',
    });
    // Plain fetch: a third-party API, not ours (no base URL, no cookies).
    const response = await fetch(`https://api.open-meteo.com/v1/forecast?${params}`);
    if (!response.ok) throw new Error(String(response.status));
    return days(((await response.json()) as { daily: Daily }).daily);
  }
}
