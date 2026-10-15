import { type BlogPost, LOCALE_OPTIONS, type Locale } from '@/core/api/types';
import { HttpClient, httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';

@Component({
  selector: 'app-blog-list-page',
  imports: [RouterLink],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="mb-4 flex items-center justify-between">
      <h1 class="text-xl font-semibold text-text">Блог</h1>
      <div class="flex items-center gap-2">
        <select class="ui-input w-40" aria-label="Язык" [value]="locale()" (change)="locale.set($any($event.target).value)">
          <option value="">Все языки</option>
          @for (option of localeOptions; track option.value) {
            <option [value]="option.value">{{ option.label }}</option>
          }
        </select>
        <a routerLink="/blog/new" class="ui-btn-primary">Новая статья</a>
      </div>
    </div>

    <div class="ui-table-wrap">
      <table class="ui-table">
        <thead>
          <tr>
            <th>Заголовок</th>
            <th>Язык</th>
            <th>Slug</th>
            <th>Статус</th>
            <th><span class="sr-only">Действия</span></th>
          </tr>
        </thead>
        <tbody>
          @for (post of posts.value() ?? []; track post.id) {
            <tr>
              <td>
                <a [routerLink]="['/blog', post.id]" class="hover:underline">{{ post.title }}</a>
                @if (post.userId) {
                  <span class="ui-badge ml-2 bg-sky-100 text-sky-800">партнёр</span>
                }
              </td>
              <td class="text-text-muted">{{ localeLabel(post.locale) }}</td>
              <td class="text-text-muted">{{ post.slug }}</td>
              <td>
                <span class="ui-badge" [class]="post.isPublished ? 'bg-brand-500/10 text-brand-700' : 'bg-text/5 text-text-muted'">
                  {{ post.isPublished ? 'Опубликовано' : 'Черновик' }}
                </span>
              </td>
              <td class="whitespace-nowrap text-right">
                <a [routerLink]="['/blog', post.id]" class="ui-btn-ghost ui-btn-xs">Изменить</a>
                <button type="button" class="ui-btn-danger ui-btn-xs" (click)="remove(post)">Удалить</button>
              </td>
            </tr>
          } @empty {
            @if (!posts.isLoading()) {
              <tr><td colspan="5" class="ui-empty">Статей пока нет</td></tr>
            }
          }
        </tbody>
      </table>
    </div>
  `,
})
export class BlogListPage {
  private readonly http = inject(HttpClient);
  protected readonly localeOptions = LOCALE_OPTIONS;
  protected readonly locale = signal<Locale | ''>('');
  protected readonly posts = httpResource<BlogPost[]>(
    () => `/blog/admin/all${this.locale() ? `?locale=${this.locale()}` : ''}`,
  );

  protected localeLabel(locale: Locale): string {
    return LOCALE_OPTIONS.find((option) => option.value === locale)?.label ?? locale;
  }

  protected async remove(post: BlogPost): Promise<void> {
    if (!window.confirm(`Удалить статью "${post.title}"?`)) return;
    await firstValueFrom(this.http.delete(`/blog/${post.id}`));
    this.posts.reload();
  }
}
