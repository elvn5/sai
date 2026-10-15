import type { TrackPoint } from '@/core/api/types';
import { cumulativeKm } from '@/core/lib/gpx';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  output,
  signal,
} from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

const WIDTH = 800;
const HEIGHT = 200;
const PAD = { top: 12, right: 12, bottom: 24, left: 44 };

interface ProfilePoint {
  // Index into the track.
  i: number;
  km: number;
  ele: number;
}

// The track's elevation over distance, as SVG (renders on the server).
// Hovering (or dragging a finger) shows the height and distance there and
// reports the track point, which the map marks.
@Component({
  selector: 'app-elevation-profile',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    @if (points().length > 1) {
      <figure class="rounded-card border border-border bg-surface p-3">
        <figcaption class="sr-only">{{ summary() }}</figcaption>
        <svg
          [attr.viewBox]="'0 0 ' + width + ' ' + height"
          class="h-auto w-full touch-pan-y select-none"
          role="img"
          [attr.aria-label]="summary()"
          (pointermove)="onMove($event)"
          (pointerdown)="onMove($event)"
          (pointerleave)="onLeave()"
          (pointercancel)="onLeave()"
        >
          <defs>
            <linearGradient [attr.id]="gradientId" x1="0" x2="0" y1="0" y2="1">
              <stop offset="0%" stop-color="var(--color-brand-500)" stop-opacity="0.35" />
              <stop offset="100%" stop-color="var(--color-brand-500)" stop-opacity="0.03" />
            </linearGradient>
          </defs>
          @for (tick of ticks(); track tick) {
            <line [attr.x1]="pad.left" [attr.x2]="width - pad.right" [attr.y1]="y(tick)" [attr.y2]="y(tick)" stroke="var(--color-border)" stroke-dasharray="3 4" />
            <text [attr.x]="pad.left - 6" [attr.y]="y(tick) + 4" text-anchor="end" class="fill-text-muted text-[11px]">{{ tick }}</text>
          }
          <path [attr.d]="areaPath()" [attr.fill]="'url(#' + gradientId + ')'" />
          <path [attr.d]="linePath()" fill="none" stroke="var(--color-brand-500)" stroke-width="2" stroke-linejoin="round" stroke-linecap="round" />
          <text [attr.x]="width - pad.right" [attr.y]="height - 6" text-anchor="end" class="fill-text-muted text-[11px]">{{ totalKm().toFixed(1) }} {{ 'trails.units.km' | transloco }}</text>
          @if (hovered(); as point) {
            <line [attr.x1]="x(point.km)" [attr.x2]="x(point.km)" [attr.y1]="pad.top" [attr.y2]="height - pad.bottom" stroke="var(--color-text-muted)" />
            <circle [attr.cx]="x(point.km)" [attr.cy]="y(point.ele)" r="5" fill="var(--color-brand-600)" stroke="var(--color-surface)" stroke-width="2" />
            <text [attr.x]="labelX(point.km)" [attr.y]="pad.top + 14" class="fill-text text-[12px] font-semibold">
              {{ point.ele }} {{ 'trails.units.m' | transloco }} · {{ point.km.toFixed(1) }} {{ 'trails.units.km' | transloco }}
            </text>
          }
        </svg>
      </figure>
    }
  `,
})
export class ElevationProfile {
  readonly track = input.required<TrackPoint[]>();
  // The hovered track point's index, or null.
  readonly hover = output<number | null>();

  private readonly transloco = inject(TranslocoService);
  protected readonly width = WIDTH;
  protected readonly height = HEIGHT;
  protected readonly pad = PAD;
  // One gradient per profile (a form shows one next to the page's).
  protected readonly gradientId = `profile-fill-${Math.random().toString(36).slice(2, 8)}`;

  protected readonly points = computed((): ProfilePoint[] => {
    const track = this.track();
    const km = cumulativeKm(track);
    const out: ProfilePoint[] = [];
    track.forEach((point, i) => {
      if (point[2] !== null) out.push({ i, km: km[i] as number, ele: point[2] });
    });
    return out;
  });
  protected readonly totalKm = computed(() => this.points().at(-1)?.km ?? 0);
  private readonly minEle = computed(() => Math.min(...this.points().map((p) => p.ele)));
  private readonly maxEle = computed(() => Math.max(...this.points().map((p) => p.ele)));

  protected readonly linePath = computed(() =>
    this.points()
      .map((p, n) => `${n === 0 ? 'M' : 'L'}${this.x(p.km).toFixed(1)},${this.y(p.ele).toFixed(1)}`)
      .join(''),
  );
  protected readonly areaPath = computed(
    () =>
      `${this.linePath()}L${this.x(this.totalKm()).toFixed(1)},${HEIGHT - PAD.bottom}L${PAD.left},${HEIGHT - PAD.bottom}Z`,
  );
  protected readonly ticks = computed(() => {
    const min = this.minEle();
    const max = this.maxEle();
    const step = Math.max(100, Math.ceil((max - min) / 4 / 100) * 100);
    const out: number[] = [];
    for (let ele = Math.ceil(min / step) * step; ele <= max; ele += step) out.push(ele);
    return out;
  });
  // For screen readers: what the chart shows.
  protected readonly summary = computed(() => {
    const m = this.transloco.translate('trails.units.m');
    const km = this.transloco.translate('trails.units.km');
    return `${this.transloco.translate('trails.detail.profile')}: ${this.minEle()}–${this.maxEle()} ${m}, ${this.totalKm().toFixed(1)} ${km}`;
  });

  protected readonly hovered = signal<ProfilePoint | null>(null);

  protected x(km: number): number {
    return PAD.left + (km / (this.totalKm() || 1)) * (WIDTH - PAD.left - PAD.right);
  }

  protected y(ele: number): number {
    const span = this.maxEle() - this.minEle() || 1;
    return PAD.top + (1 - (ele - this.minEle()) / span) * (HEIGHT - PAD.top - PAD.bottom);
  }

  // Keeps the label inside the chart near the right edge.
  protected labelX(km: number): number {
    return Math.min(this.x(km) + 8, WIDTH - 150);
  }

  protected onMove(event: PointerEvent): void {
    const svg = event.currentTarget as SVGSVGElement;
    const rect = svg.getBoundingClientRect();
    const km =
      ((((event.clientX - rect.left) / rect.width) * WIDTH - PAD.left) /
        (WIDTH - PAD.left - PAD.right)) *
      this.totalKm();
    let best = this.points()[0];
    for (const point of this.points()) {
      if (best && Math.abs(point.km - km) < Math.abs(best.km - km)) best = point;
    }
    this.hovered.set(best ?? null);
    this.hover.emit(best?.i ?? null);
  }

  protected onLeave(): void {
    this.hovered.set(null);
    this.hover.emit(null);
  }
}
