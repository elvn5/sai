import { ChangeDetectionStrategy, Component, input, model } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// The rules and risks text participants accept (the API's WaiverVersion):
// its points are translated under waiver.v1.
export const WAIVER_VERSION = 'v1';
const WAIVER_POINTS = ['health', 'guide', 'gear', 'weather', 'risk', 'insurance'];

// The rules and risks of a trip, with the box a participant ticks to
// accept them (when signing up, or later for a sign-up from the bot).
@Component({
  selector: 'app-waiver-accept',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <div class="rounded-lg border border-border bg-text/3 p-3">
      <p class="font-semibold text-text">{{ 'waiver.v1.title' | transloco }}</p>
      <ul class="mt-2 flex list-disc flex-col gap-1 pl-5 text-sm text-text-muted">
        @for (point of points; track point) {
          <li>{{ 'waiver.v1.' + point | transloco }}</li>
        }
      </ul>
      <label class="mt-3 flex cursor-pointer items-start gap-2 text-sm font-medium text-text">
        <input type="checkbox" class="mt-0.5" [checked]="accepted()" (change)="accepted.set($any($event.target).checked)" />
        {{ 'waiver.v1.accept' | transloco }}
      </label>
      @if (missing() && !accepted()) {
        <p class="ui-error mt-1" role="alert">{{ 'participantForm.errors.waiver' | transloco }}</p>
      }
    </div>
  `,
})
export class WaiverAccept {
  readonly accepted = model(false);
  // Show that accepting is needed (after a try without it).
  readonly missing = input(false);
  protected readonly points = WAIVER_POINTS;
}
