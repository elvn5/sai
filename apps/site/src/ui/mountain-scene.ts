import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';

// The palettes of the design sketches: sky top and bottom, far, middle and
// near ridges, water.
const PALETTES = {
  lake: ['#7fb3e6', '#f6d6b5', '#9fb1c9', '#5f7896', '#2f4a5f', '#3d8ab8'],
  peaks: ['#5b86c4', '#cfe3f3', '#c9d6e6', '#7d93ad', '#3b5468', '#ffffff'],
  sunset: ['#f2a65a', '#fde4c8', '#c98b6b', '#8f5a4a', '#4b2f2c', '#e8c27a'],
  meadow: ['#89c2d9', '#e9f5db', '#a7c4a0', '#6a994e', '#386641', '#a7d3e0'],
  dusk: ['#334e68', '#829ab1', '#627d98', '#486581', '#243b53', '#9fb3c8'],
} as const;

export type ScenePalette = keyof typeof PALETTES;

let nextId = 0;

// The site's illustration: layered ridges with snow caps, a sun and a lake,
// drawn in SVG (no photo to download, crisp at any size). It fills its box
// from the bottom up: `cover` crops it to fill a wide box; `bottom` keeps
// the ridges at their size along the bottom and stretches the sky above (tall
// or narrow boxes). In dark mode it dims; with `parallax` the ridges drift
// apart while the page scrolls (scroll-driven CSS, none with reduced motion).
@Component({
  selector: 'app-mountain-scene',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: {
    class: 'absolute inset-0 -z-20 block overflow-hidden',
    'aria-hidden': 'true',
    '[style.background]': "fit() === 'bottom' ? sky() : null",
  },
  styles: `
    :host-context(.dark) {
      filter: brightness(0.55) saturate(0.85);
    }
    @keyframes drift-far {
      to {
        transform: translateY(70px);
      }
    }
    @keyframes drift-mid {
      to {
        transform: translateY(40px);
      }
    }
    @keyframes drift-sun {
      to {
        transform: translateY(110px);
      }
    }
    @supports (animation-timeline: scroll()) {
      @media (prefers-reduced-motion: no-preference) {
        .parallax .far,
        .parallax .mid,
        .parallax .sun {
          animation: drift-far linear both;
          animation-timeline: scroll(root);
          animation-range: 0 100vh;
        }
        .parallax .mid {
          animation-name: drift-mid;
        }
        .parallax .sun {
          animation-name: drift-sun;
        }
      }
    }
  `,
  template: `
    <svg
      [class]="fit() === 'bottom' ? 'absolute bottom-0 left-1/2 aspect-[8/3] w-full min-w-[48rem] -translate-x-1/2' : 'size-full'"
      [class.parallax]="parallax()"
      viewBox="0 0 1600 600"
      [attr.preserveAspectRatio]="fit() === 'bottom' ? 'xMidYMax meet' : 'xMidYMax slice'"
      focusable="false"
    >
      <defs>
        <linearGradient [attr.id]="skyId" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" [attr.stop-color]="colors()[0]" />
          <stop offset="1" [attr.stop-color]="colors()[1]" />
        </linearGradient>
      </defs>
      <!-- A bottom-fitted scene's sky is its box's background. -->
      @if (fit() === 'cover') {
        <rect width="1600" height="600" [attr.fill]="'url(#' + skyId + ')'" />
      }
      <circle class="sun" cx="1240" cy="150" r="58" fill="#fff" opacity="0.55" />
      <g class="far">
        <path d="M0 360 L160 250 L300 320 L460 200 L560 150 L700 260 L840 190 L980 250 L1120 140 L1260 230 L1400 180 L1600 240 V600 H0Z" [attr.fill]="colors()[2]" />
        <path d="M560 150 L604 198 L586 194 L560 176 L538 202 L520 194Z M1120 140 L1166 190 L1146 186 L1120 168 L1096 194 L1076 188Z" fill="#fff" opacity="0.85" />
      </g>
      <path class="mid" d="M0 430 L140 360 L300 410 L460 330 L640 405 L800 345 L980 410 L1160 340 L1340 400 L1480 350 L1600 380 V600 H0Z" [attr.fill]="colors()[3]" />
      @if (lake()) {
        <rect y="470" width="1600" height="130" [attr.fill]="colors()[5]" opacity="0.55" />
      }
      <path d="M0 540 L160 490 L340 525 L520 480 L720 532 L920 495 L1120 530 L1320 490 L1600 520 V600 H0Z" [attr.fill]="colors()[4]" />
    </svg>
  `,
})
export class MountainScene {
  readonly palette = input<ScenePalette>('lake');
  // Water in the foreground (the lake palettes).
  readonly lake = input(true);
  readonly parallax = input(false);
  readonly fit = input<'cover' | 'bottom'>('cover');

  protected readonly skyId = `scene-sky-${nextId++}`;
  protected readonly colors = computed(() => PALETTES[this.palette()]);
  protected readonly sky = computed(
    () => `linear-gradient(${this.colors()[0]}, ${this.colors()[1]})`,
  );
}
