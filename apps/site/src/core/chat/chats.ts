import type { ChatThread } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { DOCUMENT, isPlatformBrowser } from '@angular/common';
import { type HttpClient, httpResource } from '@angular/common/http';
import { DestroyRef, Injectable, PLATFORM_ID, computed, inject } from '@angular/core';
import { firstValueFrom } from 'rxjs';

// How often the unread count is checked while the page is visible.
const UNREAD_EVERY_MS = 30_000;

// Starts (or finds) a chat: a partner with one of their clients (userId),
// a traveler with an organizer (partnerId).
export function openChat(
  http: HttpClient,
  peer: { userId: string } | { partnerId: string },
): Promise<ChatThread> {
  return firstValueFrom(http.post<ChatThread>('/chats', peer));
}

// Calls fn every `ms` while the page is visible (and once when it comes
// back), until the caller is destroyed. Browser only.
export function whileVisible(ms: number, fn: () => void): void {
  if (!isPlatformBrowser(inject(PLATFORM_ID))) return;
  const document = inject(DOCUMENT);
  const timer = setInterval(() => {
    if (document.visibilityState === 'visible') fn();
  }, ms);
  const onVisible = (): void => {
    if (document.visibilityState === 'visible') fn();
  };
  document.addEventListener('visibilitychange', onVisible);
  inject(DestroyRef).onDestroy(() => {
    clearInterval(timer);
    document.removeEventListener('visibilitychange', onVisible);
  });
}

// How many of the signed-in visitor's chats have unread messages: the
// badge on «Сообщения». Read in the browser, again every 30 s.
@Injectable({ providedIn: 'root' })
export class ChatUnread {
  private readonly auth = inject(AuthStore);
  private readonly unread = httpResource<{ chats: number }>(() =>
    this.auth.user() ? '/chats/unread' : undefined,
  );

  readonly count = computed(() => (this.unread.hasValue() ? this.unread.value().chats : 0));

  constructor() {
    whileVisible(UNREAD_EVERY_MS, () => this.refresh());
  }

  refresh(): void {
    if (this.auth.user()) this.unread.reload();
  }
}
