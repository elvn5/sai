import type { ListingOption } from '@/core/lib/listing-options';
import { ChangeDetectionStrategy, Component, input, model } from '@angular/core';
import type { FormValueControl } from '@angular/forms/signals';
import { TranslocoPipe } from '@jsverse/transloco';

// Several options as pill toggles (amenities, languages, destinations), in
// filters and forms alike.
@Component({
  selector: 'app-chip-toggle-group',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-wrap gap-2" role="group" [attr.aria-label]="label() || null">
      @for (option of options(); track option.value) {
        <button
          type="button"
          class="ui-pill inline-flex items-center gap-1.5 px-3"
          [attr.aria-pressed]="value().includes(option.value)"
          (click)="toggle(option.value)"
        >
          @if (option.icon) {
            <span [class]="option.icon + ' size-4'" aria-hidden="true"></span>
          }
          {{ option.labelKey | transloco }}
        </button>
      }
    </div>
  `,
})
export class ChipToggleGroup implements FormValueControl<string[]> {
  readonly value = model<string[]>([]);
  readonly options = input.required<ListingOption[]>();
  readonly label = input('');

  protected toggle(option: string): void {
    this.value.update((current) =>
      current.includes(option) ? current.filter((item) => item !== option) : [...current, option],
    );
  }
}
