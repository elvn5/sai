import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';

const SIZES = { sm: 'size-4', md: 'size-5', lg: 'size-7' } as const;

// A rating as five stars, rounded to whole stars; read as "4.6 / 5".
@Component({
  selector: 'app-rating-stars',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'inline-flex items-center', role: 'img', '[attr.aria-label]': 'label()' },
  template: `
    @for (star of stars; track star) {
      <span
        class="icon-[heroicons--star-solid]"
        [class]="sizeClass() + (star <= filled() ? ' text-amber-400' : ' text-slate-300 dark:text-slate-600')"
        aria-hidden="true"
      ></span>
    }
  `,
})
export class RatingStars {
  readonly value = input.required<number>();
  readonly size = input<keyof typeof SIZES>('sm');
  protected readonly stars = [1, 2, 3, 4, 5];
  protected readonly filled = computed(() => Math.round(this.value()));
  protected readonly sizeClass = computed(() => SIZES[this.size()]);
  protected readonly label = computed(() => `${this.value().toFixed(1)} / 5`);
}
