import { IMAGE_LOADER, type ImageLoaderConfig, NgOptimizedImage } from '@angular/common';
import { ChangeDetectionStrategy, Component } from '@angular/core';

// The resized copies in public/media (served under /media/, which nginx
// already sends to this app).
const WIDTHS = [768, 1280, 1920];

function heroUrl(config: ImageLoaderConfig): string {
  const width = WIDTHS.find((w) => w >= (config.width ?? 0)) ?? (WIDTHS.at(-1) as number);
  return `/media/${config.src}-${width}.webp`;
}

// The home page's mountain photo, the page's LCP image: preloaded, and
// sized to the screen. Its loader is its own, so other images keep theirs.
@Component({
  selector: 'app-hero-image',
  imports: [NgOptimizedImage],
  providers: [{ provide: IMAGE_LOADER, useValue: heroUrl }],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'absolute inset-0 -z-20' },
  template: `<img ngSrc="hero-mountain" ngSrcset="768w, 1280w, 1920w" sizes="100vw" fill priority alt="" class="object-cover" />`,
})
export class HeroImage {}
