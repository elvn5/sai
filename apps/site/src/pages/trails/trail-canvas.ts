import type { TrackPoint, TrailMarker } from '@/core/api/types';
import { cumulativeKm } from '@/core/lib/gpx';
import { projectTrack, steepPoints, validMarkers } from '@/core/lib/trail-geometry';
import {
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  afterRenderEffect,
  computed,
  inject,
  input,
  linkedSignal,
  model,
  viewChild,
} from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

@Component({
  selector: 'app-trail-canvas',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <div class="rounded-card border border-border bg-surface p-4">
      <h3 class="font-semibold">{{ 'trails.editor.title' | transloco }}</h3>
      <p class="my-2 text-sm text-text-muted">{{ (editable() ? 'trails.editor.hint' : 'trails.editor.readHint') | transloco }}</p>
      <canvas #canvas width="1000" height="440" class="w-full rounded-xl bg-slate-950" (click)="select($event)" [attr.aria-label]="'trails.editor.title' | transloco"></canvas>
      <label class="ui-field mt-3">
        {{ 'trails.editor.point' | transloco }} · {{ distance().toFixed(2) }} {{ 'trails.units.km' | transloco }} · {{ track()[selected()]?.[2] ?? '—' }} {{ 'trails.units.m' | transloco }}
        <input type="range" min="0" [max]="track().length - 1" [value]="selected()" (input)="selected.set(+$any($event.target).value)" />
      </label>
      @if (editable()) {
        <div class="mt-3 flex flex-wrap gap-2">
          <button type="button" class="ui-btn-primary" [disabled]="markers().length >= 100" (click)="add()">{{ 'trails.editor.add' | transloco }}</button>
          <button type="button" class="ui-btn-soft" (click)="suggest()">{{ 'trails.editor.suggest' | transloco }}</button>
        </div>
        <p class="mt-2 text-xs text-text-muted">{{ 'trails.editor.threshold' | transloco }}</p>
      }
      <ol class="mt-3 flex flex-col gap-2">
        @for (marker of visibleMarkers(); track $index; let i = $index) {
          <li class="flex items-center gap-2">
            <button type="button" class="ui-btn-soft shrink-0" (click)="selected.set(marker.pointIndex)">{{ i + 1 }} · {{ (distances()[marker.pointIndex] ?? 0).toFixed(2) }} {{ 'trails.units.km' | transloco }}</button>
            @if (editable()) {
              <input class="ui-input min-w-0 flex-1" maxlength="160" [attr.aria-label]="'trails.editor.label' | transloco" [value]="marker.label" (input)="rename(i, $any($event.target).value)" />
              <button type="button" class="ui-btn-ghost" [attr.aria-label]="'trails.editor.remove' | transloco" (click)="remove(i)">×</button>
            } @else { <span class="text-sm">{{ marker.label }}</span> }
          </li>
        }
      </ol>
    </div>
  `,
})
export class TrailCanvas {
  readonly track = input.required<TrackPoint[]>();
  readonly markers = model<TrailMarker[]>([]);
  readonly editable = input(false);
  private readonly i18n = inject(TranslocoService);
  private readonly canvas = viewChild<ElementRef<HTMLCanvasElement>>('canvas');
  protected readonly selected = linkedSignal(() => {
    this.track();
    return 0;
  });
  protected readonly distances = computed(() => cumulativeKm(this.track()));
  protected readonly distance = computed(() => this.distances()[this.selected()] ?? 0);
  protected readonly visibleMarkers = computed(() => validMarkers(this.markers(), this.track()));
  private readonly points = computed(() =>
    projectTrack(this.track()).map(([x, , z]): [number, number] => [500 + x * 38, 220 + z * 38]),
  );

  constructor() {
    afterRenderEffect(() => {
      const canvas = this.canvas()?.nativeElement;
      const points = this.points();
      const markers = this.visibleMarkers();
      const selected = this.selected();
      const ctx = canvas?.getContext('2d');
      if (!ctx) return;
      ctx.clearRect(0, 0, 1000, 440);
      ctx.strokeStyle = '#334155';
      ctx.lineWidth = 1;
      for (let x = 20; x < 1000; x += 40) {
        ctx.beginPath();
        ctx.moveTo(x, 0);
        ctx.lineTo(x, 440);
        ctx.stroke();
      }
      for (let y = 20; y < 440; y += 40) {
        ctx.beginPath();
        ctx.moveTo(0, y);
        ctx.lineTo(1000, y);
        ctx.stroke();
      }
      ctx.beginPath();
      ctx.strokeStyle = '#34d399';
      ctx.lineWidth = 4;
      points.forEach((p, i) => (i ? ctx.lineTo(p[0], p[1]) : ctx.moveTo(p[0], p[1])));
      ctx.stroke();
      markers.forEach((m, i) => {
        const p = points[m.pointIndex];
        if (!p) return;
        ctx.beginPath();
        ctx.fillStyle = '#fbbf24';
        ctx.arc(p[0], p[1], 10, 0, Math.PI * 2);
        ctx.fill();
        ctx.font = 'bold 14px sans-serif';
        ctx.fillText(String(i + 1), p[0] + 13, p[1] - 10);
      });
      const p = points[selected];
      if (p) {
        ctx.beginPath();
        ctx.fillStyle = '#fff';
        ctx.arc(p[0], p[1], 7, 0, Math.PI * 2);
        ctx.fill();
      }
    });
  }
  protected select(event: MouseEvent): void {
    const rect = (event.currentTarget as HTMLCanvasElement).getBoundingClientRect();
    const x = ((event.clientX - rect.left) * 1000) / rect.width;
    const y = ((event.clientY - rect.top) * 440) / rect.height;
    let best = Number.POSITIVE_INFINITY;
    let index = 0;
    this.points().forEach((p, i) => {
      const d = (p[0] - x) ** 2 + (p[1] - y) ** 2;
      if (d < best) {
        best = d;
        index = i;
      }
    });
    this.selected.set(index);
  }
  protected add(): void {
    if (
      this.markers().length >= 100 ||
      this.markers().some((m) => m.pointIndex === this.selected())
    )
      return;
    this.markers.update((items) => [
      ...items,
      { pointIndex: this.selected(), label: this.i18n.translate('trails.editor.defaultLabel') },
    ]);
  }
  protected suggest(): void {
    const existing = this.markers();
    const added = steepPoints(this.track())
      .filter((i) => !existing.some((m) => m.pointIndex === i))
      .map((pointIndex) => ({
        pointIndex,
        label: this.i18n.translate('trails.editor.steepLabel'),
      }));
    this.markers.set([...existing, ...added].slice(0, 100));
  }
  protected rename(index: number, label: string): void {
    this.markers.update((items) => items.map((m, i) => (i === index ? { ...m, label } : m)));
  }
  protected remove(index: number): void {
    this.markers.update((items) => items.filter((_, i) => i !== index));
  }
}
