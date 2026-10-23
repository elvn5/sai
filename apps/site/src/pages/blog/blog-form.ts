import type { BlogDifficulty, BlogPost, BlogSeason } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import {
  BLOG_DIFFICULTIES,
  BLOG_DURATION_MAX,
  BLOG_SEASONS,
  type BadgeOption,
} from '@/core/lib/blog-options';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { slugify } from '@/core/lib/slugify';
import { FieldErrors } from '@/ui/field-errors';
import { type IconOption, IconPicker } from '@/ui/icon-picker';
import { ImageUploadField } from '@/ui/image-upload-field';
import { RichTextEditor } from '@/ui/rich-text-editor';
import { HttpClient } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  linkedSignal,
  signal,
} from '@angular/core';
import { FormField, form, maxLength, submit, validateStandardSchema } from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';
import { z } from 'zod/mini';

interface PostForm {
  title: string;
  excerpt: string;
  content: string;
  coverImageUrl: string | null;
  // '' when not set.
  difficulty: string;
  duration: string;
  season: string;
  isPublished: boolean;
}

function toForm(post: BlogPost | undefined): PostForm {
  return {
    title: post?.title ?? '',
    excerpt: post?.excerpt ?? '',
    content: post?.content ?? '',
    coverImageUrl: post?.coverImageUrl ?? null,
    difficulty: post?.difficulty ?? '',
    duration: post?.duration ?? '',
    season: post?.season ?? '',
    isPublished: post?.isPublished ?? true,
  };
}

// A partner's blog post: creates one, or edits `post`. After saving it
// opens the post, or the cabinet's listings for a draft.
@Component({
  selector: 'app-blog-form',
  imports: [FieldErrors, FormField, IconPicker, ImageUploadField, RichTextEditor, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <form class="mt-8 flex flex-col gap-6" novalidate (submit)="save($event)">
      <label class="ui-field">
        {{ 'blog.form.title' | transloco }}
        <input class="ui-input text-base" [formField]="postForm.title" />
        <app-field-errors [state]="postForm.title()" />
      </label>
      <label class="ui-field">
        {{ 'blog.form.excerpt' | transloco }}
        <input class="ui-input" [formField]="postForm.excerpt" />
        <app-field-errors [state]="postForm.excerpt()" />
      </label>
      <div class="ui-field">
        <span id="blog-cover-label">{{ 'blog.form.cover' | transloco }}</span>
        <app-image-upload-field labelId="blog-cover-label" [formField]="postForm.coverImageUrl" />
      </div>
      <div class="ui-field">
        <span>{{ 'blog.form.content' | transloco }}</span>
        <app-rich-text-editor [placeholder]="'blog.form.contentPlaceholder' | transloco" [formField]="postForm.content" />
        <app-field-errors [state]="postForm.content()" />
      </div>
      <fieldset class="flex flex-col gap-4">
        <legend class="mb-1 text-sm text-text-muted">{{ 'blog.form.badgesHint' | transloco }}</legend>
        <div class="ui-field">
          <span>{{ 'blog.badges.difficulty' | transloco }}</span>
          <app-icon-picker clearable [options]="difficultyOptions()" [formField]="postForm.difficulty" />
        </div>
        <label class="ui-field">
          {{ 'blog.badges.duration' | transloco }}
          <input class="ui-input max-w-sm" [placeholder]="'blog.form.durationPlaceholder' | transloco" [formField]="postForm.duration" />
          <app-field-errors [state]="postForm.duration()" />
        </label>
        <div class="ui-field">
          <span>{{ 'blog.badges.season' | transloco }}</span>
          <app-icon-picker clearable [options]="seasonOptions()" [formField]="postForm.season" />
        </div>
      </fieldset>
      <label class="flex items-center gap-3 text-sm text-text">
        <input type="checkbox" class="size-4 accent-brand-600" [formField]="postForm.isPublished" />
        {{ 'blog.form.publish' | transloco }}
      </label>
      @if (saveFailed()) {
        <p class="ui-error" role="alert">{{ 'blog.form.saveError' | transloco }}</p>
      }
      <button type="submit" class="ui-btn-primary py-2.5" [disabled]="postForm().submitting()">
        {{ (post() ? 'blog.form.save' : 'blog.form.create') | transloco }}
      </button>
    </form>
  `,
})
export class BlogForm {
  readonly post = input<BlogPost | undefined>(undefined);

  private readonly http = inject(HttpClient);
  private readonly lang = injectLang();
  private readonly transloco = inject(TranslocoService);
  private readonly navigate = injectSiteNavigate();
  protected readonly saveFailed = signal(false);
  protected readonly difficultyOptions = computed(() =>
    this.options(BLOG_DIFFICULTIES, 'blog.difficulty'),
  );
  protected readonly seasonOptions = computed(() => this.options(BLOG_SEASONS, 'blog.season'));

  private readonly model = linkedSignal(() => toForm(this.post()));
  protected readonly postForm = form(this.model, (path) => {
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
    maxLength(path.duration, BLOG_DURATION_MAX);
  });

  protected async save(event: Event): Promise<void> {
    event.preventDefault();
    this.saveFailed.set(false);
    await submit(this.postForm, async () => {
      const value = this.model();
      const body = {
        title: value.title.trim(),
        excerpt: value.excerpt.trim() || null,
        content: value.content,
        coverImageUrl: value.coverImageUrl,
        difficulty: (value.difficulty || null) as BlogDifficulty | null,
        duration: value.duration.trim() || null,
        season: (value.season || null) as BlogSeason | null,
        isPublished: value.isPublished,
      };
      const existing = this.post();
      try {
        const saved = existing
          ? await firstValueFrom(this.http.put<BlogPost>(`/blog/${existing.id}`, body))
          : await firstValueFrom(
              // slugify adds a random suffix, so titles can repeat.
              this.http.post<BlogPost>('/blog', {
                ...body,
                locale: this.lang(),
                slug: slugify(body.title),
              }),
            );
        await this.navigate(saved.isPublished ? `/blog/${saved.slug}` : '/partner/listings');
      } catch {
        this.saveFailed.set(true);
      }
      return undefined;
    });
  }

  private options(options: BadgeOption[], prefix: string): IconOption[] {
    this.lang();
    return options.map((option) => ({
      value: option.value,
      icon: option.icon,
      label: this.transloco.translate(`${prefix}.${option.value}`),
    }));
  }
}
