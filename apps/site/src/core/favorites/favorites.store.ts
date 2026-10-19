import { AuthStore } from '@/core/auth/auth.store';
import { injectCurrentUrl } from '@/core/lib/current-url';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { Notify } from '@/core/notify/notify';
import { isPlatformBrowser } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { Injectable, PLATFORM_ID, effect, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

export type FavoriteType = 'tour' | 'hotel' | 'local_guide';

type FavoriteIds = Record<FavoriteType, string[]>;

const key = (type: FavoriteType, id: string): string => `${type}:${id}`;

// The listings the signed-in visitor saved (the hearts on cards). Pages are
// rendered anonymously, so the saved ids load in the browser once the
// session is known, from their own endpoint (/favorites/mine/ids): the
// public lists stay the same for everyone.
@Injectable({ providedIn: 'root' })
export class FavoritesStore {
  private readonly http = inject(HttpClient);
  private readonly auth = inject(AuthStore);
  private readonly notify = inject(Notify);
  private readonly saved = signal<ReadonlySet<string>>(new Set());
  // Bumped after every change, for lists of saved listings to reload.
  readonly version = signal(0);

  constructor() {
    if (!isPlatformBrowser(inject(PLATFORM_ID))) return;
    void this.auth.ensureSession();
    effect(() => {
      if (this.auth.user()) void this.load();
      else this.saved.set(new Set());
    });
  }

  has(type: FavoriteType, id: string): boolean {
    return this.saved().has(key(type, id));
  }

  // Saves or forgets a listing right away, and puts it back if the API
  // refuses. Resolves to whether it's saved now.
  async toggle(type: FavoriteType, id: string): Promise<boolean> {
    const k = key(type, id);
    const save = !this.saved().has(k);
    this.set(k, save);
    try {
      await firstValueFrom(
        save
          ? this.http.put(`/favorites/${type}/${id}`, null)
          : this.http.delete(`/favorites/${type}/${id}`),
      );
      this.version.update((v) => v + 1);
      if (save) this.notify.success({ title: 'favorites.saved' });
      else
        this.notify.info({
          title: 'favorites.removed',
          action: { label: 'favorites.undo', run: () => void this.toggle(type, id) },
        });
      return save;
    } catch {
      this.set(k, !save);
      this.notify.error({ title: 'favorites.failed', text: 'notify.tryAgain' });
      return !save;
    }
  }

  private set(k: string, saved: boolean): void {
    this.saved.update((current) => {
      const next = new Set(current);
      if (saved) next.add(k);
      else next.delete(k);
      return next;
    });
  }

  private async load(): Promise<void> {
    try {
      const ids = await firstValueFrom(this.http.get<FavoriteIds>('/favorites/mine/ids'));
      this.saved.set(
        new Set(
          (Object.keys(ids) as FavoriteType[]).flatMap((type) =>
            (ids[type] ?? []).map((id) => key(type, id)),
          ),
        ),
      );
    } catch {
      // No hearts lit; saving still works.
    }
  }
}

// A heart's click: toggles the listing, or sends a guest to sign in and
// come back here.
export function injectFavoriteToggle(): (type: FavoriteType, id: string) => Promise<void> {
  const store = inject(FavoritesStore);
  const auth = inject(AuthStore);
  const navigate = injectSiteNavigate();
  const url = injectCurrentUrl();
  return async (type, id) => {
    if (!(await auth.ensureSession())) {
      await navigate(`/login?redirect=${encodeURIComponent(url())}`);
      return;
    }
    await store.toggle(type, id);
  };
}
