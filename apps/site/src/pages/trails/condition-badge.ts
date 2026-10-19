import type { ConditionSummary } from '@/core/api/types';
import { FRESH_REPORT_DAYS, TRAIL_STATES, daysSince, optionIcon } from '@/core/lib/trail-options';
import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// How fresh the latest condition report is, and what it said. The age is
// counted where it renders: cached HTML may be a few minutes old, which a
// day count doesn't notice.
@Component({
  selector: 'app-condition-badge',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (summary(); as report) {
      <span class="inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium" [class]="fresh() ? 'bg-emerald-500/15 text-emerald-800 dark:text-emerald-200' : 'bg-text/10 text-text'">
        <span class="relative flex size-2" aria-hidden="true">
          @if (fresh()) {
            <span class="absolute inline-flex size-full animate-ping rounded-full bg-emerald-400 opacity-60 motion-reduce:hidden"></span>
          }
          <span class="relative inline-flex size-2 rounded-full" [class]="fresh() ? 'bg-emerald-500' : 'bg-text-muted'"></span>
        </span>
        <span [class]="stateIcon() + ' size-3.5'" aria-hidden="true"></span>
        {{ 'trails.state.' + report.trailState | transloco }} ·
        @if (age() === 0) {
          {{ 'trails.conditions.today' | transloco }}
        } @else {
          {{ 'trails.conditions.daysAgo' | transloco: { count: age() } }}
        }
      </span>
    } @else {
      <span class="inline-flex items-center gap-1 rounded-full bg-text/5 px-2.5 py-1 text-xs text-text-muted">
        <span class="icon-[heroicons--question-mark-circle] size-3.5" aria-hidden="true"></span>{{ 'trails.conditions.none' | transloco }}
      </span>
    }
  `,
})
export class ConditionBadge {
  readonly summary = input.required<ConditionSummary | null>();

  protected readonly age = computed(() => {
    const summary = this.summary();
    return summary ? daysSince(summary.observedOn) : 0;
  });
  protected readonly fresh = computed(
    () => Boolean(this.summary()) && this.age() <= FRESH_REPORT_DAYS,
  );
  protected readonly stateIcon = computed(() =>
    optionIcon(TRAIL_STATES, this.summary()?.trailState ?? ''),
  );
}
