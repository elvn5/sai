import type { BlogPost } from '@/core/api/types';
import { bindPageMeta } from '@/core/seo/seo.service';
import { BlogForm } from '@/pages/blog/blog-form';
import { LoadError } from '@/ui/load-error';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

// /blog/edit/:id: a partner edits one of their own posts.
@Component({
  selector: 'app-blog-edit-page',
  imports: [BlogForm, LoadError, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="mx-auto max-w-3xl px-4 py-12">
      <h1 class="text-3xl font-extrabold text-text">{{ 'blog.form.editTitle' | transloco }}</h1>
      @if (mine.error()) {
        <div class="mt-8"><app-load-error (retry)="mine.reload()" /></div>
      } @else if (post(); as item) {
        <app-blog-form [post]="item" />
      } @else if (mine.hasValue()) {
        <p class="mt-8 text-text-muted">{{ 'blog.form.notFound' | transloco }}</p>
      }
    </section>
  `,
})
export class BlogEditPage {
  readonly id = input.required<string>();
  protected readonly mine = httpResource<BlogPost[]>(() => '/blog/mine');
  protected readonly post = computed(() =>
    this.mine.value()?.find((post) => post.id === this.id()),
  );

  constructor() {
    const transloco = inject(TranslocoService);
    bindPageMeta(() => ({
      title: transloco.translate('blog.form.editTitle'),
      description: transloco.translate('blog.metaDescription'),
      robots: 'noindex',
      alternates: null,
    }));
  }
}
