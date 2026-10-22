import { Analytics } from '@/core/analytics/analytics';
import { CookieConsentStore } from '@/core/analytics/consent';
import { LEGAL } from '@/core/legal/documents';
import { SiteLink } from '@/ui/site-link';
import {
  ChangeDetectionStrategy,
  Component,
  afterNextRender,
  computed,
  inject,
} from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';

// The cookie notice, in the browser only. With analytics on the site it
// asks whether analytics cookies are allowed (nothing is sent before a
// yes); without it, it only tells about the cookies the site needs.
@Component({
  selector: 'app-cookie-consent',
  imports: [SiteLink, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (visible()) {
      <div
        class="fixed inset-x-3 bottom-[calc(4.75rem+env(safe-area-inset-bottom))] z-50 flex max-w-md flex-wrap items-center gap-3 rounded-xl border border-border bg-surface px-3.5 py-3 text-sm text-text shadow-[0_12px_28px_-8px_rgb(15_23_42/0.25)] sm:right-auto sm:left-4 lg:bottom-4"
        role="region"
        aria-live="polite"
        [attr.aria-label]="'cookieConsent.title' | transloco"
        animate.enter="motion-toast-in"
        animate.leave="motion-toast-out"
      >
        <span class="icon-[heroicons--information-circle] size-5 shrink-0 text-brand-600 dark:text-brand-300" aria-hidden="true"></span>
        @if (asks) {
          <p class="min-w-0 flex-1 text-[0.8rem] text-text-muted">
            {{ 'cookieConsent.analyticsMessage' | transloco }}
            <a [appSiteLink]="privacyPath" class="underline">{{ 'cookieConsent.privacy' | transloco }}</a>
          </p>
          <div class="flex w-full justify-end gap-2">
            <button type="button" class="ui-btn-ghost shrink-0" (click)="consent.set('necessary')">
              {{ 'cookieConsent.necessaryOnly' | transloco }}
            </button>
            <button type="button" class="ui-btn-primary shrink-0" (click)="consent.set('all')">
              {{ 'cookieConsent.acceptAll' | transloco }}
            </button>
          </div>
        } @else {
          <p class="min-w-0 flex-1 text-[0.8rem] text-text-muted">{{ 'cookieConsent.message' | transloco }}</p>
          <button type="button" class="ui-btn-primary shrink-0" (click)="consent.acknowledge()">
            {{ 'cookieConsent.accept' | transloco }}
          </button>
        }
      </div>
    }
  `,
})
export class CookieConsent {
  protected readonly consent = inject(CookieConsentStore);
  protected readonly asks = inject(Analytics).available;
  protected readonly privacyPath = LEGAL.privacy.path;
  protected readonly visible = computed(
    () =>
      this.consent.loaded() &&
      (this.asks ? this.consent.choice() === null : !this.consent.answered()),
  );

  constructor() {
    afterNextRender(() => this.consent.init());
  }
}
