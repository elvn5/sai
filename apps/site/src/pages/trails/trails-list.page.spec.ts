import { TrailsListPage } from '@/pages/trails/trails-list.page';
import { pageTestProviders } from '@/testing/providers';
import { HttpTestingController } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';

describe('route catalog', () => {
  it('sends region, activity, grade, season, search and sort to the API', async () => {
    TestBed.configureTestingModule({ providers: pageTestProviders() });
    const fixture = TestBed.createComponent(TrailsListPage);
    for (const [name, value] of Object.entries({
      area: 'karakol',
      kind: 'trek',
      grade: '1a,1b',
      season: 'summer',
      search: '  test  ',
      sort: 'name',
    })) {
      fixture.componentRef.setInput(name, value);
    }
    fixture.detectChanges();
    const http = TestBed.inject(HttpTestingController);
    const request = http.expectOne((req) => req.url === '/trails');
    expect(request.request.params.get('area')).toBe('karakol');
    expect(request.request.params.get('kind')).toBe('trek');
    expect(request.request.params.get('grade')).toBe('1a,1b');
    expect(request.request.params.get('season')).toBe('summer');
    expect(request.request.params.get('search')).toBe('test');
    expect(request.request.params.get('sort')).toBe('name');
    request.flush([]);
    http.expectOne('/trails/areas').flush({ karakol: 1 });
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('app-state-block')).not.toBeNull();
  });

  it('shows a retryable error rather than an empty catalog on API failure', async () => {
    const response: ResponseInit = {};
    TestBed.configureTestingModule({ providers: pageTestProviders(response) });
    const fixture = TestBed.createComponent(TrailsListPage);
    fixture.detectChanges();
    const http = TestBed.inject(HttpTestingController);
    http.expectOne('/trails').flush(null, { status: 503, statusText: 'Unavailable' });
    http.expectOne('/trails/areas').flush({});
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('app-load-error')).not.toBeNull();
    expect(response.status).toBe(503);
  });
});
