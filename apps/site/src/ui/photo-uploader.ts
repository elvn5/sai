import { IMAGE_TYPES, UploadApi } from '@/core/api/upload';
import { APP_CONFIG } from '@/core/config/app-config';
import { resolveMediaUrl } from '@/core/lib/media-url';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  model,
  signal,
} from '@angular/core';
import type { FormValueControl } from '@angular/forms/signals';
import { TranslocoPipe } from '@jsverse/transloco';

// Photos uploaded straight to storage as they're picked; the field holds
// their URLs. With `coverBadge`, the first is the cover and any other can be
// moved first.
@Component({
  selector: 'app-photo-uploader',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="grid grid-cols-3 gap-2 sm:grid-cols-4">
      @for (url of value(); track url; let index = $index) {
        <div class="relative aspect-square overflow-hidden rounded-lg border border-border">
          <img [src]="src(url)" alt="" class="size-full object-cover" />
          @if (coverBadge() && index === 0) {
            <span class="absolute top-1 left-1 rounded bg-black/60 px-1.5 py-0.5 text-xs text-white">{{ 'listing.photos.cover' | transloco }}</span>
          }
          <div class="absolute top-1 right-1 flex gap-1">
            @if (coverBadge() && index > 0) {
              <button type="button" class="flex size-7 items-center justify-center rounded-full bg-black/60 text-white hover:bg-black/80" [attr.aria-label]="'listing.photos.makeCover' | transloco" [title]="'listing.photos.makeCover' | transloco" (click)="makeFirst(index)">
                <span class="icon-[heroicons--star] size-4" aria-hidden="true"></span>
              </button>
            }
            <button type="button" class="flex size-7 items-center justify-center rounded-full bg-black/60 text-white hover:bg-black/80" [attr.aria-label]="'listing.photos.remove' | transloco" (click)="remove(index)">
              <span class="icon-[heroicons--x-mark] size-4" aria-hidden="true"></span>
            </button>
          </div>
        </div>
      }
      @for (n of uploadingSlots(); track n) {
        <div class="flex aspect-square items-center justify-center rounded-lg border border-dashed border-border" role="status">
          <span class="icon-[heroicons--arrow-path] size-6 animate-spin text-text-muted" aria-hidden="true"></span>
          <span class="sr-only">{{ 'upload.uploading' | transloco }}</span>
        </div>
      }
      @if (remaining() > 0) {
        <button type="button" class="flex aspect-square flex-col items-center justify-center gap-1 rounded-lg border border-dashed border-border text-text-muted transition-colors hover:border-brand-500 hover:text-brand-700" (click)="file.click()">
          <span class="icon-[heroicons--photo] size-6" aria-hidden="true"></span>
          <span class="text-xs">{{ 'listing.photos.add' | transloco }}</span>
        </button>
      }
    </div>
    <input #file type="file" accept="image/jpeg,image/png,image/webp" [multiple]="limit() > 1" class="hidden" (change)="onSelect($event)" />
    @if (error(); as key) {
      <p class="ui-error mt-2" role="alert">{{ key | transloco }}</p>
    }
  `,
})
export class PhotoUploader implements FormValueControl<string[]> {
  readonly value = model<string[]>([]);
  readonly limit = input.required<number>();
  // Storage folder ("hotels", "reviews"…).
  readonly folder = input.required<string>();
  readonly coverBadge = input(false);

  private readonly uploads = inject(UploadApi);
  private readonly apiBaseUrl = inject(APP_CONFIG).apiBaseUrl;
  protected readonly uploading = signal(0);
  protected readonly error = signal<string | null>(null);
  protected readonly remaining = computed(
    () => this.limit() - this.value().length - this.uploading(),
  );
  protected readonly uploadingSlots = computed(() =>
    Array.from({ length: this.uploading() }, (_, i) => i),
  );

  protected src(url: string): string {
    return resolveMediaUrl(this.apiBaseUrl, url);
  }

  protected async onSelect(event: Event): Promise<void> {
    const input = event.target as HTMLInputElement;
    const files = Array.from(input.files ?? []).slice(0, Math.max(this.remaining(), 0));
    input.value = '';
    this.error.set(null);
    this.uploading.update((n) => n + files.length);
    // In parallel; each photo is added when ready, so one failure keeps the rest.
    await Promise.all(
      files.map(async (file) => {
        try {
          if (!IMAGE_TYPES.has(file.type)) throw new Error('unsupported-file-type');
          const url = await this.uploads.uploadToStorage(file, this.folder());
          this.value.update((list) => [...list, url]);
        } catch (error) {
          this.error.set(
            error instanceof Error && error.message === 'unsupported-file-type'
              ? 'listing.photos.unsupportedFileType'
              : 'listing.photos.uploadError',
          );
        } finally {
          this.uploading.update((n) => n - 1);
        }
      }),
    );
  }

  protected remove(index: number): void {
    this.value.update((list) => list.filter((_, i) => i !== index));
  }

  protected makeFirst(index: number): void {
    this.value.update((list) => {
      const next = [...list];
      const [photo] = next.splice(index, 1);
      return photo ? [photo, ...next] : list;
    });
  }
}
