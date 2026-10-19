import { type Guide, type GuideAttachment, GuideDifficulty, GuideSeason } from '@/core/api/types';
import { UploadApi } from '@/core/api/upload';
import { injectLang } from '@/core/i18n/current-lang';
import { fileSize } from '@/core/lib/format';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { slugify } from '@/core/lib/slugify';
import { bindPageMeta } from '@/core/seo/seo.service';
import { DIFFICULTY_ICONS, EQUIPMENT_CATALOG, SEASON_ICONS } from '@/pages/guides/guide-options';
import { FieldErrors } from '@/ui/field-errors';
import { IconMultiPicker, type IconOption, IconPicker } from '@/ui/icon-picker';
import { ImageUploadField } from '@/ui/image-upload-field';
import { RichTextEditor } from '@/ui/rich-text-editor';
import { HttpClient } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormField, form, maxLength, submit, validateStandardSchema } from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';
import { z } from 'zod/mini';

interface GuideForm {
  title: string;
  excerpt: string;
  content: string;
  coverImageUrl: string | null;
  difficulty: GuideDifficulty;
  season: GuideSeason;
  equipment: string[];
}

// /guides/new: an approved partner publishes a guide (browser only).
@Component({
  selector: 'app-guide-new-page',
  imports: [
    FieldErrors,
    FormField,
    IconMultiPicker,
    IconPicker,
    ImageUploadField,
    RichTextEditor,
    TranslocoPipe,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="mx-auto max-w-5xl px-4 py-12">
      <h1 class="text-3xl font-extrabold text-text">{{ 'guides.newTitle' | transloco }}</h1>

      <form class="mt-8 grid grid-cols-1 gap-8 lg:grid-cols-[280px_1fr]" novalidate (submit)="save($event)">
        <aside class="ui-card order-2 flex h-fit flex-col gap-6 p-4 lg:sticky lg:top-24 lg:order-1">
          <div class="ui-field">
            <span>{{ 'guides.difficultyLabel' | transloco }}</span>
            <app-icon-picker [options]="difficultyOptions()" [formField]="guideForm.difficulty" />
          </div>
          <div class="ui-field">
            <span>{{ 'guides.seasonLabel' | transloco }}</span>
            <app-icon-picker [options]="seasonOptions()" [formField]="guideForm.season" />
          </div>
          <div class="ui-field">
            <span>{{ 'guides.equipmentLabel' | transloco }}</span>
            <span class="ui-hint">{{ 'guides.equipmentHint' | transloco }}</span>
            <app-icon-multi-picker [options]="equipmentOptions()" [formField]="guideForm.equipment" />
          </div>
        </aside>

        <div class="order-1 flex flex-col gap-6 lg:order-2">
          <label class="ui-field">
            {{ 'guides.titleLabel' | transloco }}
            <input class="ui-input text-base" [formField]="guideForm.title" />
            <app-field-errors [state]="guideForm.title()" />
          </label>
          <label class="ui-field">
            {{ 'guides.excerptLabel' | transloco }}
            <input class="ui-input" [formField]="guideForm.excerpt" />
            <app-field-errors [state]="guideForm.excerpt()" />
          </label>
          <div class="ui-field">
            <span id="guide-cover-label">{{ 'guides.coverLabel' | transloco }}</span>
            <app-image-upload-field labelId="guide-cover-label" [formField]="guideForm.coverImageUrl" />
          </div>
          <div class="ui-field">
            <span>{{ 'guides.contentLabel' | transloco }}</span>
            <app-rich-text-editor [placeholder]="'guides.contentPlaceholder' | transloco" [formField]="guideForm.content" />
            <app-field-errors [state]="guideForm.content()" />
          </div>
          <div class="ui-field">
            <label for="guide-files">{{ 'guides.filesLabel' | transloco }}</label>
            <span class="ui-hint">{{ 'guides.filesHint' | transloco }}</span>
            <input
              id="guide-files"
              type="file"
              multiple
              accept=".pdf,.gpx,.kml,.zip,.xml"
              class="ui-file"
              [disabled]="uploadingFiles()"
              (change)="addFiles($event)"
            />
            @if (uploadingFiles()) {
              <p class="ui-hint" role="status">{{ 'upload.uploading' | transloco }}</p>
            }
            @if (attachments().length > 0) {
              <ul class="mt-2 flex flex-col gap-2">
                @for (file of attachments(); track file.url; let index = $index) {
                  <li class="ui-card flex items-center justify-between px-3 py-2 text-sm">
                    <span class="flex items-center gap-2 text-text">
                      <span class="icon-[heroicons--document] size-4 shrink-0 text-text-muted" aria-hidden="true"></span>
                      {{ file.name }}
                      <span class="text-text-muted">({{ size(file.size) }})</span>
                    </span>
                    <button
                      type="button"
                      class="ui-btn-ghost ui-btn-xs"
                      [attr.aria-label]="('guides.removeFile' | transloco) + ': ' + file.name"
                      (click)="removeFile(index)"
                    >
                      <span class="icon-[heroicons--x-mark] size-4" aria-hidden="true"></span>
                    </button>
                  </li>
                }
              </ul>
            }
          </div>

          @if (saveFailed()) {
            <p class="ui-error" role="alert">{{ 'guides.createError' | transloco }}</p>
          }
          <button type="submit" class="ui-btn-primary py-2.5" [disabled]="guideForm().submitting() || uploadingFiles()">
            {{ 'guides.submit' | transloco }}
          </button>
        </div>
      </form>
    </section>
  `,
})
export class GuideNewPage {
  private readonly http = inject(HttpClient);
  private readonly uploads = inject(UploadApi);
  private readonly transloco = inject(TranslocoService);
  private readonly navigate = injectSiteNavigate();
  private readonly lang = injectLang();

  protected readonly attachments = signal<GuideAttachment[]>([]);
  protected readonly uploadingFiles = signal(false);
  protected readonly saveFailed = signal(false);
  protected readonly size = fileSize;

  private readonly model = signal<GuideForm>({
    title: '',
    excerpt: '',
    content: '',
    coverImageUrl: null,
    difficulty: GuideDifficulty.EASY,
    season: GuideSeason.ALL_YEAR,
    equipment: [],
  });
  protected readonly guideForm = form(this.model, (path) => {
    const required = this.transloco.translate('login.required');
    validateStandardSchema(
      path,
      z.object({
        title: z.string().check(z.trim(), z.minLength(1, required)),
        content: z
          .string()
          .check(z.refine((html) => html.replace(/<[^>]*>/g, '').trim() !== '', required)),
      }),
    );
    // The API's limits.
    maxLength(path.title, 255);
    maxLength(path.excerpt, 500);
    maxLength(path.content, 20000);
  });

  // Options read the language at render: labels follow the switcher.
  protected readonly difficultyOptions = computed((): IconOption[] => {
    this.lang();
    return Object.values(GuideDifficulty).map((value) => ({
      value,
      icon: DIFFICULTY_ICONS[value],
      label: this.transloco.translate(`guides.difficulty.${value}`),
    }));
  });
  protected readonly seasonOptions = computed((): IconOption[] => {
    this.lang();
    return Object.values(GuideSeason).map((value) => ({
      value,
      icon: SEASON_ICONS[value],
      label: this.transloco.translate(`guides.season.${value}`),
    }));
  });
  protected readonly equipmentOptions = computed((): IconOption[] => {
    this.lang();
    return EQUIPMENT_CATALOG.map(({ key, icon }) => ({
      value: key,
      icon,
      label: this.transloco.translate(`guides.equipmentItems.${key}`),
    }));
  });

  constructor() {
    bindPageMeta(() => ({
      title: this.transloco.translate('guides.newTitle'),
      description: this.transloco.translate('guides.metaDescription'),
      robots: 'noindex',
      alternates: null,
    }));
  }

  protected async addFiles(event: Event): Promise<void> {
    const input = event.target as HTMLInputElement;
    const files = Array.from(input.files ?? []);
    if (files.length === 0) return;
    this.uploadingFiles.set(true);
    this.saveFailed.set(false);
    try {
      for (const file of files) {
        const uploaded = await this.uploads.uploadFile(file);
        this.attachments.update((list) => [...list, uploaded]);
      }
    } catch {
      this.saveFailed.set(true);
    } finally {
      this.uploadingFiles.set(false);
      input.value = '';
    }
  }

  protected removeFile(index: number): void {
    this.attachments.update((list) => list.filter((_, i) => i !== index));
  }

  protected async save(event: Event): Promise<void> {
    event.preventDefault();
    this.saveFailed.set(false);
    await submit(this.guideForm, async () => {
      const value = this.model();
      const title = value.title.trim();
      try {
        const guide = await firstValueFrom(
          this.http.post<Guide>('/guides', {
            locale: this.lang(),
            slug: slugify(title),
            title,
            excerpt: value.excerpt.trim() || null,
            content: value.content,
            difficulty: value.difficulty,
            season: value.season,
            equipmentChecklist: value.equipment,
            coverImageUrl: value.coverImageUrl,
            attachments: this.attachments(),
          }),
        );
        await this.navigate(`/guides/${guide.slug}`);
      } catch {
        this.saveFailed.set(true);
      }
      return undefined;
    });
  }
}
