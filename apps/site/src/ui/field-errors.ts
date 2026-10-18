import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import type { FieldState } from '@angular/forms/signals';

// A form field's validation messages, once the visitor has touched it (or
// tried to submit).
@Component({
  selector: 'app-field-errors',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (state().touched() && state().invalid()) {
      @for (error of state().errors(); track $index) {
        <p class="ui-error motion-drop-in mt-1" role="alert"><span class="icon-[heroicons--exclamation-circle] size-4 shrink-0" aria-hidden="true"></span>{{ error.message }}</p>
      }
    }
  `,
})
export class FieldErrors {
  readonly state = input.required<FieldState<unknown>>();
}
