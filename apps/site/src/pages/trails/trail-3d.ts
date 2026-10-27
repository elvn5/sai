import type { TrackPoint, TrailMarker } from '@/core/api/types';
import { type SatelliteAtlas, loadSatellite, satelliteUV } from '@/core/lib/satellite';
import {
  type Terrain,
  drapedTrack,
  loadTerrain,
  surfaceHeight,
  terrainPosition,
} from '@/core/lib/terrain';
import { projectTrack, validMarkers } from '@/core/lib/trail-geometry';
import { NgComponentOutlet } from '@angular/common';
import {
  CUSTOM_ELEMENTS_SCHEMA,
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  EnvironmentInjector,
  afterNextRender,
  computed,
  createEnvironmentInjector,
  effect,
  inject,
  input,
  signal,
} from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';
import { NgtArgs, extend, injectStore } from 'angular-three';
import { NgtCanvas, provideNgtRenderer } from 'angular-three/dom';
import {
  AmbientLight,
  BufferGeometry,
  CanvasTexture,
  Color,
  DirectionalLight,
  Float32BufferAttribute,
  GridHelper,
  Mesh,
  MeshBasicMaterial,
  MeshStandardMaterial,
  SRGBColorSpace,
  SphereGeometry,
} from 'three';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';
import { Line2 } from 'three/addons/lines/Line2.js';
import { LineGeometry } from 'three/addons/lines/LineGeometry.js';
import { LineMaterial } from 'three/addons/lines/LineMaterial.js';

extend({
  AmbientLight,
  DirectionalLight,
  MeshStandardMaterial,
  GridHelper,
  Mesh,
  MeshBasicMaterial,
  SphereGeometry,
  TrackLine: Line2,
});

@Component({
  selector: 'app-trail-scene',
  imports: [NgtArgs],
  schemas: [CUSTOM_ELEMENTS_SCHEMA],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <ngt-track-line [geometry]="geometry()" [material]="outline" [renderOrder]="1" />
    <ngt-track-line [geometry]="geometry()" [material]="material" [renderOrder]="2" />
    @if (terrain()) {
      <ngt-ambient-light [intensity]="1.6" />
      <ngt-directional-light [position]="[-8, 12, 6]" [intensity]="1.4" />
      <ngt-mesh [geometry]="surface()" [material]="surfaceMaterial" />
    } @else {
    <ngt-grid-helper *args="[14, 14, '#475569', '#334155']" [position]="[0, -0.05, 0]" />
    }
    @for (position of markerPositions(); track $index) {
      <ngt-mesh [position]="position">
        <ngt-sphere-geometry *args="[0.1, 12, 12]" />
        <ngt-mesh-basic-material color="#fbbf24" />
      </ngt-mesh>
    }

  `,
})
export class TrailScene {
  private readonly store = injectStore();
  protected readonly material = new LineMaterial({
    color: '#ff563d',
    linewidth: 4,
    depthWrite: false,
    toneMapped: false,
  });
  protected readonly outline = new LineMaterial({
    color: '#182028',
    linewidth: 7,
    depthWrite: false,
    toneMapped: false,
  });
  protected readonly geometry = computed(() =>
    new LineGeometry().setPositions(this.points().flat()),
  );
  protected readonly surfaceMaterial = new MeshStandardMaterial({
    vertexColors: true,
    roughness: 0.95,
    metalness: 0,
  });
  protected readonly surface = computed(() => {
    const t = this.terrain();
    const geometry = new BufferGeometry();
    if (!t) return geometry;
    const positions: number[] = [];
    const colors: number[] = [];
    const indices: number[] = [];
    const uvs: number[] = [];
    const satellite = this.satellite();
    const low = new Color('#42634a');
    const rock = new Color('#a49a85');
    const snow = new Color('#e9edf2');
    for (let row = 0; row <= t.rows; row++) {
      for (let col = 0; col <= t.columns; col++) {
        const height = t.heights[row * (t.columns + 1) + col] ?? t.base;
        positions.push(
          ...terrainPosition(
            t,
            t.north - (row / t.rows) * (t.north - t.south),
            t.west + (col / t.columns) * (t.east - t.west),
            height,
          ),
        );
        if (satellite)
          uvs.push(
            ...satelliteUV(
              satellite,
              t.north - (row / t.rows) * (t.north - t.south),
              t.west + (col / t.columns) * (t.east - t.west),
            ),
          );
        const color =
          height < 3300
            ? low.clone().lerp(rock, Math.max(0, Math.min(1, (height - 2200) / 1100)))
            : rock.clone().lerp(snow, Math.min(1, (height - 3300) / 1100));
        colors.push(color.r, color.g, color.b);
        if (row < t.rows && col < t.columns) {
          const a = row * (t.columns + 1) + col;
          const c = a + t.columns + 1;
          indices.push(a, c, a + 1, a + 1, c, c + 1);
        }
      }
    }
    geometry.setAttribute('position', new Float32BufferAttribute(positions, 3));
    geometry.setAttribute('color', new Float32BufferAttribute(colors, 3));
    if (satellite) geometry.setAttribute('uv', new Float32BufferAttribute(uvs, 2));
    geometry.setIndex(indices);
    geometry.computeVertexNormals();
    return geometry;
  });
  constructor() {
    inject(DestroyRef).onDestroy(() => this.surfaceMaterial.dispose());
    effect((cleanup) => {
      const atlas = this.satellite();
      const texture = atlas ? new CanvasTexture(atlas.canvas) : null;
      if (texture) {
        texture.colorSpace = SRGBColorSpace;
        texture.anisotropy = Math.min(8, this.store.gl().capabilities.getMaxAnisotropy());
      }
      this.surfaceMaterial.map = texture;
      this.surfaceMaterial.vertexColors = !texture;
      this.surfaceMaterial.needsUpdate = true;
      this.store.invalidate()();
      cleanup(() => texture?.dispose());
    });
    inject(DestroyRef).onDestroy(() => this.outline.dispose());
    effect((cleanup) => {
      const surface = this.surface();
      cleanup(() => surface.dispose());
    });
    inject(DestroyRef).onDestroy(() => this.material.dispose());
    effect((cleanup) => {
      const geometry = this.geometry();
      cleanup(() => geometry.dispose());
    });
    effect((cleanup) => {
      const canvas = this.store.gl.domElement();
      const resize = (): void => {
        this.material.resolution.set(canvas.clientWidth, canvas.clientHeight);
        this.outline.resolution.set(canvas.clientWidth, canvas.clientHeight);
        this.store.invalidate()();
      };
      const observer = new ResizeObserver(resize);
      observer.observe(canvas);
      resize();
      cleanup(() => observer.disconnect());
      const controls = new OrbitControls(this.store.camera(), this.store.gl.domElement());
      const invalidate = this.store.invalidate();
      controls.target.set(0, 1, 0);
      controls.minDistance = 3;
      controls.maxDistance = 40;
      const onChange = (): void => invalidate();
      controls.addEventListener('change', onChange);
      controls.update();
      cleanup(() => {
        controls.removeEventListener('change', onChange);
        controls.dispose();
      });
    });
  }
  readonly track = input.required<TrackPoint[]>();
  readonly markers = input<TrailMarker[]>([]);
  readonly terrain = input<Terrain | null>(null);
  readonly satellite = input<SatelliteAtlas | null>(null);
  protected readonly points = computed(() => {
    const t = this.terrain();
    return t ? drapedTrack(t, this.track()) : projectTrack(this.track());
  });
  protected readonly markerPositions = computed(() =>
    validMarkers(this.markers(), this.track()).flatMap((m) => {
      const t = this.terrain();
      const source = this.track()[m.pointIndex];
      const point =
        t && source
          ? terrainPosition(t, source[0], source[1], surfaceHeight(t, source[0], source[1]))
          : this.points()[m.pointIndex];
      if (t && point) point[1] += 0.08;
      return point ? [point] : [];
    }),
  );
}

@Component({
  selector: 'app-trail-scene-canvas',
  imports: [NgtCanvas, TrailScene],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block h-96 sm:h-[32rem]' },
  template: `
    <ngt-canvas [camera]="{ position: [8, 7, 9], fov: 45 }" [dpr]="[1, 1.5]" frameloop="demand">
      <app-trail-scene *canvasContent [track]="track()" [markers]="markers()" [terrain]="terrain()" [satellite]="satellite()" />
    </ngt-canvas>
  `,
})
export class TrailSceneCanvas {
  readonly terrain = input<Terrain | null>(null);
  readonly satellite = input<SatelliteAtlas | null>(null);
  readonly track = input.required<TrackPoint[]>();
  readonly markers = input<TrailMarker[]>([]);
}

// Loaded only on demand; the custom renderer and Three.js stay out of
// the application shell. Neither the renderer nor WebGL runs during SSR.
@Component({
  selector: 'app-trail-3d',
  imports: [NgComponentOutlet, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block' },
  template: `
    <section class="overflow-hidden rounded-card border border-border bg-slate-950 text-white">
      <div class="p-4">
        <h3 class="font-semibold">{{ 'trails.editor.threeTitle' | transloco }}</h3>
        <p class="mt-1 text-sm text-slate-300">{{ 'trails.editor.threeHint' | transloco }}</p>
      </div>
      @if (loading()) { <p class="px-4 pb-3 text-sm text-slate-300" role="status">{{ 'trails.editor.terrainLoading' | transloco }}</p> }
      @if (failed()) {
        <div class="px-4 pb-3 text-sm text-amber-200" role="status">{{ 'trails.editor.terrainFailed' | transloco }}
          <button type="button" class="ml-2 underline" (click)="retry()">{{ 'trails.editor.terrainRetry' | transloco }}</button>
        </div>
      }
      @if (imageryLoading()) { <p class="px-4 pb-3 text-sm text-slate-300" role="status">{{ 'trails.editor.satelliteLoading' | transloco }}</p> }
      @if (imageryFailed()) {
        <p class="px-4 pb-3 text-sm text-amber-200" role="status">{{ 'trails.editor.satelliteFailed' | transloco }}
          <button type="button" class="ml-2 underline" (click)="retryImagery()">{{ 'trails.editor.terrainRetry' | transloco }}</button>
        </p>
      }
      @if (imagery()) {
        <div class="flex gap-2 px-4 pb-3">
          <button type="button" class="rounded border px-3 py-1 text-sm" [class.bg-slate-700]="showSatellite()" [attr.aria-pressed]="showSatellite()" (click)="showSatellite.set(true)">{{ 'trails.editor.satellite' | transloco }}</button>
          <button type="button" class="rounded border px-3 py-1 text-sm" [class.bg-slate-700]="!showSatellite()" [attr.aria-pressed]="!showSatellite()" (click)="showSatellite.set(false)">{{ 'trails.editor.elevationColors' | transloco }}</button>
        </div>
      }
      @if (supported()) {
        <ng-container *ngComponentOutlet="scene; environmentInjector: renderer; inputs: sceneInputs()" />
      } @else {
        <p class="p-4 text-sm" role="status">{{ 'trails.editor.noWebgl' | transloco }}</p>
      }
      @if (terrain()) {
        <p class="px-4 pb-4 text-xs text-slate-300">{{ (imagery() && showSatellite() ? 'trails.editor.satelliteNote' : 'trails.editor.terrainNote') | transloco }}
          <a href="https://registry.opendata.aws/terrain-tiles/" target="_blank" rel="noopener noreferrer" class="underline">Mapzen Terrain Tiles</a> · SRTM / GMTED2010 courtesy of USGS.
        </p>
      }
      @if (imagery() && showSatellite()) {
        <p class="px-4 pb-4 text-xs text-slate-300"><a href="https://www.arcgis.com/home/item.html?id=10df2279f9684e4a9f6a7f08febac2a9" target="_blank" rel="noopener noreferrer" class="underline">Powered by Esri · World Imagery</a> — Esri, Vantor, Earthstar Geographics, and the GIS User Community.</p>
      }
      @if (missingElevation() && !terrain()) {
        <p class="p-4 text-sm text-amber-200">{{ 'trails.editor.missingElevation' | transloco }}</p>
      }
    </section>
  `,
})
export class Trail3d {
  readonly track = input.required<TrackPoint[]>();
  readonly markers = input<TrailMarker[]>([]);
  protected readonly scene = TrailSceneCanvas;
  protected readonly renderer = createEnvironmentInjector(
    [provideNgtRenderer()],
    inject(EnvironmentInjector),
  );
  protected readonly sceneInputs = computed(() => ({
    track: this.track(),
    markers: this.markers(),
    terrain: this.terrain(),
    satellite: this.showSatellite() ? this.imagery() : null,
  }));
  protected readonly missingElevation = computed(() => this.track().some((p) => p[2] === null));
  protected readonly supported = signal(false);
  protected readonly terrain = signal<Terrain | null>(null);
  protected readonly loading = signal(false);
  protected readonly failed = signal(false);
  protected readonly imagery = signal<SatelliteAtlas | null>(null);
  protected readonly imageryLoading = signal(false);
  protected readonly imageryFailed = signal(false);
  protected readonly showSatellite = signal(true);
  private readonly imageryAttempts = signal(0);
  protected retryImagery(): void {
    this.imageryAttempts.update((n) => n + 1);
  }
  private readonly attempts = signal(0);
  protected retry(): void {
    this.attempts.update((n) => n + 1);
  }
  constructor() {
    inject(DestroyRef).onDestroy(() => this.renderer.destroy());
    effect((cleanup) => {
      const track = this.track();
      this.attempts();
      this.terrain.set(null);
      this.loading.set(false);
      this.failed.set(false);
      if (!this.supported() || track.length < 2) return;
      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(), 45000);
      let active = true;
      this.terrain.set(null);
      this.loading.set(true);
      this.failed.set(false);
      loadTerrain(track, controller.signal)
        .then((terrain) => {
          if (active) this.terrain.set(terrain);
        })
        .catch(() => {
          if (active) this.failed.set(true);
        })
        .finally(() => {
          clearTimeout(timeout);
          if (active) this.loading.set(false);
        });
      cleanup(() => {
        active = false;
        clearTimeout(timeout);
        controller.abort();
      });
    });
    effect((cleanup) => {
      const terrain = this.terrain();
      this.imageryAttempts();
      this.imagery.set(null);
      this.imageryLoading.set(false);
      this.imageryFailed.set(false);
      if (!terrain) return;
      let active = true;
      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(), 45000);
      this.imageryLoading.set(true);
      loadSatellite(terrain, controller.signal)
        .then((atlas) => {
          if (active) this.imagery.set(atlas);
        })
        .catch(() => {
          controller.abort();
          if (active) this.imageryFailed.set(true);
        })
        .finally(() => {
          clearTimeout(timeout);
          if (active) this.imageryLoading.set(false);
        });
      cleanup(() => {
        active = false;
        clearTimeout(timeout);
        controller.abort();
      });
    });
    afterNextRender(() => {
      try {
        const gl = document.createElement('canvas').getContext('webgl2');
        this.supported.set(!!gl);
        gl?.getExtension('WEBGL_lose_context')?.loseContext();
      } catch {
        this.supported.set(false);
      }
    });
  }
}
