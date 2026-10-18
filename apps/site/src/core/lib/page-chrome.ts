import { DestroyRef, Injectable, inject, signal } from '@angular/core';

// What a page asks of the frame around it: the home page lets its hero run
// under a see-through header on phones; a page with its own bar at the
// bottom (a tour's "Book") hides the bottom navigation; a detail page with
// its own back button over the photo (immersive) hides the header on phones.
@Injectable({ providedIn: 'root' })
export class PageChrome {
  readonly overlayHeader = signal(false);
  readonly ownBottomBar = signal(false);
  readonly immersive = signal(false);
}

// Sets one of the page's wishes while the calling component lives.
export function usePageChrome(wish: 'overlayHeader' | 'ownBottomBar' | 'immersive'): void {
  const chrome = inject(PageChrome);
  chrome[wish].set(true);
  inject(DestroyRef).onDestroy(() => chrome[wish].set(false));
}
