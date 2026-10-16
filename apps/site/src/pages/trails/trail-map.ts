import type { TrackPoint } from '@/core/api/types';
import {
  ChangeDetectionStrategy,
  Component,
  DOCUMENT,
  DestroyRef,
  ElementRef,
  ViewEncapsulation,
  afterNextRender,
  effect,
  inject,
  input,
  viewChild,
} from '@angular/core';
import type { CircleMarker, CircleMarkerOptions, Map as LeafletMap } from 'leaflet';

const MARKER: CircleMarkerOptions = { color: '#fff', weight: 2, fillOpacity: 1 };
const START = '#16a34a';
const FINISH = '#dc2626';
const HIGHLIGHT = '#2563eb';
const TRACK_DRAW_MS = 1200;
// Leaflet's stylesheet, copied from the package at build time (angular.json
// assets) under /media/, which production nginx sends to this app: loaded
// with the map, it stays out of every other page.
const LEAFLET_CSS = '/media/leaflet/leaflet.css?v=1.9.4';

let stylesheet: Promise<void> | null = null;

function loadStylesheet(document: Document): Promise<void> {
  stylesheet ??= new Promise((resolve) => {
    const link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = LEAFLET_CSS;
    // A map without its styles still works: don't wait forever.
    link.onload = link.onerror = () => resolve();
    document.head.append(link);
  });
  return stylesheet;
}

// A topographic map (OpenTopoMap) with the route's track, or just its
// start. Leaflet needs the DOM, so it loads in the browser only, after the
// page renders; the box keeps its size meanwhile.
@Component({
  selector: 'app-trail-map',
  changeDetection: ChangeDetectionStrategy.OnPush,
  // Styles the elements Leaflet creates.
  encapsulation: ViewEncapsulation.None,
  styles: `
    app-trail-map .leaflet-container {
      z-index: 0;
      font: inherit;
    }
  `,
  host: { class: 'block' },
  template: `<div #container class="h-80 w-full overflow-hidden rounded-card border border-border bg-text/5 sm:h-96"></div>`,
})
export class TrailMap {
  readonly track = input.required<TrackPoint[]>();
  readonly startLatitude = input<number | null>(null);
  readonly startLongitude = input<number | null>(null);
  // Index into the track to mark (from the elevation profile).
  readonly highlight = input<number | null>(null);

  private readonly container = viewChild.required<ElementRef<HTMLDivElement>>('container');
  private readonly document = inject(DOCUMENT);
  private destroyed = false;
  private map: LeafletMap | null = null;
  private marker: CircleMarker | null = null;
  private observer: IntersectionObserver | null = null;

  constructor() {
    afterNextRender(() => void this.create());
    effect(() => this.mark(this.highlight()));
    inject(DestroyRef).onDestroy(() => {
      this.destroyed = true;
      this.observer?.disconnect();
      this.map?.remove();
      this.map = null;
    });
  }

  private async create(): Promise<void> {
    const [module] = await Promise.all([import('leaflet'), loadStylesheet(this.document)]);
    if (this.destroyed) return;
    // A CommonJS package: the bundled import holds the library as `default`.
    const L = (module as unknown as { default?: typeof module }).default ?? module;
    const map = L.map(this.container().nativeElement, { scrollWheelZoom: false });
    L.tileLayer('https://{s}.tile.opentopomap.org/{z}/{x}/{y}.png', {
      maxZoom: 17,
      attribution:
        '© <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>, SRTM | © <a href="https://opentopomap.org">OpenTopoMap</a> (CC-BY-SA)',
    }).addTo(map);

    const points = this.track().map((p): [number, number] => [p[0], p[1]]);
    const lat = this.startLatitude();
    const lng = this.startLongitude();
    if (points.length > 1) {
      const line = L.polyline(points, { color: '#ef4444', weight: 4, opacity: 0.9 }).addTo(map);
      L.circleMarker(points[0] as [number, number], {
        ...MARKER,
        radius: 7,
        fillColor: START,
      }).addTo(map);
      L.circleMarker(points.at(-1) as [number, number], {
        ...MARKER,
        radius: 7,
        fillColor: FINISH,
      }).addTo(map);
      map.fitBounds(line.getBounds(), { padding: [24, 24] });
      this.drawWhenSeen(line.getElement());
    } else if (lat !== null && lng !== null) {
      L.circleMarker([lat, lng], { ...MARKER, radius: 8, fillColor: START }).addTo(map);
      map.setView([lat, lng], 13);
    }
    this.marker = L.circleMarker([0, 0], { ...MARKER, radius: 6, fillColor: HIGHLIGHT });
    this.map = map;
  }

  // The track draws itself from start to finish the first time the map
  // scrolls into view.
  private drawWhenSeen(element: Element | undefined): void {
    if (!(element instanceof SVGPathElement)) return;
    if (matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    const length = element.getTotalLength();
    element.style.strokeDasharray = `${length}`;
    element.style.strokeDashoffset = `${length}`;
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (!entry?.isIntersecting) return;
        observer.disconnect();
        element
          .animate([{ strokeDashoffset: length }, { strokeDashoffset: 0 }], {
            duration: TRACK_DRAW_MS,
            easing: 'cubic-bezier(0.2, 0, 0, 1)',
          })
          .finished.finally(() => {
            // Zooming redraws the path: no dashes left behind.
            element.style.strokeDasharray = '';
            element.style.strokeDashoffset = '';
          });
      },
      { threshold: 0.4 },
    );
    observer.observe(this.container().nativeElement);
    this.observer = observer;
  }

  private mark(index: number | null): void {
    if (!this.map || !this.marker) return;
    const point = index === null ? undefined : this.track()[index];
    if (point) this.marker.setLatLng([point[0], point[1]]).addTo(this.map);
    else this.marker.remove();
  }
}
