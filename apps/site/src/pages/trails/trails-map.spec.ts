import type { Trail } from '@/core/api/types';
import { LEAFLET_LOADER, type loadLeaflet } from '@/core/lib/leaflet';
import { TrailsMap } from '@/pages/trails/trails-map';
import { pageTestProviders } from '@/testing/providers';
import { TestBed } from '@angular/core/testing';
import { TranslocoService } from '@jsverse/transloco';

describe('catalog map', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('updates markers with filters and uses safe, localized links', async () => {
    const links: HTMLAnchorElement[] = [];
    const map = {
      setView: () => map,
      fitBounds: vi.fn(),
      invalidateSize: vi.fn(),
      remove: vi.fn(),
    };
    const layers = {
      addTo: () => layers,
      clearLayers: () => {
        links.length = 0;
      },
    };
    const tile = { on: () => tile, addTo: () => tile };
    const leaflet = {
      map: () => map,
      layerGroup: () => layers,
      tileLayer: () => tile,
      latLngBounds: (points: unknown) => points,
      divIcon: (options: unknown) => options,
      marker: () => ({
        bindPopup: (link: HTMLAnchorElement) => {
          links.push(link);
          return { addTo: () => undefined };
        },
      }),
    };
    vi.stubGlobal(
      'ResizeObserver',
      class {
        observe(): void {}
        disconnect(): void {}
      },
    );
    TestBed.configureTestingModule({
      providers: [
        ...pageTestProviders(),
        {
          provide: LEAFLET_LOADER,
          useValue: async () => leaflet as unknown as Awaited<ReturnType<typeof loadLeaflet>>,
        },
      ],
    });
    TestBed.inject(TranslocoService).setActiveLang('ky');
    const fixture = TestBed.createComponent(TrailsMap);
    const located = {
      id: 'one',
      name: '<img src=x onerror=alert(1)>',
      slug: 'own-route',
      startLatitude: 42,
      startLongitude: 74,
    } as Trail;
    fixture.componentRef.setInput('trails', [
      located,
      { ...located, id: 'two', startLatitude: null },
    ]);
    fixture.detectChanges();
    TestBed.tick();
    await Promise.resolve();
    TestBed.tick();
    await fixture.whenStable();
    expect(links).toHaveLength(1);
    expect(links[0]?.getAttribute('href')).toBe('/ky/trails/own-route');
    expect(links[0]?.textContent).toBe(located.name);
    expect(links[0]?.querySelector('img')).toBeNull();
    fixture.componentRef.setInput('trails', []);
    fixture.detectChanges();
    TestBed.tick();
    await Promise.resolve();
    TestBed.tick();
    await fixture.whenStable();
    expect(links).toHaveLength(0);
    fixture.destroy();
    expect(map.remove).toHaveBeenCalledOnce();
  });

  it('shows a fallback if the map library cannot load', async () => {
    TestBed.configureTestingModule({
      providers: [
        ...pageTestProviders(),
        {
          provide: LEAFLET_LOADER,
          useValue: async () => {
            throw new Error('offline');
          },
        },
      ],
    });
    const fixture = TestBed.createComponent(TrailsMap);
    fixture.componentRef.setInput('trails', []);
    fixture.detectChanges();
    TestBed.tick();
    await Promise.resolve();
    TestBed.tick();
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('[role="status"]')?.textContent).toContain(
      'trails.map.unavailable',
    );
  });
});
