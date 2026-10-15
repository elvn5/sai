import type { Paginated } from '@/core/api/types';
import { pagedList } from '@/core/lib/paged-list';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ApplicationRef, Injector, runInInjectionContext, signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';

function setup() {
  TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] });
  const page = signal(2);
  const params = signal<Record<string, string>>({ destination: 'karakol' });
  const list = runInInjectionContext(TestBed.inject(Injector), () =>
    pagedList<{ id: number }>({ url: '/hotels', pageSize: 2, page, params }),
  );
  return {
    list,
    page,
    http: TestBed.inject(HttpTestingController),
    app: TestBed.inject(ApplicationRef),
  };
}

const items = (...ids: number[]): Paginated<{ id: number }> => ({
  items: ids.map((id) => ({ id })),
  total: 7,
});

describe('pagedList', () => {
  it('renders page N and appends the next pages in place', async () => {
    const { list, http, app } = setup();
    TestBed.tick();
    http.expectOne('/hotels?destination=karakol&limit=2&offset=2').flush(items(3, 4));
    await app.whenStable();
    expect(list.items().map((item) => item.id)).toEqual([3, 4]);
    expect(list.hasMore()).toBe(true);
    expect(list.nextPage()).toBe(3);

    const more = list.loadMore();
    http.expectOne('/hotels?destination=karakol&limit=2&offset=4').flush(items(5, 6));
    await more;
    expect(list.items().map((item) => item.id)).toEqual([3, 4, 5, 6]);
    expect(list.nextPage()).toBe(4);
  });

  it('stays readable when the API fails', async () => {
    const { list, http, app } = setup();
    TestBed.tick();
    http.expectOne(() => true).flush(null, { status: 502, statusText: 'Bad Gateway' });
    await app.whenStable();
    expect(list.first.error()).toBeTruthy();
    expect(list.items()).toEqual([]);
    expect(list.total()).toBe(0);
    expect(list.hasMore()).toBe(false);
  });
});
