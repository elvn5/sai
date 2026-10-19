import type { TourDay } from '@/core/api/types';
import { ChangeDetectionStrategy, Component, computed, input, signal } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// Days shown before "Show all N days".
const FOLDED_DAYS = 3;

// A tour's program as a timeline, one marker per day (the first one
// filled). A long program folds after three days.
@Component({
  selector: 'app-tour-program',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <ol class="relative flex flex-col gap-3.5">
      @for (day of shown(); track $index; let index = $index; let last = $last) {
        <li class="relative flex gap-3" [class.motion-roll-in]="index >= folded">
          @if (!last || hidden() > 0) {
            <span class="absolute top-8 bottom-[-0.875rem] left-[0.9375rem] w-px bg-border" aria-hidden="true"></span>
          }
          <span
            class="relative flex size-[1.875rem] shrink-0 items-center justify-center rounded-full text-[0.8125rem] font-bold"
            [class]="index === 0 ? 'bg-brand-600 text-white' : 'border-2 border-brand-600 bg-bg text-brand-600 dark:border-brand-400 dark:text-brand-300'"
            aria-hidden="true"
          >{{ index + 1 }}</span>
          <div class="min-w-0 pt-1">
            <p class="font-semibold text-text"><span class="sr-only">{{ 'tours.program.day' | transloco: { day: index + 1 } }}: </span>{{ day.title }}</p>
            @if (day.text) {
              <p class="mt-0.5 text-[0.8125rem] whitespace-pre-line text-text-muted">{{ day.text }}</p>
            }
          </div>
        </li>
      }
    </ol>
    @if (days().length > folded) {
      <button type="button" class="ui-btn-ghost mt-2 -ml-2" [attr.aria-expanded]="open()" (click)="open.set(!open())">
        {{ (open() ? 'tours.program.less' : 'tours.program.all') | transloco: { count: days().length } }}
        <span class="icon-[heroicons--chevron-down] size-4 transition-transform duration-200" [class.rotate-180]="open()" aria-hidden="true"></span>
      </button>
    }
  `,
})
export class TourProgram {
  readonly days = input.required<TourDay[]>();
  protected readonly folded = FOLDED_DAYS;
  protected readonly open = signal(false);
  protected readonly shown = computed(() =>
    this.open() ? this.days() : this.days().slice(0, FOLDED_DAYS),
  );
  protected readonly hidden = computed(() => this.days().length - this.shown().length);
}
