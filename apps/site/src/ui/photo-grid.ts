import { APP_CONFIG } from '@/core/config/app-config';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { SharedTransition } from '@/ui/shared-transition';
import { Dialog, type DialogRef } from '@angular/cdk/dialog';
import { NgOptimizedImage } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  type TemplateRef,
  computed,
  inject,
  input,
  signal,
  viewChild,
} from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

const HERO_VISIBLE = 5;
const TOUR_VISIBLE = 3;

// Photos as a grid; any of them opens a lightbox. "hero" is the big mosaic
// at the top of a listing (first photo large, up to five, "+N" on the last);
// "tour" is the tour page's gallery (the cover large, two more beside it,
// "+N photos" on the last; the cover may fly in from the card, sharedName);
// "compact" is a plain grid (reviews, posts).
@Component({
  selector: 'app-photo-grid',
  imports: [NgOptimizedImage, SharedTransition, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (photos().length > 0) {
      @if (variant() === 'tour') {
        <div class="grid h-[20.625rem] gap-2" [class]="photos().length > 1 ? 'grid-cols-[2fr_1fr] grid-rows-2' : 'grid-cols-1'">
          @for (url of visible(); track url; let index = $index; let last = $last) {
            @if (index === 0 && sharedName(); as name) {
              <button type="button" class="relative row-span-2 overflow-hidden rounded-[0.875rem]" [attr.aria-label]="label(index)" [appSharedTransition]="name" [sharedAlways]="true" [sharedMedia]="sharedMedia()" (click)="open(index)">
                <img [ngSrc]="src(url)" fill priority sizes="(min-width: 1024px) 40vw, 100vw" [alt]="alt()" class="object-cover" />
              </button>
            } @else {
              <button type="button" class="relative overflow-hidden rounded-[0.875rem]" [class.row-span-2]="index === 0" [attr.aria-label]="label(index)" (click)="open(index)">
                <img [ngSrc]="src(url)" fill [priority]="index === 0" [sizes]="index === 0 ? '(min-width: 1024px) 40vw, 100vw' : '20vw'" [alt]="index === 0 ? alt() : ''" class="object-cover transition hover:scale-105" />
                @if (last && hidden() > 0) {
                  <span class="absolute inset-0 flex items-center justify-center gap-1.5 bg-black/45 font-semibold text-white">
                    <span class="icon-[heroicons--photo] size-5" aria-hidden="true"></span>{{ 'gallery.more' | transloco: { count: hidden() } }}
                  </span>
                }
              </button>
            }
          }
        </div>
      } @else if (variant() === 'hero') {
        <div class="grid h-64 grid-cols-4 grid-rows-2 gap-2 overflow-hidden rounded-card sm:h-96">
          @for (url of visible(); track url; let index = $index; let last = $last) {
            <button
              type="button"
              class="relative overflow-hidden"
              [class]="index === 0 ? 'col-span-4 row-span-2 sm:col-span-2' : 'hidden sm:block'"
              [attr.aria-label]="label(index)"
              (click)="open(index)"
            >
              <img
                [ngSrc]="src(url)"
                fill
                [priority]="index === 0"
                [sizes]="index === 0 ? '(min-width: 640px) 50vw, 100vw' : '25vw'"
                [alt]="index === 0 ? alt() : ''"
                class="object-cover transition hover:scale-105"
              />
              @if (last && hidden() > 0) {
                <span class="absolute inset-0 flex items-center justify-center bg-black/50 text-lg font-semibold text-white">
                  +{{ hidden() }}
                </span>
              }
            </button>
          }
        </div>
      } @else {
        <div class="grid grid-cols-3 gap-2 sm:grid-cols-4">
          @for (url of photos(); track url; let index = $index) {
            <button type="button" class="relative aspect-square overflow-hidden rounded-lg" [attr.aria-label]="label(index)" (click)="open(index)">
              <img [ngSrc]="src(url)" fill sizes="25vw" alt="" class="object-cover transition hover:scale-105" />
            </button>
          }
        </div>
      }
    }

    <ng-template #lightbox>
      <div
        class="relative flex h-full w-full items-center justify-center bg-black/95 p-4 text-white"
        tabindex="-1"
        (keydown.arrowright)="step(1)"
        (keydown.arrowleft)="step(-1)"
        (pointerdown)="swipeFrom = $event.clientX"
        (pointerup)="swipe($event.clientX)"
      >
        <button type="button" class="absolute top-4 right-4 flex size-10 items-center justify-center rounded-full bg-white/10 hover:bg-white/20" [attr.aria-label]="'gallery.close' | transloco" (click)="ref?.close()">
          <span class="icon-[heroicons--x-mark] size-6" aria-hidden="true"></span>
        </button>
        @if (photos().length > 1) {
          <button type="button" class="absolute left-2 flex size-12 items-center justify-center rounded-full bg-white/10 hover:bg-white/20" [attr.aria-label]="'gallery.previous' | transloco" (click)="step(-1)">
            <span class="icon-[heroicons--chevron-left] size-7" aria-hidden="true"></span>
          </button>
        }
        @for (i of [index()]; track i) {
          <img [src]="src(photos()[i] ?? '')" [alt]="alt()" class="motion-zoom-in max-h-[85vh] max-w-[90vw] touch-pan-y object-contain select-none" draggable="false" />
        }
        @if (photos().length > 1) {
          <button type="button" class="absolute right-2 flex size-12 items-center justify-center rounded-full bg-white/10 hover:bg-white/20" [attr.aria-label]="'gallery.next' | transloco" (click)="step(1)">
            <span class="icon-[heroicons--chevron-right] size-7" aria-hidden="true"></span>
          </button>
        }
        <span class="absolute bottom-4 text-sm text-white/80" aria-live="polite">{{ index() + 1 }} / {{ photos().length }}</span>
      </div>
    </ng-template>
  `,
})
export class PhotoGrid {
  readonly photos = input.required<string[]>();
  readonly alt = input('');
  readonly variant = input<'hero' | 'tour' | 'compact'>('compact');
  // The first photo's view-transition name ("tour" only), and the screens
  // it applies on.
  readonly sharedName = input<string | null>(null);
  readonly sharedMedia = input<string | null>(null);

  private readonly apiBaseUrl = inject(APP_CONFIG).apiBaseUrl;
  private readonly dialog = inject(Dialog);
  private readonly transloco = inject(TranslocoService);
  private readonly lightbox = viewChild.required<TemplateRef<unknown>>('lightbox');
  protected ref: DialogRef | null = null;
  protected readonly index = signal(0);

  protected readonly visible = computed(() => {
    const variant = this.variant();
    if (variant === 'compact') return this.photos();
    return this.photos().slice(0, variant === 'hero' ? HERO_VISIBLE : TOUR_VISIBLE);
  });
  protected readonly hidden = computed(() => this.photos().length - this.visible().length);

  protected src(url: string): string {
    return resolveMediaUrl(this.apiBaseUrl, url);
  }

  protected label(index: number): string {
    return `${this.alt() || this.transloco.translate('listing.photos.heading')} ${index + 1} / ${this.photos().length}`;
  }

  // Opens the lightbox at a photo (also from outside: "All photos").
  open(index: number): void {
    this.index.set(index);
    this.ref = this.dialog.open(this.lightbox(), {
      ariaLabel: this.alt() || this.transloco.translate('listing.photos.heading'),
      width: '100vw',
      height: '100vh',
      maxWidth: '100vw',
    });
    this.ref.closed.subscribe(() => {
      this.ref = null;
    });
  }

  protected swipeFrom: number | null = null;

  // A sideways swipe on the photo: the next or the previous one.
  protected swipe(x: number): void {
    const from = this.swipeFrom;
    this.swipeFrom = null;
    if (from === null || Math.abs(x - from) < 50 || this.photos().length < 2) return;
    this.step(x < from ? 1 : -1);
  }

  protected step(delta: number): void {
    const count = this.photos().length;
    this.index.update((index) => (index + delta + count) % count);
  }
}
