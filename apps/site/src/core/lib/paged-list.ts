import type { Paginated } from '@/core/api/types';
import { loaded } from '@/core/lib/resource';
import { HttpClient, type HttpResourceRef, httpResource } from '@angular/common/http';
import { type Signal, computed, inject, linkedSignal, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

export interface PagedListOptions {
  // The API list ("/hotels"), answering { items, total } to limit/offset.
  url: string;
  pageSize: number;
  // The page the URL asks for (?page=N, 1-based): rendered on the server.
  page: Signal<number>;
  // The API's filters, without limit and offset.
  params: Signal<Record<string, string>>;
}

export interface PagedList<T> {
  // The requested page, rendered on the server.
  readonly first: HttpResourceRef<Paginated<T> | undefined>;
  // That page plus the ones loaded in place after it.
  readonly items: Signal<T[]>;
  readonly total: Signal<number>;
  readonly hasMore: Signal<boolean>;
  // The page right after the loaded ones (for the crawlable next link).
  readonly nextPage: Signal<number>;
  readonly loadingMore: Signal<boolean>;
  loadMore(): Promise<void>;
  reload(): void;
}

// A catalog page: the server renders page N of a paginated API list, and the
// browser appends the following pages in place ("show more"). Call in an
// injection context.
export function pagedList<T>(options: PagedListOptions): PagedList<T> {
  const http = inject(HttpClient);
  const { url, pageSize, page, params } = options;
  const offsetOf = (loaded: number): number => (page() - 1) * pageSize + loaded;

  const first = httpResource<Paginated<T>>(() => ({
    url,
    params: { ...params(), limit: pageSize, offset: offsetOf(0) },
  }));
  // Pages loaded in place; reset with the query.
  const more = linkedSignal<unknown, T[]>({
    source: () => [page(), params()],
    computation: () => [],
  });
  const loadingMore = signal(false);
  const items = computed(() => [...(loaded(first)?.items ?? []), ...more()]);
  const total = computed(() => loaded(first)?.total ?? 0);
  const hasMore = computed(() => first.hasValue() && offsetOf(items().length) < total());
  const nextPage = computed(() => page() + Math.max(1, Math.ceil(items().length / pageSize)));

  return {
    first,
    items,
    total,
    hasMore,
    nextPage,
    loadingMore: loadingMore.asReadonly(),
    async loadMore(): Promise<void> {
      if (loadingMore() || !hasMore()) return;
      loadingMore.set(true);
      try {
        const next = await firstValueFrom(
          http.get<Paginated<T>>(url, {
            params: { ...params(), limit: pageSize, offset: offsetOf(items().length) },
          }),
        );
        more.update((list) => [...list, ...next.items]);
      } finally {
        loadingMore.set(false);
      }
    },
    reload(): void {
      more.set([]);
      first.reload();
    },
  };
}
