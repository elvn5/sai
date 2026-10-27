import { InjectionToken } from '@angular/core';
import type * as Leaflet from 'leaflet';

export const LEAFLET_LOADER = new InjectionToken<typeof loadLeaflet>('LEAFLET_LOADER', {
  providedIn: 'root',
  factory: () => loadLeaflet,
});

// Browser-only. Normal viewport tiles with visible attribution; no bulk
// downloads or offline cache (OSMF tile usage policy).
export const MAP_TILES = 'https://tile.openstreetmap.org/{z}/{x}/{y}.png';
export const MAP_ATTRIBUTION =
  '© <a href="https://www.openstreetmap.org/copyright">OpenStreetMap contributors</a>';

export async function loadLeaflet(document: Document): Promise<typeof Leaflet> {
  if (!document.querySelector('link[data-leaflet]')) {
    const link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = '/media/leaflet/leaflet.css?v=1.9.4';
    link.dataset['leaflet'] = '';
    document.head.append(link);
  }
  const module = await import('leaflet');
  return (module as unknown as { default?: typeof module }).default ?? module;
}
