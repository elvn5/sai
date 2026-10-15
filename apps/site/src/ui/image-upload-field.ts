import { UploadApi } from '@/core/api/upload';
import { APP_CONFIG } from '@/core/config/app-config';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { ChangeDetectionStrategy, Component, inject, input, model, signal } from '@angular/core';
import type { FormValueControl } from '@angular/forms/signals';
import { TranslocoPipe } from '@jsverse/transloco';

// An image picked and uploaded right away (a cover); the field holds its URL.
@Component({
  selector: 'app-image-upload-field',
  imports: [TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-3">
      @if (value()) {
        <img [src]="preview()" alt="" class="h-48 w-full rounded-lg border border-border object-cover" />
      }
      <div class="flex flex-wrap items-center gap-2">
        <button type="button" class="ui-btn-soft" [disabled]="uploading()" [attr.aria-describedby]="labelId()" (click)="file.click()">
          <span class="icon-[heroicons--photo] size-4" aria-hidden="true"></span>
          {{ (uploading() ? 'upload.uploading' : value() ? 'upload.replace' : 'upload.choose') | transloco }}
        </button>
        @if (value()) {
          <button type="button" class="ui-btn-danger" (click)="value.set(null)">{{ 'upload.remove' | transloco }}</button>
        }
      </div>
      <input #file type="file" accept="image/jpeg,image/png,image/webp" class="hidden" (change)="onSelect($event)" />
      @if (failed()) {
        <p class="ui-error" role="alert">{{ 'upload.failed' | transloco }}</p>
      }
    </div>
  `,
})
export class ImageUploadField implements FormValueControl<string | null> {
  readonly value = model<string | null>(null);
  // The id of the field's visible label, for the button's description.
  readonly labelId = input<string | null>(null);

  private readonly uploads = inject(UploadApi);
  private readonly apiBaseUrl = inject(APP_CONFIG).apiBaseUrl;
  protected readonly uploading = signal(false);
  protected readonly failed = signal(false);

  protected preview(): string {
    return resolveMediaUrl(this.apiBaseUrl, this.value());
  }

  protected async onSelect(event: Event): Promise<void> {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    this.uploading.set(true);
    this.failed.set(false);
    try {
      this.value.set(await this.uploads.uploadImage(file));
    } catch {
      this.failed.set(true);
    } finally {
      this.uploading.set(false);
      input.value = '';
    }
  }
}
