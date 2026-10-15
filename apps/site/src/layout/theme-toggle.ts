import { ColorModeStore } from '@/core/theme/color-mode';
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// Light / dark mode. The server can't know the visitor's mode, so the
// button shows both icons and CSS picks one (.dark on <html>): no flash.
@Component({
  selector: 'app-theme-toggle',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <button
      type="button"
      class="flex size-9 items-center justify-center rounded-md transition-colors hover:bg-white/10"
      [attr.aria-label]="'theme.toggle' | transloco"
      [title]="'theme.toggle' | transloco"
      (click)="toggle($event)"
    >
      <span class="icon-[heroicons--sun-20-solid] size-5 dark:hidden" aria-hidden="true"></span>
      <span class="icon-[heroicons--moon-20-solid] hidden size-5 dark:block" aria-hidden="true"></span>
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
