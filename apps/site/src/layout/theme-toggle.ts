import { ColorModeStore } from '@/core/theme/color-mode';
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// Light / dark mode. The server can't know the visitor's mode, so the
// button holds both icons and CSS shows one (.dark on <html>): no flash.
// Switching, the sun turns away as the moon turns in.
@Component({
  selector: 'app-theme-toggle',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <button
      type="button"
      class="ui-icon-btn"
      [attr.aria-label]="'theme.toggle' | transloco"
      [title]="'theme.toggle' | transloco"
      (click)="toggle($event)"
    >
      <span class="icon-swap" aria-hidden="true">
        <span class="icon-[heroicons--sun] size-[1.2rem]" data-dark-off></span>
        <span class="icon-[heroicons--moon] size-[1.2rem]" data-light-off></span>
      </span>
    </button>
  `,
})
export class ThemeToggle {
  private readonly colorMode = inject(ColorModeStore);

  protected toggle(event: MouseEvent): void {
    // Keyboard presses report 0,0: no circle from the corner.
    const origin = event.detail > 0 ? { x: event.clientX, y: event.clientY } : undefined;
    this.colorMode.toggle(origin);
  }
}
