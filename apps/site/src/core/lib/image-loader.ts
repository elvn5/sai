import { APP_CONFIG, type AppConfig } from '@/core/config/app-config';
import {
  IMAGE_CONFIG,
  IMAGE_LOADER,
  type ImageConfig,
  type ImageLoader,
  type ImageLoaderConfig,
} from '@angular/common';
import { type Provider, inject } from '@angular/core';

// The widths services/media makes thumbnails at (imageconv.ThumbnailWidths)
// — keep the two lists the same.
export const IMAGE_WIDTHS: readonly number[] = [128, 256, 384, 640, 828, 1080, 1280, 1920];

// The thumbnail of an uploaded image (an /uploads file or an object in the
// R2 bucket) at the smallest width that covers `width`. Anything else — and
// a request without a width, NgOptimizedImage's plain src — keeps its URL.
export function thumbnailUrl(
  config: Pick<AppConfig, 'apiBaseUrl' | 'mediaPublicUrl'>,
  src: string,
  width?: number,
): string {
  const source = width ? mediaSource(config, src) : null;
  if (!source || !width) return src;
  const fit = IMAGE_WIDTHS.find((w) => w >= width) ?? IMAGE_WIDTHS[IMAGE_WIDTHS.length - 1];
  return `${config.apiBaseUrl}/images/${fit}/${source}`;
}

function mediaSource(
  config: Pick<AppConfig, 'apiBaseUrl' | 'mediaPublicUrl'>,
  src: string,
): string | null {
  if (/[?#]/.test(src)) return null;
  for (const uploads of [`${config.apiBaseUrl}/uploads/`, '/uploads/']) {
    if (src.startsWith(uploads)) return `uploads/${src.slice(uploads.length)}`;
  }
  const bucket = config.mediaPublicUrl && `${config.mediaPublicUrl}/`;
  if (bucket && src.startsWith(bucket)) return `r2/${src.slice(bucket.length)}`;
  return null;
}

// NgOptimizedImage everywhere gets srcset entries at IMAGE_WIDTHS, served as
// thumbnails, instead of the full upload (originals are up to 2000px).
export function provideImageThumbnails(): Provider[] {
  return [
    {
      provide: IMAGE_LOADER,
      useFactory: (): ImageLoader => {
        const config = inject(APP_CONFIG);
        return ({ src, width }: ImageLoaderConfig) => thumbnailUrl(config, src, width);
      },
    },
    { provide: IMAGE_CONFIG, useValue: { breakpoints: [...IMAGE_WIDTHS] } satisfies ImageConfig },
  ];
}
