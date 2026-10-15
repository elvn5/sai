import { ChangeDetectionStrategy, Component, input } from '@angular/core';

// One headline number of the cabinet, with its change against the previous
// period (whole percent; null hides it). The change carries an arrow and a
// sign, not just a color.
@Component({
  selector: 'app-stat-tile',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <div class="ui-card flex h-full flex-col gap-1 px-4 py-3">
      <p class="flex items-center gap-2 text-sm text-text-muted">
        <span [class]="icon() + ' size-4 shrink-0'" aria-hidden="true"></span>
        <span class="truncate">{{ label() }}</span>
      </p>
      <p class="text-2xl font-extrabold text-text tabular-nums">{{ value() }}</p>
      <p class="flex flex-wrap items-center gap-x-2 text-xs text-text-muted">
        @if (delta(); as change) {
          <span class="inline-flex items-center gap-0.5 font-semibold" [class]="change >= 0 ? 'text-emerald-800 dark:text-emerald-300' : 'text-red-800 dark:text-red-300'">
            <span [class]="(change >= 0 ? 'icon-[heroicons--arrow-trending-up]' : 'icon-[heroicons--arrow-trending-down]') + ' size-3.5'" aria-hidden="true"></span>
            {{ change >= 0 ? '+' : '' }}{{ change }}%
          </span>
        } @else if (delta() === 0) {
          <span class="font-semibold">0%</span>
        }
        @if (hint()) {
          <span>{{ hint() }}</span>
        }
      </p>
    </div>
  `,
})
export class StatTile {
  readonly label = input.required<string>();
  readonly value = input.required<string>();
  readonly icon = input.required<string>();
  readonly hint = input<string | null>(null);
  readonly delta = input<number | null>(null);
}
