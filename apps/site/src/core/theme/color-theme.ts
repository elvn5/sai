import { readStorage, writeStorage } from '@/core/lib/storage';
import { DOCUMENT, Injectable, inject, signal } from '@angular/core';

export type ColorTheme = 'red' | 'azure';

// Same key as the Nuxt site.
export const COLOR_THEME_KEY = 'color-theme';

const FAVICONS: Record<ColorTheme, string> = { red: '/logo.svg', azure: '/logo-azure.svg' };

// The brand palette: red (default) or azure ([data-color-theme="azure"] on
// <html>, set before the first paint by the script in index.html).
@Injectable({ providedIn: 'root' })
export class ColorThemeStore {
  private readonly document = inject(DOCUMENT);
  private readonly themeSignal = signal<ColorTheme>('red');
  readonly theme = this.themeSignal.asReadonly();

  // Browser only (afterNextRender).
  init(): void {
    this.themeSignal.set(readStorage(COLOR_THEME_KEY) === 'azure' ? 'azure' : 'red');
  }

  toggle(): void {
    const next: ColorTheme = this.themeSignal() === 'red' ? 'azure' : 'red';
    writeStorage(COLOR_THEME_KEY, next);
    const root = this.document.documentElement;
    if (next === 'azure') root.setAttribute('data-color-theme', 'azure');
    else root.removeAttribute('data-color-theme');
    this.document.getElementById('favicon')?.setAttribute('href', FAVICONS[next]);
    this.themeSignal.set(next);
  }
}
