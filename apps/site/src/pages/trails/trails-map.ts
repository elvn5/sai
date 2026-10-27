import type { Trail } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { localizedPath } from '@/core/i18n/langs';
import { LEAFLET_LOADER, MAP_ATTRIBUTION, MAP_TILES } from '@/core/lib/leaflet';
import { trailStart } from '@/core/lib/trail-map';
import {
  ChangeDetectionStrategy,
  Component,
  DOCUMENT,
  DestroyRef,
  ElementRef,
  ViewEncapsulation,
  afterNextRender,
  computed,
  effect,
  inject,
  input,
  signal,
  viewChild,
} from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';
import type * as Leaflet from 'leaflet';

@Component({
  selector: 'app-trails-map',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  encapsulation: ViewEncapsulation.None,
  host: { class: 'block' },
  styles: `
    app-trails-map .leaflet-container { z-index: 0; font: inherit; }
    .trail-map-pin { display: grid; place-items: center; border: 2px solid white; border-radius: 50%; background: #166534; color: white; font: bold 12px sans-serif; box-shadow: 0 2px 6px #0006; }
    .trail-map-pin:focus { outline: 3px solid #2563eb; outline-offset: 3px; }
  `,
  template: `
    <div class="overflow-hidden rounded-card border border-border bg-surface">
      <div class="flex flex-wrap items-center justify-between gap-2 border-b border-border px-4 py-2">
        <p class="text-sm text-text-muted">{{ 'trails.map.count' | transloco: { count: mapped().length, total: trails().length } }}</p>
        <button type="button" class="ui-btn-ghost text-sm" (click)="fit()">{{ 'trails.map.fit' | transloco }}</button>
      </div>
      <div #container class="h-[55vh] min-h-80 max-h-[640px] w-full bg-text/5" role="region" [attr.aria-label]="'trails.map.title' | transloco"></div>
      @if (failed()) {
        <p class="px-4 py-3 text-sm text-text-muted" role="status">{{ 'trails.map.unavailable' | transloco }}</p>
      }
      @if (mapped().length < trails().length) {
        <p class="px-4 py-3 text-sm text-text-muted">{{ 'trails.map.missingCoordinates' | transloco }}</p>
      }
    </div>
  `,
})
export class TrailsMap {
  readonly trails = input.required<Trail[]>();
  protected readonly mapped = computed(() => this.trails().filter((trail) => trailStart(trail)));
  protected readonly failed = signal(false);
  private readonly container = viewChild.required<ElementRef<HTMLDivElement>>('container');
  private readonly document = inject(DOCUMENT);
  private readonly loadLeaflet = inject(LEAFLET_LOADER);
  private readonly lang = injectLang();
  private readonly ready = signal(false);
  private readonly destroyRef = inject(DestroyRef);
  private L: typeof Leaflet | null = null;
  private map: Leaflet.Map | null = null;
  private markers: Leaflet.LayerGroup | null = null;
  private resize: ResizeObserver | null = null;

  constructor() {
    afterNextRender(() => void this.create());
    effect(() => {
      const trails = this.mapped();
      const lang = this.lang();
      if (!this.ready() || !this.L || !this.map || !this.markers) return;
      this.markers.clearLayers();
      for (const [index, trail] of trails.entries()) {
        const point = trailStart(trail);
        if (!point) continue;
        const link = this.document.createElement('a');
        link.href = localizedPath(lang, `/trails/${encodeURIComponent(trail.slug)}`);
        link.textContent = trail.name;
        link.style.fontWeight = '700';
        const icon = this.L.divIcon({
          className: 'trail-map-pin',
          html: String(index + 1),
          iconSize: [30, 30],
          iconAnchor: [15, 15],
        });
        this.L.marker(point, { icon, title: trail.name, alt: trail.name, keyboard: true })
          .bindPopup(link)
          .addTo(this.markers);
      }
      this.fit();
    });
    this.destroyRef.onDestroy(() => {
      this.resize?.disconnect();
      this.map?.remove();
    });
  }

  private async create(): Promise<void> {
    try {
      const L = await this.loadLeaflet(this.document);
      if (this.destroyRef.destroyed) return;
      this.L = L;
      this.map = L.map(this.container().nativeElement, { scrollWheelZoom: false }).setView(
        [41.3, 74.7],
        7,
      );
      L.tileLayer(MAP_TILES, { maxZoom: 19, attribution: MAP_ATTRIBUTION })
        .on('tileerror', () => this.failed.set(true))
        .addTo(this.map);
      this.markers = L.layerGroup().addTo(this.map);
      this.resize = new ResizeObserver(() => this.map?.invalidateSize());
      this.resize.observe(this.container().nativeElement);
      this.ready.set(true);
    } catch {
      this.failed.set(true);
    }
  }

  protected fit(): void {
    if (!this.L || !this.map) return;
    const points = this.mapped().flatMap((trail) => {
      const point = trailStart(trail);
      return point ? [point] : [];
    });
    if (points.length)
      this.map.fitBounds(this.L.latLngBounds(points), { padding: [35, 35], maxZoom: 13 });
    else this.map.setView([41.3, 74.7], 7);
  }
}
