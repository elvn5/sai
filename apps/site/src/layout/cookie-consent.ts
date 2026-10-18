import { readStorage, writeStorage } from '@/core/lib/storage';
import { ChangeDetectionStrategy, Component, afterNextRender, signal } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

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
        class="fixed inset-x-3 bottom-[calc(4.75rem+env(safe-area-inset-bottom))] z-50 flex max-w-md items-center gap-3 rounded-xl border border-border bg-surface px-3.5 py-3 text-sm text-text shadow-[0_12px_28px_-8px_rgb(15_23_42/0.25)] sm:right-auto sm:left-4 lg:bottom-4"
        role="region"
        aria-live="polite"
        [attr.aria-label]="'cookieConsent.title' | transloco"
        animate.enter="motion-toast-in"
        animate.leave="motion-toast-out"
      >
        <span class="icon-[heroicons--information-circle] size-5 shrink-0 text-brand-600 dark:text-brand-300" aria-hidden="true"></span>
        <p class="text-[0.8rem] text-text-muted">{{ 'cookieConsent.message' | transloco }}</p>
        <button type="button" class="ui-btn-primary shrink-0" (click)="accept()">
          {{ 'cookieConsent.accept' | transloco }}
        </button>
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
