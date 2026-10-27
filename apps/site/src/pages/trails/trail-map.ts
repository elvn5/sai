import type { TrackPoint } from '@/core/api/types';
import { LEAFLET_LOADER, MAP_ATTRIBUTION, MAP_TILES, type loadLeaflet } from '@/core/lib/leaflet';
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
  signal,
  untracked,
  viewChild,
} from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';
import type { CircleMarker, CircleMarkerOptions, LayerGroup, Map as LeafletMap } from 'leaflet';

const MARKER: CircleMarkerOptions = { color: '#fff', weight: 2, fillOpacity: 1 };
const START = '#16a34a';
const FINISH = '#dc2626';
const HIGHLIGHT = '#2563eb';
// An OpenStreetMap map with the route's track, or just its
// start. Leaflet needs the DOM, so it loads in the browser only, after the
// page renders; the box keeps its size meanwhile.
@Component({
  selector: 'app-trail-map',
  imports: [TranslocoPipe],
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
  template: `
    <div #container class="h-80 w-full overflow-hidden rounded-card border border-border bg-text/5 sm:h-96" role="region" [attr.aria-label]="'trails.map.title' | transloco"></div>
    @if (failed()) { <p class="mt-2 text-sm text-text-muted" role="status">{{ 'trails.map.unavailable' | transloco }}</p> }
  `,
})
export class TrailMap {
  readonly track = input.required<TrackPoint[]>();
  readonly startLatitude = input<number | null>(null);
  readonly startLongitude = input<number | null>(null);
  // Index into the track to mark (from the elevation profile).
  readonly highlight = input<number | null>(null);
  protected readonly failed = signal(false);

  private readonly container = viewChild.required<ElementRef<HTMLDivElement>>('container');
  private readonly document = inject(DOCUMENT);
  private readonly loadLeaflet = inject(LEAFLET_LOADER);
  private destroyed = false;
  private map: LeafletMap | null = null;
  private marker: CircleMarker | null = null;
  private resize: ResizeObserver | null = null;
  private readonly ready = signal(false);
  private L: Awaited<ReturnType<typeof loadLeaflet>> | null = null;
  private layers: LayerGroup | null = null;

  constructor() {
    afterNextRender(() => void this.create().catch(() => this.failed.set(true)));
    effect(() => {
      const track = this.track();
      const lat = this.startLatitude();
      const lng = this.startLongitude();
      if (this.ready()) untracked(() => this.draw(track, lat, lng));
    });
    effect(() => {
      if (this.ready()) this.mark(this.highlight());
    });
    inject(DestroyRef).onDestroy(() => {
      this.destroyed = true;
      this.resize?.disconnect();
      this.map?.remove();
      this.map = null;
    });
  }

  private async create(): Promise<void> {
    const L = await this.loadLeaflet(this.document);
    if (this.destroyed) return;
    this.L = L;
    this.map = L.map(this.container().nativeElement, { scrollWheelZoom: false }).setView(
      [41.3, 74.7],
      7,
    );
    L.tileLayer(MAP_TILES, { maxZoom: 19, attribution: MAP_ATTRIBUTION }).addTo(this.map);
    this.layers = L.layerGroup().addTo(this.map);
    this.marker = L.circleMarker([0, 0], { ...MARKER, radius: 6, fillColor: HIGHLIGHT });
    this.resize = new ResizeObserver(() => this.map?.invalidateSize());
    this.resize.observe(this.container().nativeElement);
    this.ready.set(true);
  }

  private draw(track: TrackPoint[], lat: number | null, lng: number | null): void {
    const L = this.L;
    const map = this.map;
    const layers = this.layers;
    if (!L || !map || !layers) return;
    layers.clearLayers();
    const points = track.map((p): [number, number] => [p[0], p[1]]);
    if (points.length > 1) {
      const line = L.polyline(points, { color: '#ef4444', weight: 4, opacity: 0.9 }).addTo(layers);
      L.circleMarker(points[0] as [number, number], {
        ...MARKER,
        radius: 7,
        fillColor: START,
      }).addTo(layers);
      L.circleMarker(points.at(-1) as [number, number], {
        ...MARKER,
        radius: 7,
        fillColor: FINISH,
      }).addTo(layers);
      map.fitBounds(line.getBounds(), { padding: [24, 24] });
    } else if (lat !== null && lng !== null) {
      L.circleMarker([lat, lng], { ...MARKER, radius: 8, fillColor: START }).addTo(layers);
      map.setView([lat, lng], 13);
    }
    this.mark(this.highlight());
  }

  private mark(index: number | null): void {
    if (!this.map || !this.marker) return;
    const point = index === null ? undefined : this.track()[index];
    if (point) this.marker.setLatLng([point[0], point[1]]).addTo(this.map);
    else this.marker.remove();
  }
}
