import { readStorage, writeStorage } from '@/core/lib/storage';
import { Injectable, computed, signal } from '@angular/core';

// The visitor's answer to the cookie notice, kept in localStorage:
// 'all' — analytics cookies too, 'necessary' — only those the site needs.
// 'accepted' is the answer to the earlier notice, which only told about
// the necessary cookies: it says nothing about analytics.
export type CookieChoice = 'all' | 'necessary';
export const CONSENT_KEY = 'cookie-consent';
const LEGACY_ACCEPTED = 'accepted';

@Injectable({ providedIn: 'root' })
export class CookieConsentStore {
  // undefined until read (browser only, after the first render).
  private readonly stored = signal<string | null | undefined>(undefined);

  readonly loaded = computed(() => this.stored() !== undefined);
  // The answer about analytics; null — not asked yet (or the old notice).
  readonly choice = computed<CookieChoice | null>(() => {
    const value = this.stored();
    return value === 'all' || value === 'necessary' ? value : null;
  });
  // The visitor has seen some notice (the old one counts).
  readonly answered = computed(() => this.choice() !== null || this.stored() === LEGACY_ACCEPTED);
  readonly analyticsAllowed = computed(() => this.choice() === 'all');

  // Browser only: in afterNextRender.
  init(): void {
    if (!this.loaded()) this.stored.set(readStorage(CONSENT_KEY));
  }

  set(choice: CookieChoice): void {
    writeStorage(CONSENT_KEY, choice);
    this.stored.set(choice);
  }

  // The notice without analytics to ask about: "got it".
  acknowledge(): void {
    writeStorage(CONSENT_KEY, LEGACY_ACCEPTED);
    this.stored.set(LEGACY_ACCEPTED);
  }

  // Asks again (the "cookie settings" link in the footer).
  reopen(): void {
    this.stored.set(null);
  }
}
