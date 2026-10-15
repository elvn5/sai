import type { ListingStatEvent, ReviewTargetType } from '@/core/api/types';
import { APP_CONFIG } from '@/core/config/app-config';
import { isPlatformBrowser } from '@angular/common';
import {
  Injectable,
  PLATFORM_ID,
  type Signal,
  afterNextRender,
  effect,
  inject,
  signal,
  untracked,
} from '@angular/core';

// Counts listing page views and contact clicks for the owner's partner
// cabinet. Fire-and-forget; keepalive lets a click that leaves for WhatsApp
// or a phone call still reach the server. Browser only.
@Injectable({ providedIn: 'root' })
export class ListingStats {
  private readonly apiBaseUrl = inject(APP_CONFIG).apiBaseUrl;
  private readonly browser = isPlatformBrowser(inject(PLATFORM_ID));

  track(targetType: ReviewTargetType, targetId: string, event: ListingStatEvent): void {
    if (!this.browser) return;
    void fetch(`${this.apiBaseUrl}/listing-stats/events`, {
      method: 'POST',
      credentials: 'include',
      keepalive: true,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ targetType, targetId, event }),
    }).catch(() => undefined);
  }
}

// Counts one view of a listing page, once it's shown in the browser. Call in
// an injection context.
export function trackListingView(
  targetType: ReviewTargetType,
  targetId: Signal<string | undefined>,
): void {
  const stats = inject(ListingStats);
  const shown = signal(false);
  let counted: string | undefined;
  afterNextRender(() => shown.set(true));
  effect(() => {
    const id = targetId();
    if (!shown() || !id || id === counted) return;
    counted = id;
    untracked(() => stats.track(targetType, id, 'view'));
  });
}
