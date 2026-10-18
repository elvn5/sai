import { injectLang } from '@/core/i18n/current-lang';
import { formatDate } from '@/core/lib/format';
import { ChangeDetectionStrategy, Component, computed, input, signal } from '@angular/core';

export interface DailyPoint {
  // YYYY-MM-DD (a Bishkek calendar day).
  day: string;
  value: number;
}

const WIDTH = 600;
const HEIGHT = 120;
// Surface gap between neighboring bars.
const GAP = 2;

// One measure per day as thin bars on a zero baseline, with a tooltip for
// the hovered day (the whole column is the target). Two measures get two
// charts, never a second axis. Screen readers get the numbers as a table.
@Component({
  selector: 'app-daily-bar-chart',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <figure class="flex flex-col gap-2">
      <figcaption class="flex items-baseline justify-between gap-2">
        <span class="text-sm font-semibold text-text">{{ title() }}</span>
        <span class="text-sm text-text-muted tabular-nums">{{ total() }}</span>
      </figcaption>
      <div class="relative" (pointerleave)="hovered.set(null)">
        <svg [attr.viewBox]="'0 0 ' + width + ' ' + height" preserveAspectRatio="none" class="h-36 w-full" aria-hidden="true">
          <line x1="0" [attr.x2]="width" [attr.y1]="height - 0.5" [attr.y2]="height - 0.5" stroke="var(--color-border)" stroke-width="1" />
          @for (bar of bars(); track bar.day; let index = $index) {
            @if (bar.value > 0) {
              <path
                class="motion-grow-y [transform-box:fill-box]"
                [style.--i]="index"
                [attr.d]="bar.path"
                [attr.fill]="hovered() === index ? 'var(--color-brand-500)' : 'var(--color-brand-200)'"
              />
            }
            <rect [attr.x]="index * step()" y="0" [attr.width]="step()" [attr.height]="height" fill="transparent" (pointerenter)="hovered.set(index)" (pointerdown)="hovered.set(index)" />
          }
        </svg>
        @if (tooltip(); as tip) {
          <div class="pointer-events-none absolute -top-8 -translate-x-1/2 rounded-md bg-slate-900 px-2 py-1 text-xs font-medium whitespace-nowrap text-white shadow-sm dark:bg-slate-100 dark:text-slate-900" [style.left]="tip.left">{{ tip.text }}</div>
        }
      </div>
      @if (points().length > 0) {
        <div class="flex justify-between text-xs text-text-muted" aria-hidden="true">
          <span>{{ label(points()[0]!.day) }}</span>
          <span>{{ label(points()[points().length - 1]!.day) }}</span>
        </div>
      }
      <table class="sr-only">
        <caption>{{ title() }}</caption>
        <tbody>
          @for (point of points(); track point.day) {
            <tr><th scope="row">{{ label(point.day) }}</th><td>{{ point.value }}</td></tr>
          }
        </tbody>
      </table>
    </figure>
  `,
})
export class DailyBarChart {
  readonly title = input.required<string>();
  readonly points = input.required<DailyPoint[]>();

  private readonly lang = injectLang();
  protected readonly width = WIDTH;
  protected readonly height = HEIGHT;
  protected readonly hovered = signal<number | null>(null);

  protected readonly total = computed(() => this.points().reduce((sum, p) => sum + p.value, 0));
  protected readonly step = computed(() => WIDTH / Math.max(1, this.points().length));
  // Bars with a rounded top on a square base (the baseline).
  protected readonly bars = computed(() => {
    const max = Math.max(1, ...this.points().map((p) => p.value));
    const step = this.step();
    const width = Math.max(1, step - GAP);
    return this.points().map((point, index) => {
      const height = point.value === 0 ? 0 : Math.max(3, (point.value / max) * (HEIGHT - 4));
      const x = index * step + GAP / 2;
      const y = HEIGHT - height;
      const r = Math.min(4, width / 2, height);
      return {
        day: point.day,
        value: point.value,
        path: `M${x},${HEIGHT}V${y + r}Q${x},${y} ${x + r},${y}H${x + width - r}Q${x + width},${y} ${x + width},${y + r}V${HEIGHT}Z`,
      };
    });
  });
  protected readonly tooltip = computed(() => {
    const index = this.hovered();
    const point = index === null ? undefined : this.points()[index];
    if (index === null || !point) return null;
    return {
      text: `${this.label(point.day)}: ${point.value}`,
      left: `${((index + 0.5) / this.points().length) * 100}%`,
    };
  });

  protected label(day: string): string {
    return formatDate(this.lang(), day, { day: 'numeric', month: 'short' });
  }
}
