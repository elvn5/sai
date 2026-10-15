import type { BlogPost } from '@/core/api/types';
import { injectLang } from '@/core/i18n/current-lang';
import { injectSiteNavigate } from '@/core/lib/site-navigate';
import { slugify } from '@/core/lib/slugify';
import { FieldErrors } from '@/ui/field-errors';
import { ImageUploadField } from '@/ui/image-upload-field';
import { RichTextEditor } from '@/ui/rich-text-editor';
import { HttpClient } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
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
  isPublished: boolean;
}

function toForm(post: BlogPost | undefined): PostForm {
  return {
    title: post?.title ?? '',
    excerpt: post?.excerpt ?? '',
    content: post?.content ?? '',
    coverImageUrl: post?.coverImageUrl ?? null,
    isPublished: post?.isPublished ?? true,
  };
}

// A partner's blog post: creates one, or edits `post`. After saving it
// opens the post, or the cabinet's listings for a draft.
@Component({
  selector: 'app-blog-form',
  imports: [FieldErrors, FormField, ImageUploadField, RichTextEditor, TranslocoPipe],
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
}
