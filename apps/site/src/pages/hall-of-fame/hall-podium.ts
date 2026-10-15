import type { HallEntry } from '@/core/api/types';
import { personName } from '@/core/lib/format';
import { PersonAvatar } from '@/ui/person-avatar';
import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

const HEIGHTS: Record<number, string> = { 1: 'h-36 sm:h-44', 2: 'h-28 sm:h-32', 3: 'h-20 sm:h-24' };
const COLORS: Record<number, string> = {
  1: 'from-amber-300 to-amber-600',
  2: 'from-slate-200 to-slate-500',
  3: 'from-orange-300 to-orange-700',
};
const DELAYS = ['0.3s', '0s', '0.6s'];

// The top three, rising onto a podium: 2nd · 1st · 3rd.
@Component({
  selector: 'app-hall-podium',
  imports: [PersonAvatar, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  styles: `
    .step { animation: rise 0.9s cubic-bezier(0.34, 1.56, 0.64, 1) both; }
    .crown { animation: bob 2.4s ease-in-out infinite; }
    @keyframes rise { from { opacity: 0; transform: translateY(60px) scale(0.9); } }
    @keyframes bob { 50% { transform: translate(-50%, -4px) rotate(-6deg); } }
    @media (prefers-reduced-motion: reduce) { .step, .crown { animation: none; } }
  `,
  template: `
    <ol class="flex items-end justify-center gap-3 sm:gap-6">
      @for (place of places(); track place.entry.user.id) {
        <li class="step flex w-24 flex-col items-center sm:w-36" [style.animation-delay]="place.delay" [style.order]="place.order">
          <div class="relative mb-2">
            @if (place.entry.rank === 1) {
              <span class="crown icon-[mdi--crown] absolute -top-7 left-1/2 size-8 -translate-x-1/2 text-amber-400 drop-shadow" aria-hidden="true"></span>
            }
            <app-person-avatar
              [person]="place.entry.user"
              [size]="place.entry.rank === 1 ? 80 : 56"
              [class]="place.entry.rank === 1 ? 'ring-4 ring-amber-400' : 'ring-2 ring-white/40'"
            />
          </div>
          <p class="w-full truncate text-center text-sm font-semibold text-text">{{ name(place.entry) }}</p>
          <p class="text-xs font-medium text-text-muted">{{ 'hallOfFame.points' | transloco: { count: place.entry.points } }}</p>
          <div
            class="mt-2 flex w-full items-start justify-center rounded-t-xl bg-linear-to-b pt-3 text-3xl font-black text-white shadow-lg"
            [class]="heights[place.entry.rank] + ' ' + colors[place.entry.rank]"
            aria-hidden="true"
          >
            {{ place.entry.rank }}
          </div>
        </li>
      }
    </ol>
  `,
})
export class HallPodium {
  readonly leaders = input.required<HallEntry[]>();
  protected readonly heights = HEIGHTS;
  protected readonly colors = COLORS;

  // In rank order for screen readers; CSS order lays them out 2 · 1 · 3.
  protected readonly places = computed(() =>
    this.leaders()
      .slice(0, 3)
      .map((entry, index) => ({
        entry,
        order: [1, 0, 2][index] ?? index,
        delay: DELAYS[[1, 0, 2][index] ?? 0],
      })),
  );

  protected name(entry: HallEntry): string {
    return personName(entry.user);
  }
}
