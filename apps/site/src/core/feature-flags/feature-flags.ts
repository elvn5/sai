import { readStorage, writeStorage } from '@/core/lib/storage';
import { isPlatformBrowser } from '@angular/common';
import { Injectable, PLATFORM_ID, afterNextRender, computed, inject, signal } from '@angular/core';

// Parts of the site the developer menu (layout/dev-menu.ts) can switch off
// in this browser, to check a page without them. Everything is on unless
// switched off here.
export const FEATURE_FLAGS = [
  'favorites',
  'tourAlerts',
  'tourDates',
  'showMore',
  'tourGallery',
  'tourProgram',
  'placesLeft',
  'passwordReset',
  'viewTransitions',
] as const;

export type FeatureFlag = (typeof FEATURE_FLAGS)[number];

export type FlagOverrides = Partial<Record<FeatureFlag, boolean>>;

export const FEATURE_FLAGS_KEY = 'feature-flags';

// Known flags with a boolean value; anything else in storage is dropped.
export function parseOverrides(raw: string | null): FlagOverrides {
  if (!raw) return {};
  try {
    const value: unknown = JSON.parse(raw);
    if (!value || typeof value !== 'object') return {};
    const record = value as Record<string, unknown>;
    return Object.fromEntries(
      FEATURE_FLAGS.flatMap((flag) =>
        typeof record[flag] === 'boolean' ? [[flag, record[flag]]] : [],
      ),
    );
  } catch {
    return {};
  }
}

// The switches, kept in localStorage. Pages render on the server with
// everything on, so the saved switches apply after the first render in the
// browser: the hydrated HTML matches the server's.
@Injectable({ providedIn: 'root' })
export class FeatureFlags {
  private readonly browser = isPlatformBrowser(inject(PLATFORM_ID));
  private readonly overrides = signal<FlagOverrides>({});
  // How many flags differ from the default.
  readonly changedCount = computed(() => Object.keys(this.overrides()).length);

  constructor() {
    if (!this.browser) return;
    afterNextRender(() => this.overrides.set(parseOverrides(readStorage(FEATURE_FLAGS_KEY))));
  }

  isOn(flag: FeatureFlag): boolean {
    return this.overrides()[flag] ?? true;
  }

  set(flag: FeatureFlag, on: boolean): void {
    this.overrides.update((current) => {
      const { [flag]: _, ...rest } = current;
      return on ? rest : { ...rest, [flag]: false };
    });
    this.save();
  }

  reset(): void {
    this.overrides.set({});
    this.save();
  }

  private save(): void {
    if (this.browser) writeStorage(FEATURE_FLAGS_KEY, JSON.stringify(this.overrides()));
  }
}
