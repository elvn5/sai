import { ColorThemeStore } from '@/core/theme/color-theme';
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// Red / azure brand palette. The dot shows the current brand color.
@Component({
  selector: 'app-color-theme-toggle',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <button
      type="button"
      class="flex size-9 items-center justify-center rounded-md transition-colors hover:bg-white/10"
      [attr.aria-label]="'colorTheme.toggle' | transloco"
      [title]="'colorTheme.toggle' | transloco"
      (click)="colorTheme.toggle()"
    >
      <span class="block size-4 rounded-full border border-current bg-brand-400" aria-hidden="true"></span>
    </button>
  `,
})
export class ColorThemeToggle {
  protected readonly colorTheme = inject(ColorThemeStore);
}
