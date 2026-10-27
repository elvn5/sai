import type { TrackPoint, TrailMarker } from '@/core/api/types';
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
  BufferGeometry,
  GridHelper,
  Line,
  LineBasicMaterial,
  Mesh,
  MeshBasicMaterial,
  SphereGeometry,
  Vector3,
} from 'three';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';

extend({ GridHelper, Mesh, MeshBasicMaterial, SphereGeometry, TrackLine: Line });

@Component({
  selector: 'app-trail-scene',
  imports: [NgtArgs],
  schemas: [CUSTOM_ELEMENTS_SCHEMA],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <ngt-track-line [geometry]="geometry()" [material]="material" />
    <ngt-grid-helper *args="[14, 14, '#475569', '#334155']" [position]="[0, -0.05, 0]" />
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
  protected readonly material = new LineBasicMaterial({ color: '#34d399' });
  protected readonly geometry = computed(() =>
    new BufferGeometry().setFromPoints(this.points().map((p) => new Vector3(...p))),
  );
  constructor() {
    inject(DestroyRef).onDestroy(() => this.material.dispose());
    effect((cleanup) => {
      const geometry = this.geometry();
      cleanup(() => geometry.dispose());
    });
    effect((cleanup) => {
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
  protected readonly points = computed(() => projectTrack(this.track()));
  protected readonly markerPositions = computed(() =>
    validMarkers(this.markers(), this.track()).flatMap((m) => {
      const point = this.points()[m.pointIndex];
      return point ? [point] : [];
    }),
  );
}

@Component({
  selector: 'app-trail-scene-canvas',
  imports: [NgtCanvas, TrailScene],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'block h-80 sm:h-96' },
  template: `
    <ngt-canvas [camera]="{ position: [10, 9, 12], fov: 45 }" [dpr]="[1, 1.5]" frameloop="demand">
      <app-trail-scene *canvasContent [track]="track()" [markers]="markers()" />
    </ngt-canvas>
  `,
})
export class TrailSceneCanvas {
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
      @if (supported()) {
        <ng-container *ngComponentOutlet="scene; environmentInjector: renderer; inputs: sceneInputs()" />
      } @else {
        <p class="p-4 text-sm" role="status">{{ 'trails.editor.noWebgl' | transloco }}</p>
      }
      @if (missingElevation()) {
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
  }));
  protected readonly missingElevation = computed(() => this.track().some((p) => p[2] === null));
  protected readonly supported = signal(false);
  constructor() {
    inject(DestroyRef).onDestroy(() => this.renderer.destroy());
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
