import {
  type BlogDifficulty,
  type BlogPost,
  type BlogSeason,
  LOCALE_OPTIONS,
  Locale,
} from '@/core/api/types';
import { ConfigStore } from '@/core/config/app-config';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { slugify } from '@/core/lib/slugify';
import { ImageUploadField } from '@/ui/image-upload-field';
import { RichTextEditor } from '@/ui/rich-text-editor';
import { Dialog, type DialogRef } from '@angular/cdk/dialog';
import { HttpClient } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  type TemplateRef,
  computed,
  effect,
  inject,
  input,
  signal,
  untracked,
  viewChild,
} from '@angular/core';
import { FormField, form, maxLength, submit } from '@angular/forms/signals';
import { Router, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';

interface PostForm {
  locale: Locale;
  slug: string;
  title: string;
  excerpt: string;
  content: string;
  coverImageUrl: string | null;
  seoTitle: string;
  seoDescription: string;
  // '' when not set.
  difficulty: BlogDifficulty | '';
  duration: string;
  season: BlogSeason | '';
  isPublished: boolean;
}

// The badges on the post's card on the site.
const DIFFICULTY_OPTIONS: { value: BlogDifficulty; label: string }[] = [
  { value: 'any', label: 'Любой уровень' },
  { value: 'easy', label: 'Лёгкий' },
  { value: 'moderate', label: 'Средний' },
  { value: 'hard', label: 'Сложный' },
  { value: 'expert', label: 'Экспертный' },
];
const SEASON_OPTIONS: { value: BlogSeason; label: string }[] = [
  { value: 'winter', label: 'Зима' },
  { value: 'spring', label: 'Весна' },
  { value: 'summer', label: 'Лето' },
  { value: 'autumn', label: 'Осень' },
  { value: 'all_year', label: 'Круглый год' },
];

// Writes or edits a blog post (services/content); /blog/new creates one.
@Component({
  selector: 'app-blog-edit-page',
  imports: [FormField, ImageUploadField, RichTextEditor, RouterLink],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <form class="mx-auto max-w-3xl" (submit)="save($event)" novalidate>
      <div class="mb-4 flex items-center justify-between">
        <h1 class="text-xl font-semibold text-text">{{ isNew() ? 'Новая статья' : 'Редактирование статьи' }}</h1>
        <div class="flex gap-2">
          <a routerLink="/blog" class="ui-btn-ghost">Назад</a>
          <button type="button" class="ui-btn-ghost" (click)="openPreview()">
            <span class="icon-[heroicons--eye] size-4" aria-hidden="true"></span>Предпросмотр
          </button>
          @if (!isNew()) {
            <button type="button" class="ui-btn-danger" (click)="remove()">Удалить</button>
          }
          <button type="submit" class="ui-btn-primary" [disabled]="postForm().submitting()">Сохранить</button>
        </div>
      </div>

      @if (loaded()) {
        <div class="flex flex-col gap-4">
          <div class="grid grid-cols-2 gap-4">
            <label class="ui-field">
              Язык
              <select class="ui-input" [formField]="postForm.locale">
                @for (option of localeOptions; track option.value) {
                  <option [value]="option.value">{{ option.label }}</option>
                }
              </select>
            </label>
            <label class="ui-field">
              Slug (для URL)
              <input class="ui-input" [formField]="postForm.slug" (input)="slugTouched = true" />
            </label>
          </div>
          <label class="ui-field">Заголовок <input class="ui-input" [formField]="postForm.title" /></label>
          <label class="ui-field">
            Краткое описание (анонс)
            <textarea class="ui-input" rows="2" [formField]="postForm.excerpt"></textarea>
          </label>
          <div class="grid grid-cols-3 gap-4">
            <label class="ui-field">
              Сложность
              <select class="ui-input" [formField]="postForm.difficulty">
                <option value="">Не указана</option>
                @for (option of difficultyOptions; track option.value) {
                  <option [value]="option.value">{{ option.label }}</option>
                }
              </select>
            </label>
            <label class="ui-field">
              Длительность
              <input class="ui-input" placeholder="Например, 2–5 часов" [formField]="postForm.duration" />
            </label>
            <label class="ui-field">
              Сезон
              <select class="ui-input" [formField]="postForm.season">
                <option value="">Не указан</option>
                @for (option of seasonOptions; track option.value) {
                  <option [value]="option.value">{{ option.label }}</option>
                }
              </select>
            </label>
          </div>
          <span class="ui-hint -mt-2">Бейджи на карточке статьи; пустые не показываются</span>
          <div class="ui-field">Обложка <app-image-upload-field [formField]="postForm.coverImageUrl" /></div>
          <div class="ui-field">Содержание <app-rich-text-editor [formField]="postForm.content" /></div>

          <p class="mt-2 border-t border-border pt-4 text-sm font-semibold text-text-muted">SEO</p>
          <label class="ui-field">
            SEO заголовок
            <input class="ui-input" [formField]="postForm.seoTitle" />
            <span class="ui-hint">Если пусто — используется заголовок статьи</span>
          </label>
          <label class="ui-field">
            SEO описание
            <textarea class="ui-input" rows="2" [formField]="postForm.seoDescription"></textarea>
            <span class="ui-hint">Если пусто — используется краткое описание</span>
          </label>
          <label class="flex items-center gap-2 text-sm text-text">
            <input type="checkbox" class="size-4 accent-brand-600" [formField]="postForm.isPublished" />
            Опубликовано (видно на сайте)
          </label>
          @if (saveError()) {
            <p class="ui-error">{{ saveError() }}</p>
          }
        </div>
      }
    </form>

    <ng-template #preview>
      <div class="ui-dialog" role="dialog" aria-label="Предпросмотр статьи">
        @if (model().coverImageUrl) {
          <img [src]="coverUrl()" [alt]="model().title" class="mb-6 aspect-video w-full rounded-lg object-cover" />
        }
        <h2 class="text-2xl font-bold text-text">{{ model().title || 'Без названия' }}</h2>
        @if (model().excerpt) {
          <p class="mt-2 text-text-muted">{{ model().excerpt }}</p>
        }
        <div class="prose prose-sm mt-6 max-w-none" [innerHTML]="model().content"></div>
        <div class="mt-6 flex justify-end">
          <button type="button" class="ui-btn-soft" (click)="previewRef?.close()">Закрыть</button>
        </div>
      </div>
    </ng-template>
  `,
})
export class BlogEditPage {
  // Route param: a post id, or "new".
  readonly id = input.required<string>();

  private readonly http = inject(HttpClient);
  private readonly router = inject(Router);
  private readonly dialog = inject(Dialog);
  private readonly apiBaseUrl = inject(ConfigStore).config.apiBaseUrl;
  private readonly previewTemplate = viewChild.required<TemplateRef<unknown>>('preview');
  protected previewRef: DialogRef | null = null;

  protected readonly localeOptions = LOCALE_OPTIONS;
  protected readonly difficultyOptions = DIFFICULTY_OPTIONS;
  protected readonly seasonOptions = SEASON_OPTIONS;
  protected readonly loaded = signal(false);
  protected readonly saveError = signal<string | null>(null);
  protected slugTouched = false;

  protected readonly model = signal<PostForm>({
    locale: Locale.RU,
    slug: '',
    title: '',
    excerpt: '',
    content: '',
    coverImageUrl: null,
    seoTitle: '',
    seoDescription: '',
    difficulty: '',
    duration: '',
    season: '',
    isPublished: false,
  });
  // The API's limit.
  protected readonly postForm = form(this.model, (path) => maxLength(path.duration, 40));
  private readonly title = computed(() => this.model().title);

  constructor() {
    effect(() => {
      const id = this.id();
      untracked(() => void this.load(id));
    });
    // A new post's slug follows its title until edited by hand.
    effect(() => {
      const slug = slugify(this.title());
      untracked(() => {
        if (this.isNew() && !this.slugTouched && this.model().slug !== slug) {
          this.model.update((value) => ({ ...value, slug }));
        }
      });
    });
  }

  protected isNew(): boolean {
    return this.id() === 'new';
  }

  protected coverUrl(): string {
    return resolveMediaUrl(this.apiBaseUrl, this.model().coverImageUrl);
  }

  private async load(id: string): Promise<void> {
    if (id === 'new') {
      this.loaded.set(true);
      return;
    }
    const post = await firstValueFrom(this.http.get<BlogPost>(`/blog/admin/${id}`));
    this.model.set({
      locale: post.locale,
      slug: post.slug,
      title: post.title,
      excerpt: post.excerpt ?? '',
      content: post.content,
      coverImageUrl: post.coverImageUrl,
      seoTitle: post.seoTitle ?? '',
      seoDescription: post.seoDescription ?? '',
      difficulty: post.difficulty ?? '',
      duration: post.duration ?? '',
      season: post.season ?? '',
      isPublished: post.isPublished,
    });
    this.loaded.set(true);
  }

  protected openPreview(): void {
    this.previewRef = this.dialog.open(this.previewTemplate());
  }

  protected async save(event: Event): Promise<void> {
    event.preventDefault();
    this.saveError.set(null);
    await submit(this.postForm, async () => {
      const value = this.model();
      const payload = {
        locale: value.locale,
        slug: value.slug,
        title: value.title,
        excerpt: value.excerpt || null,
        content: value.content,
        coverImageUrl: value.coverImageUrl,
        seoTitle: value.seoTitle || null,
        seoDescription: value.seoDescription || null,
        difficulty: value.difficulty || null,
        duration: value.duration.trim() || null,
        season: value.season || null,
        isPublished: value.isPublished,
      };
      try {
        if (this.isNew()) {
          const created = await firstValueFrom(this.http.post<{ id: string }>('/blog', payload));
          await this.router.navigate(['/blog', created.id]);
        } else {
          await firstValueFrom(this.http.put(`/blog/${this.id()}`, payload));
          await this.router.navigateByUrl('/blog');
        }
      } catch {
        this.saveError.set('Не удалось сохранить статью');
      }
      return undefined;
    });
  }

  protected async remove(): Promise<void> {
    if (this.isNew() || !window.confirm('Удалить эту статью?')) return;
    await firstValueFrom(this.http.delete(`/blog/${this.id()}`));
    await this.router.navigateByUrl('/blog');
  }
}
