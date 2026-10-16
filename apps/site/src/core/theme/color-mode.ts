import { readStorage, writeStorage } from '@/core/lib/storage';
import { DOCUMENT, Injectable, inject, signal } from '@angular/core';

export type ColorMode = 'light' | 'dark';

// The storage key of the color mode, as the former Nuxt site named it:
// visitors' saved choice keeps working.
export const COLOR_MODE_KEY = 'nuxt-color-mode';

// Light or dark mode: the `.dark` class on <html>. An inline script in
// index.html sets it before the first paint (no flash) from the saved
// choice or the system setting; this store takes over in the browser.
@Injectable({ providedIn: 'root' })
export class ColorModeStore {
  private readonly root = inject(DOCUMENT).documentElement;
  private readonly modeSignal = signal<ColorMode>('light');
  readonly mode = this.modeSignal.asReadonly();

  // Browser only (afterNextRender).
  init(): void {
    this.modeSignal.set(this.root.classList.contains('dark') ? 'dark' : 'light');
    // Follow the system until the visitor picks a mode.
    const saved = readStorage(COLOR_MODE_KEY);
    if (saved !== 'light' && saved !== 'dark') {
      matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (event) => {
        if (readStorage(COLOR_MODE_KEY) === 'light' || readStorage(COLOR_MODE_KEY) === 'dark')
          return;
        this.apply(event.matches ? 'dark' : 'light');
      });
    }
  }

  // Switches mode, revealing the new one in a circle growing from the
  // pointer where the browser supports view transitions.
  toggle(origin?: { x: number; y: number }): void {
    const next: ColorMode = this.modeSignal() === 'dark' ? 'light' : 'dark';
    writeStorage(COLOR_MODE_KEY, next);
    const document = this.root.ownerDocument;
    const reduceMotion = matchMedia('(prefers-reduced-motion: reduce)').matches;
    if (!origin || reduceMotion || !document.startViewTransition) {
      this.apply(next);
      return;
    }
    const { x, y } = origin;
    const radius = Math.hypot(Math.max(x, innerWidth - x), Math.max(y, innerHeight - y));
    document
      .startViewTransition(() => this.apply(next))
      .ready.then(() => {
        this.root.animate(
          { clipPath: [`circle(0px at ${x}px ${y}px)`, `circle(${radius}px at ${x}px ${y}px)`] },
          {
            duration: 500,
            easing: 'cubic-bezier(0.47, 0, 0.745, 0.715)',
            pseudoElement: '::view-transition-new(root)',
          },
        );
      });
  }

  private apply(mode: ColorMode): void {
    this.root.classList.toggle('dark', mode === 'dark');
    this.modeSignal.set(mode);
  }
}
