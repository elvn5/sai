import { ChangeDetectionStrategy, Component, input, model } from '@angular/core';
import type { FormValueControl } from '@angular/forms/signals';

export interface IconOption {
  value: string;
  // A static icon class (icon-[mdi--tent]).
  icon: string;
  label: string;
}

// Big icon chips to pick one option (difficulty, season)…
@Component({
  selector: 'app-icon-picker',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="grid max-w-sm grid-cols-[repeat(auto-fit,minmax(5.5rem,1fr))] gap-2" role="group">
      @for (option of options(); track option.value) {
        <button
          type="button"
          class="ui-chip-card"
          [attr.aria-pressed]="value() === option.value"
          (click)="value.set(option.value)"
        >
          <span [class]="option.icon + ' size-6'" aria-hidden="true"></span>
          <span class="text-center leading-tight">{{ option.label }}</span>
        </button>
      }
    </div>
  `,
})
export class IconPicker implements FormValueControl<string> {
  readonly value = model('');
  readonly options = input.required<IconOption[]>();
}

// …or several (equipment).
@Component({
  selector: 'app-icon-multi-picker',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="grid max-w-sm grid-cols-[repeat(auto-fit,minmax(5.5rem,1fr))] gap-2" role="group">
      @for (option of options(); track option.value) {
        <button
          type="button"
          class="ui-chip-card"
          [attr.aria-pressed]="value().includes(option.value)"
          (click)="toggle(option.value)"
        >
          <span [class]="option.icon + ' size-6'" aria-hidden="true"></span>
          <span class="text-center leading-tight">{{ option.label }}</span>
        </button>
      }
    </div>
  `,
})
export class IconMultiPicker implements FormValueControl<string[]> {
  readonly value = model<string[]>([]);
  readonly options = input.required<IconOption[]>();

  protected toggle(option: string): void {
    this.value.update((current) =>
      current.includes(option) ? current.filter((entry) => entry !== option) : [...current, option],
    );
  }
}
