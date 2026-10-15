import { ChangeDetectionStrategy, Component, input, model, signal } from '@angular/core';
import type { FormValueControl } from '@angular/forms/signals';

// Picks 1–5 stars: a radio group (arrow keys move between stars).
@Component({
  selector: 'app-rating-input',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="inline-flex items-center gap-1" role="radiogroup" [attr.aria-label]="label()" (mouseleave)="hovered.set(0)">
      @for (star of stars; track star) {
        <button
          type="button"
          role="radio"
          class="rounded transition-transform hover:scale-110"
          [attr.aria-checked]="value() === star"
          [attr.aria-label]="star + ' / 5'"
          [tabindex]="value() === star || (value() === 0 && star === 1) ? 0 : -1"
          (mouseenter)="hovered.set(star)"
          (click)="value.set(star)"
          (keydown.arrowright)="move($event, 1)"
          (keydown.arrowup)="move($event, 1)"
          (keydown.arrowleft)="move($event, -1)"
          (keydown.arrowdown)="move($event, -1)"
        >
          <span
            class="icon-[heroicons--star-solid] size-8"
            [class]="star <= (hovered() || value()) ? 'text-amber-400' : 'text-slate-300 dark:text-slate-600'"
            aria-hidden="true"
          ></span>
        </button>
      }
    </div>
  `,
})
export class RatingInput implements FormValueControl<number> {
  readonly value = model(0);
  readonly label = input('');
  protected readonly stars = [1, 2, 3, 4, 5];
  protected readonly hovered = signal(0);

  protected move(event: Event, delta: number): void {
    event.preventDefault();
    const next = Math.min(5, Math.max(1, (this.value() || 0) + delta));
    this.value.set(next);
    const group = (event.target as HTMLElement).parentElement;
    (group?.children[next - 1] as HTMLElement | undefined)?.focus();
  }
}
