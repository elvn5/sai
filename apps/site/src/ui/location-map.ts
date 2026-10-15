import { mapsLink } from '@/core/lib/contact-links';
import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { inject } from '@angular/core';
import { DomSanitizer, type SafeResourceUrl } from '@angular/platform-browser';
import { TranslocoPipe } from '@jsverse/transloco';

// About 1 km around the point.
const BBOX_DELTA = 0.01;

// The address and an OpenStreetMap embed (no API key), plus a link to
// Google Maps.
@Component({
  selector: 'app-location-map',
  // A box of its own: deferred blocks observe their root element.
  host: { class: 'block' },
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (embedUrl() || address()) {
      <div class="flex flex-col gap-3">
        @if (address()) {
          <p class="flex items-start gap-2 text-text">
            <span class="icon-[heroicons--map-pin] mt-0.5 size-5 shrink-0 text-brand-600" aria-hidden="true"></span>
            {{ address() }}
          </p>
        }
        @if (embedUrl(); as url) {
          <iframe [src]="url" [title]="address() || ('listing.map.title' | transloco)" loading="lazy" class="h-64 w-full rounded-card border border-border"></iframe>
        }
        @if (externalUrl(); as url) {
          <a [href]="url" target="_blank" rel="noopener noreferrer" class="inline-flex items-center gap-1 text-sm text-brand-700 hover:underline dark:text-brand-300">
            {{ 'listing.map.openInGoogle' | transloco }}
            <span class="icon-[heroicons--arrow-top-right-on-square] size-4" aria-hidden="true"></span>
          </a>
        }
      </div>
    }
  `,
})
export class LocationMap {
  readonly latitude = input<number | null>(null);
  readonly longitude = input<number | null>(null);
  readonly address = input<string | null>(null);
  private readonly sanitizer = inject(DomSanitizer);

  protected readonly embedUrl = computed((): SafeResourceUrl | null => {
    const lat = this.latitude();
    const lng = this.longitude();
    if (lat === null || lng === null) return null;
    const bbox = [lng - BBOX_DELTA, lat - BBOX_DELTA, lng + BBOX_DELTA, lat + BBOX_DELTA].join(',');
    // Numbers only, on a fixed OpenStreetMap address: safe to embed.
    return this.sanitizer.bypassSecurityTrustResourceUrl(
      `https://www.openstreetmap.org/export/embed.html?bbox=${bbox}&layer=mapnik&marker=${lat},${lng}`,
    );
  });
  protected readonly externalUrl = computed(() =>
    mapsLink(this.latitude(), this.longitude(), this.address()),
  );
}
