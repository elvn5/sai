import { readStorage, writeStorage } from '@/core/lib/storage';
import { ChangeDetectionStrategy, Component, afterNextRender, signal } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// Same key as the Nuxt site: one answer covers both.
const CONSENT_KEY = 'cookie-consent';

// The notice about the cookies the site needs (session, settings). Shown in
// the browser only, until the visitor accepts it.
@Component({
  selector: 'app-cookie-consent',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (visible()) {
      <div
        class="fixed inset-x-0 bottom-0 z-50 border-t border-border bg-surface px-4 py-4 shadow-lg"
        role="region"
        aria-live="polite"
        [attr.aria-label]="'cookieConsent.title' | transloco"
      >
        <div class="mx-auto flex max-w-4xl flex-col items-center gap-3 sm:flex-row sm:justify-between">
          <p class="text-sm text-text-muted">{{ 'cookieConsent.message' | transloco }}</p>
          <button
            type="button"
            class="shrink-0 rounded-md bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700"
            (click)="accept()"
          >
            {{ 'cookieConsent.accept' | transloco }}
          </button>
        </div>
      </div>
    }
  `,
})
export class CookieConsent {
  protected readonly visible = signal(false);

  constructor() {
    afterNextRender(() => this.visible.set(!readStorage(CONSENT_KEY)));
  }

  protected accept(): void {
    writeStorage(CONSENT_KEY, 'accepted');
    this.visible.set(false);
  }
}
