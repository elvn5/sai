import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// Stand-ins for a list that is still loading (after navigating in the
// browser; the server sends lists filled in): rows like the tour and blog
// lists, or cards like the hotel and route grids.
@Component({
  selector: 'app-list-skeleton',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block', role: 'status' },
  template: `
    <span class="sr-only">{{ 'a11y.loading' | transloco }}</span>
    @if (variant() === 'rows') {
      <ul class="flex flex-col gap-4" aria-hidden="true">
        @for (item of items(); track item) {
          <li class="ui-card flex flex-col gap-4 p-4 sm:flex-row">
            <div class="ui-skeleton aspect-video w-full shrink-0 sm:w-56"></div>
            <div class="flex flex-1 flex-col justify-center gap-3">
              <div class="flex gap-2">
                <div class="ui-skeleton h-5 w-20 rounded-full"></div>
                <div class="ui-skeleton h-5 w-24 rounded-full"></div>
              </div>
              <div class="ui-skeleton h-6 w-3/4"></div>
              <div class="ui-skeleton h-4 w-full"></div>
              <div class="ui-skeleton h-4 w-2/3"></div>
            </div>
          </li>
        }
      </ul>
    } @else {
      <ul [class]="'grid gap-4 ' + columns()" aria-hidden="true">
        @for (item of items(); track item) {
          <li class="ui-card overflow-hidden">
            <div class="ui-skeleton aspect-[4/3] rounded-none"></div>
            <div class="flex flex-col gap-3 p-4">
              <div class="ui-skeleton h-4 w-1/3"></div>
              <div class="ui-skeleton h-5 w-3/4"></div>
              <div class="ui-skeleton h-4 w-1/2"></div>
            </div>
          </li>
        }
      </ul>
    }
  `,
})
export class ListSkeleton {
  readonly variant = input<'rows' | 'cards'>('cards');
  readonly count = input(3);
  // The grid's column classes, to match the list it stands in for.
  readonly columns = input('grid-cols-1 sm:grid-cols-2 lg:grid-cols-3');

  protected readonly items = computed(() => Array.from({ length: this.count() }, (_, i) => i));
}
