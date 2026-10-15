import { UploadApi } from '@/core/api/upload';
import { ConfigStore } from '@/core/config/app-config';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { ChangeDetectionStrategy, Component, inject, model, signal } from '@angular/core';
import type { FormValueControl } from '@angular/forms/signals';

// An image picked and uploaded right away; the field holds its URL.
@Component({
  selector: 'app-image-upload-field',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex items-center gap-3">
      @if (value()) {
        <img [src]="preview()" alt="" class="size-20 rounded border border-border object-cover" />
      } @else {
        <div class="flex size-20 items-center justify-center rounded border border-dashed border-border text-text-muted">
          <span class="icon-[heroicons--photo] size-6" aria-hidden="true"></span>
        </div>
      }
      <div class="flex flex-col gap-1">
        <button type="button" class="ui-btn-soft ui-btn-xs" [disabled]="uploading()" (click)="file.click()">
          {{ uploading() ? 'Загрузка…' : value() ? 'Заменить' : 'Загрузить' }}
        </button>
        @if (value()) {
          <button type="button" class="ui-btn-danger ui-btn-xs" (click)="value.set(null)">Удалить</button>
        }
      </div>
      <input #file type="file" accept="image/*" class="hidden" (change)="onSelect($event)" />
    </div>
    @if (error()) {
      <p class="ui-error mt-1">{{ error() }}</p>
    }
  `,
})
export class ImageUploadField implements FormValueControl<string | null> {
  readonly value = model<string | null>(null);

  private readonly uploads = inject(UploadApi);
  private readonly apiBaseUrl = inject(ConfigStore).config.apiBaseUrl;
  protected readonly uploading = signal(false);
  protected readonly error = signal<string | null>(null);

  protected preview(): string {
    return resolveMediaUrl(this.apiBaseUrl, this.value());
  }

  protected async onSelect(event: Event): Promise<void> {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    this.uploading.set(true);
    this.error.set(null);
    try {
      this.value.set(await this.uploads.uploadImage(file));
    } catch {
      this.error.set('Не удалось загрузить изображение');
    } finally {
      this.uploading.set(false);
      input.value = '';
    }
  }
}
